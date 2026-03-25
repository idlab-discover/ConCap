package scenarios

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	kubeexec "k8s.io/client-go/util/exec"
)

// ScenarioInterface defines the common interface for all scenario types
type ScenarioInterface interface {
	// FromYAML parses a YAML file into a scenario
	FromYAML(filePath string) error
	// DeployAllPods deploys all pods for the scenario
	DeployAllPods(ctx context.Context) error
	// StartTrafficCapture starts traffic capture on the target pod(s)
	StartTrafficCapture(ctx context.Context) error
	// ExecuteAttack executes the attack
	ExecuteAttack(ctx context.Context) error
	// DownloadResults downloads the pcap capture and tcpdump log file from the target pod
	DownloadResults(ctx context.Context, outputDir string) error
	// ProcessResults processes the results of the attack
	ProcessResults(ctx context.Context, outputDir string, processingPods []*ProcessingPod) error
	// DeleteAllPods deletes all pods for the scenario
	DeleteAllPods(ctx context.Context) error
	// Execute executes the entire scenario workflow
	Execute(ctx context.Context, outputDir string) error
	// GetName returns the scenario name
	GetName() string
}

type partialResultsDownloader interface {
	DownloadPartialResults(ctx context.Context, outputDir string) error
}

// BaseScenario contains common fields and methods for all scenario types
type BaseScenario struct {
	UUID      uuid.UUID `yaml:"uuid"`
	Name      string    `yaml:"name"`
	InitTime  time.Time `yaml:"initTime"`
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
func ExecuteScenario(ctx context.Context, s ScenarioInterface, outputDir string) error {
	// 1. Deploy the pods for this scenario
	err := s.DeployAllPods(ctx)
	if err != nil {
		return fmt.Errorf("failed to deploy pods for scenario: %w", err)
	}

	// Defer pod deletion with error handling
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if deleteErr := s.DeleteAllPods(cleanupCtx); deleteErr != nil {
			log.Printf("Error: failed to clean up pods for scenario: %v", deleteErr)
		}
	}()

	// 2. Start traffic capture on the target pod(s)
	err = s.StartTrafficCapture(ctx)
	if err != nil {
		return fmt.Errorf("failed to start traffic capture for scenario: %w", err)
	}

	// 3. Execute the attack
	err = s.ExecuteAttack(ctx)
	if err != nil {
		attackErr := fmt.Errorf("failed to execute attack for scenario: %w", err)
		if isAttackTimeout(err) {
			if partialDownloader, ok := s.(partialResultsDownloader); ok {
				if partialErr := partialDownloader.DownloadPartialResults(ctx, outputDir); partialErr != nil {
					return errors.Join(attackErr, fmt.Errorf("failed to preserve partial results for scenario: %w", partialErr))
				}
			}
		}
		return attackErr
	}

	// 4. Download the pcap capture and updated scenario file
	err = s.DownloadResults(ctx, outputDir)
	if err != nil {
		return fmt.Errorf("failed to download results for scenario: %w", err)
	}

	return nil
}

func isAttackTimeout(err error) bool {
	var exitErr kubeexec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitStatus() == 124
}
