package scenarios

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// ScenarioInterface defines the common interface for all scenario types
type ScenarioInterface interface {
	// FromYAML parses a YAML file into a scenario
	FromYAML(filePath string) error
	// DeployAllPods deploys all pods for the scenario
	DeployAllPods() error
	// StartTrafficCapture starts traffic capture on the target pod(s)
	StartTrafficCapture() error
	// ExecuteAttack executes the attack
	ExecuteAttack() error
	// DownloadResults downloads the pcap capture and tcpdump log file from the target pod
	DownloadResults(outputDir string) error
	// ProcessResults processes the results of the attack
	ProcessResults(outputDir string, processingPods []*ProcessingPod) error
	// DeleteAllPods deletes all pods for the scenario
	DeleteAllPods() error
	// Execute executes the entire scenario workflow
	Execute(outputDir string) error
	// GetName returns the scenario name
	GetName() string
}

// BaseScenario contains common fields and methods for all scenario types
type BaseScenario struct {
	UUID      uuid.UUID `yaml:"uuid"`
	Name      string    `yaml:"name"`
	StartTime time.Time `yaml:"startTime"`
	StopTime  time.Time `yaml:"stopTime"`
	Type      string    `yaml:"type"`
}

// GetName returns the scenario name
func (s *BaseScenario) GetName() string {
	return s.Name
}

// ExecuteScenario executes the scenario from start to finish.
// 1. Deploys the pods
// 2. Start traffic capture on the target pod(s)
// 3. Executes the attack
// 4. Downloads the pcap capture and updated scenario file
// 5. Cleans up the pods
func ExecuteScenario(s ScenarioInterface, outputDir string) error {
	// 1. Deploy the pods for this scenario
	err := s.DeployAllPods()
	if err != nil {
		return fmt.Errorf("failed to deploy pods for scenario: %v", err)
	}

	// Defer pod deletion with error handling
	defer func() {
		if deleteErr := s.DeleteAllPods(); deleteErr != nil {
			log.Printf("Error: failed to clean up pods for scenario: %v", deleteErr)
		}
	}()

	// 2. Start traffic capture on the target pod(s)
	err = s.StartTrafficCapture()
	if err != nil {
		return fmt.Errorf("failed to start traffic capture for scenario: %v", err)
	}

	// 3. Execute the attack
	err = s.ExecuteAttack()
	if err != nil {
		return fmt.Errorf("failed to execute attack for scenario: %v", err)
	}

	// 4. Download the pcap capture and updated scenario file
	err = s.DownloadResults(outputDir)
	if err != nil {
		return fmt.Errorf("failed to download results for scenario: %v", err)
	}

	return nil
}
