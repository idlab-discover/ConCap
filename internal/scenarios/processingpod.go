package scenarios

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	kubeapi "github.com/idlab-discover/concap/internal/kubernetes"
	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
)

type ProcessingPod struct {
	Name           string `yaml:"name"`
	ContainerImage string `yaml:"containerImage"`
	Command        string `yaml:"command"`
	CPURequest     string `yaml:"cpuRequest"`
	MemRequest     string `yaml:"memRequest"`
}

// ReadProcessingPod will unmarshall the yaml into the in-memory ProcessingPod representation
func ReadProcessingPod(filePath string) (*ProcessingPod, error) {
	pod := ProcessingPod{}

	fileHandler, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open processing pod file %s: %w", filePath, err)
	}
	defer fileHandler.Close()

	b, err := io.ReadAll(fileHandler)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML: %w", err)
	}

	err = yaml.UnmarshalStrict(b, &pod)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	// Default resource requests to help K8s with scheduling
	if pod.CPURequest == "" {
		pod.CPURequest = "100m"
	}
	if pod.MemRequest == "" {
		pod.MemRequest = "250Mi"
	}

	return &pod, nil
}

func (p *ProcessingPod) ProcessPcap(ctx context.Context, filePath string, scenarioName string, targetName string, outputDir string, labels map[string]string) error {
	inputFileContainer := filepath.Join("/data/input", scenarioName+"-"+targetName+".pcap")
	outputFileContainer := filepath.Join("/data/output", scenarioName+"-"+targetName+".csv")
	outputFileDownload := filepath.Join(outputDir, p.Name+".csv")
	outputLogFile := filepath.Join(outputDir, p.Name+".log")

	// Copy the pcap file to the pod
	err := kubeapi.CopyFileToPod(ctx, p.Name, p.Name, filePath, inputFileContainer)
	if err != nil {
		return fmt.Errorf("error uploading pcap file to pod: %w", err)
	}

	// Execute the processing command in the processing pod
	envVars := make(map[string]string)
	envVars["INPUT_FILE"] = inputFileContainer
	envVars["INPUT_FILE_NAME"] = scenarioName + "-" + targetName
	envVars["OUTPUT_FILE"] = outputFileContainer
	log.Println("Analyzing traffic using pod: ", p.Name)
	stdo, stde, err := kubeapi.ExecShellInContainerWithEnvVars(ctx, apiv1.NamespaceDefault, p.Name, p.Name, p.Command, envVars)
	if err != nil {
		log.Printf("stdout: %s\nstderr: %s", stdo, stde)
		return fmt.Errorf("error analyzing traffic: %w", err)
	}
	// Print the output of the processing command to log file
	if err := writeAnalysisLog(outputLogFile, stdo, stde); err != nil {
		return err
	}

	// Download the output file from the pod
	err = kubeapi.CopyFileFromPod(ctx, p.Name, p.Name, outputFileContainer, outputFileDownload, false)
	if err != nil {
		return fmt.Errorf("error downloading output file from pod: %w", err)
	}

	// Add labels to the output file
	// Extract the headers and values from the scenario file
	keys := make([]string, 0, len(labels))
	values := make([]string, 0, len(labels))

	for key, value := range labels {
		keys = append(keys, key)
		values = append(values, value)
	}
	err = p.AddColumnsToCSV(outputFileDownload, keys, values, true)
	if err != nil {
		return fmt.Errorf("error adding labels to output file: %w", err)
	}

	return nil
}

func (p *ProcessingPod) DeployPod(ctx context.Context) error {
	exists, err := kubeapi.PodExists(ctx, p.Name)
	if err != nil {
		return fmt.Errorf("check whether pod %s exists: %w", p.Name, err)
	}
	if !exists {
		log.Printf("Creating Pod %s\n", p.Name)
		podSpec := ProcessingPodSpec(p)
		_, err = kubeapi.CreateReadyPod(ctx, podSpec)
		if err != nil {
			return fmt.Errorf("create processing pod %s: %w", p.Name, err)
		}
		log.Printf("Processing pod %s created\n", p.Name)
	} else {
		log.Printf("Processing pod %s already exists\n", p.Name)
	}
	return nil
}

// AddLabelsToFlowsCSV adds the labels of the scenario to the flows CSV file.
// The labels are stored in a map. The keys are added to the header of the CSV file if addHeader is true, and the values are added to the rows.
//
// Parameters:
// - filepath: the path to the CSV file
// - addHeader: whether to add the labels to the header of the CSV file
//
// Returns an error if the file cannot be opened, read, or written to.
func (p *ProcessingPod) AddColumnsToCSV(filepath string, headers []string, values []string, addHeader bool) error {
	// Open the CSV file
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}

	// Read the CSV file
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) == 0 {
		file.Close()
		return fmt.Errorf("csv file is empty")
	}

	// Close the original file to allow reopening it in write mode
	if err := file.Close(); err != nil {
		return fmt.Errorf("close CSV file after reading: %w", err)
	}

	// if addHeader is true, add the headers to the first row
	i := 0
	if addHeader {
		records[0] = append(records[0], headers...)
		i = 1
	}

	// Add the labels to the rest of the rows
	for ; i < len(records); i++ {
		records[i] = append(records[i], values...)
	}

	// Reopen the file in write mode
	file, err = os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("reopen CSV file for writing: %w", err)
	}
	defer file.Close()

	// Write the updated records back to the original file
	writer := csv.NewWriter(file)
	if err := writer.WriteAll(records); err != nil {
		return fmt.Errorf("write CSV file: %w", err)
	}

	// Close the writer to ensure all data is flushed
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV file: %w", err)
	}
	return nil
}

func writeAnalysisLog(outputPath, stdout, stderr string) error {
	logFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create traffic analysis log file: %w", err)
	}

	if _, err := fmt.Fprintf(logFile, "stdout:\n%s\nstderr:\n%s\n", stdout, stderr); err != nil {
		logFile.Close()
		return fmt.Errorf("write traffic analysis log file: %w", err)
	}

	if err := logFile.Close(); err != nil {
		return fmt.Errorf("close traffic analysis log file: %w", err)
	}

	return nil
}
