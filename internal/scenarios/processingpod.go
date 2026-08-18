package scenarios

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	kubeapi "github.com/idlab-discover/concap/internal/kubernetes"
	"gopkg.in/yaml.v2"
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

func (p *ProcessingPod) ProcessPcap(ctx context.Context, filePath string, scenarioName string, targetName string, outputDir string) error {
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
	stdo, stde, err := kubeapi.ExecShellInContainerWithEnvVars(ctx, kubeapi.WorkloadNamespace, p.Name, p.Name, p.Command, envVars)
	if err != nil {
		log.Printf("stdout: %s\nstderr: %s", stdo, stde)
		return fmt.Errorf("error analyzing traffic: %w", err)
	}
	// Print the output of the processing command to log file
	if err := writeAnalysisLog(outputLogFile, p, stdo, stde); err != nil {
		return err
	}

	// Download the output file from the pod
	err = kubeapi.CopyFileFromPod(ctx, p.Name, p.Name, outputFileContainer, outputFileDownload, false)
	if err != nil {
		return fmt.Errorf("error downloading output file from pod: %w", err)
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

func writeAnalysisLog(outputPath string, processingPod *ProcessingPod, stdout, stderr string) error {
	logFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create traffic analysis log file: %w", err)
	}

	if _, err := fmt.Fprintf(
		logFile,
		"processor:\n  name: %s\n  image: %s\n  command: |-\n%s\nstdout:\n%s\nstderr:\n%s\n",
		processingPod.Name,
		processingPod.ContainerImage,
		indentLogBlock(processingPod.Command),
		stdout,
		stderr,
	); err != nil {
		logFile.Close()
		return fmt.Errorf("write traffic analysis log file: %w", err)
	}

	if err := logFile.Close(); err != nil {
		return fmt.Errorf("close traffic analysis log file: %w", err)
	}

	return nil
}

func indentLogBlock(value string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}
