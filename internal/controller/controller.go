package controller

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/idlab-discover/concap/internal/scenarios"
)

type ScenarioScheduleRequest struct {
	ScenarioPath string
	OutputDir    string
}

var (
	ProcessingPods []*scenarios.ProcessingPod
	mutex          sync.Mutex // Mutex to protect access to processingPods
)

// DeployFlowExtractionPods creates the flow extraction pods if they do not exist yet.
func DeployFlowExtractionPods(processingPodPaths []string) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(processingPodPaths))
	log.Print("Creating flow extraction pods")

	for _, processingPodPath := range processingPodPaths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			processingPod, err := scenarios.ReadProcessingPod(path)
			if err != nil {
				errCh <- fmt.Errorf("read processing pod %s: %w", path, err)
				return
			}
			err = processingPod.DeployPod()
			if err != nil {
				errCh <- fmt.Errorf("deploy processing pod %s: %w", processingPod.Name, err)
				return
			}

			// Lock the mutex before accessing the shared slice
			mutex.Lock()
			ProcessingPods = append(ProcessingPods, processingPod)
			mutex.Unlock() // Unlock the mutex after updating the slice
		}(processingPodPath)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	log.Print("Flow extraction pods Created")
	return nil
}

// Goroutine receiving scenario requests and scheduling them for execution.
func ScheduleScenarioWorker(ch <-chan ScenarioScheduleRequest, results chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	for sceneRequest := range ch {
		if err := processScenarioRequest(sceneRequest); err != nil {
			results <- err
		}
	}
}

// processScenarioRequest processes a scenario request.
func processScenarioRequest(sceneRequest ScenarioScheduleRequest) error {
	// Read the scenario
	scenario, err := scenarios.CreateScenario(sceneRequest.ScenarioPath)
	if err != nil {
		return fmt.Errorf("read scenario %s: %w", sceneRequest.ScenarioPath, err)
	}

	scenarioName := scenario.GetName()
	log.Printf("Scenario loaded: %s\n", scenarioName)

	// Create the output directory
	scenarioOutputFolder := filepath.Join(sceneRequest.OutputDir, scenarioName)
	if err := os.MkdirAll(scenarioOutputFolder, 0777); err != nil {
		return fmt.Errorf("create output directory for scenario %s: %w", scenarioName, err)
	}

	// Execute the scenario
	err = scenario.Execute(scenarioOutputFolder)
	if err != nil {
		return fmt.Errorf("execute scenario %s: %w", scenarioName, err)
	}

	// Process the results of the scenario
	log.Printf("Analyzing traffic for scenario %v...", scenarioName)
	err = scenario.ProcessResults(scenarioOutputFolder, ProcessingPods)
	if err != nil {
		return fmt.Errorf("process results for scenario %s: %w", scenarioName, err)
	}
	log.Println("Traffic analysis completed for all targets in scenario:", scenarioName)

	log.Printf("Scenario finished: %s\n", scenarioName)
	return nil
}

// JoinErrors combines worker errors into a single error.
func JoinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
