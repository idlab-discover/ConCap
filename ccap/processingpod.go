package ccap

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
)

type ProcessingPod struct {
	Name           string `yaml:"name"`
	ContainerImage string `yaml:"containerImage"`
	Command        string `yaml:"command"`
}

// ReadProcessingPod will unmarshall the yaml into the in-memory ProcessingPod representation
func ReadProcessingPod(filePath string) (*ProcessingPod, error) {
	pod := ProcessingPod{}

	fileHandler, err := os.Open(filePath)
	if err != nil {
		log.Println("Failed to open file " + filePath + " : " + err.Error())
		return nil, err
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

	return &pod, nil
}

func (p *ProcessingPod) ProcessPcap(filePath string, scn *scenario.Scenario) error {
	inputFileContainer := filepath.Join("/data/input", scn.Name+".pcap")
	outputFileContainer := filepath.Join("/data/output", p.Name+".csv")
	outputFileDownload := filepath.Join(scn.OutputDir, p.Name+".csv")
	outputLogFile := filepath.Join(scn.OutputDir, p.Name+".log")

	// Copy the pcap file to the pod
	err := kubeapi.CopyFileToPod(p.Name, p.Name, filePath, inputFileContainer)
	if err != nil {
		return fmt.Errorf("error uploading pcap file to pod: %w", err)
	}

	// Execute the processing command in the processing pod
	envVars := make(map[string]string)
	envVars["INPUT_FILE"] = inputFileContainer
	envVars["INPUT_FILE_NAME"] = scn.Name
	envVars["OUTPUT_FILE"] = outputFileContainer
	log.Println("Analyzing traffic using pod: ", p.Name)
	stdo, stde, err := kubeapi.ExecShellInContainerWithEnvVars(apiv1.NamespaceDefault, p.Name, p.Name, p.Command, envVars)
	if err != nil {
		log.Printf("stdout: %s\nstderr: %s", stdo, stde)
		return fmt.Errorf("error analyzing traffic: %w", err)
	}
	// Print the output of the processing command to log file
	logFile, err := os.Create(outputLogFile)
	if err != nil {
		return fmt.Errorf("error creating log file traffic analysis: %w", err)
	}
	logFile.WriteString("stdout:\n" + stdo + "\n")
	logFile.WriteString("stderr:\n" + stde + "\n")
	logFile.Close()

	// Download the output file from the pod
	err = kubeapi.CopyFileFromPod(p.Name, p.Name, outputFileContainer, outputFileDownload, false)
	if err != nil {
		return fmt.Errorf("error downloading output file from pod: %w", err)
	}

	// Add labels to the output file
	// Extract the headers and values from the scenario file
	keys := make([]string, 0, len(scn.Labels))
	values := make([]string, 0, len(scn.Labels))

	for key, value := range scn.Labels {
		keys = append(keys, key)
		values = append(values, value)
	}
	err = p.AddColumnsToCSV(outputFileDownload, keys, values, true)
	if err != nil {
		return fmt.Errorf("error adding labels to output file: %w", err)
	}

	return nil
}

func (p *ProcessingPod) DeployPod() error {
	exists, err := kubeapi.PodExists(p.Name)
	if err != nil {
		log.Fatalf("Error checking if pod %s exists: %v\n", p.Name, err)
	}
	if !exists {
		log.Printf("Creating Pod %s\n", p.Name)
		podSpec := scenario.ProcessingPodSpec(p.Name, p.ContainerImage)
		_, err = kubeapi.CreateRunningPod(podSpec)
		if err != nil {
			log.Fatalf("Error running processing pod: %v", err)
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

	// Close the original file to allow reopening it in write mode
	file.Close()

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
		return fmt.Errorf("error reopening file for writing: %v", err)
	}
	defer file.Close()

	// Write the updated records back to the original file
	writer := csv.NewWriter(file)
	if err := writer.WriteAll(records); err != nil {
		return fmt.Errorf("error writing to CSV file: %v", err)
	}

	// Close the writer to ensure all data is flushed
	writer.Flush()
	return nil
}
