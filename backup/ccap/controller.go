package ccap

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	scenarios "github.com/idlab-discover/concap/ccap/scenarios"
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
func DeployFlowExtractionPods(processingPodPaths []string) {
	var wg sync.WaitGroup
	log.Print("Creating flow extraction pods")

	for _, processingPodPath := range processingPodPaths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			processingPod, err := scenarios.ReadProcessingPod(path)
			if err != nil {
				log.Fatalf("Error reading processing pod: %v", err)
			}
			err = processingPod.DeployPod()
			if err != nil {
				log.Fatalf("Error deploying processing pod: %v", err)
			}

			// Lock the mutex before accessing the shared slice
			mutex.Lock()
			ProcessingPods = append(ProcessingPods, processingPod)
			mutex.Unlock() // Unlock the mutex after updating the slice
		}(processingPodPath)
	}
	wg.Wait()
	log.Print("Flow extraction pods Created")
}

// Goroutine receiving scenario requests and scheduling them for execution
func ScheduleScenarioWorker(ch chan ScenarioScheduleRequest, wg *sync.WaitGroup) {
	defer wg.Done()
	for sceneRequest := range ch {
		processScenarioRequest(sceneRequest)
	}
}

// processScenarioRequest processes a scenario request
func processScenarioRequest(sceneRequest ScenarioScheduleRequest) {
	// Read the scenario
	scenario, err := scenarios.CreateScenario(sceneRequest.ScenarioPath)
	if err != nil {
		log.Printf("failed to read scenario %s: %s ", sceneRequest.ScenarioPath, err)
		return
	}

	scenarioName := scenario.GetName()
	log.Printf("Scenario loaded: %s\n", scenarioName)

	// Create the output directory
	scenarioOutputFolder := filepath.Join(sceneRequest.OutputDir, scenarioName)
	if err := os.MkdirAll(scenarioOutputFolder, 0777); err != nil {
		log.Printf("error creating scenario output folder: %v", err)
		return
	}

	// Execute the scenario
	err = scenario.Execute(scenarioOutputFolder)
	if err != nil {
		log.Printf("error executing scenario: %v", err)
		return
	}

	// Process the results of the scenario
	log.Printf("Analyzing traffic for scenario %v...", scenarioName)
	err = scenario.ProcessResults(scenarioOutputFolder, ProcessingPods)
	if err != nil {
		log.Printf("error processing results: %v", err)
		return
	}
	log.Println("Traffic analysis completed for all targets in scenario:", scenarioName)

	log.Printf("Scenario finished: %s\n", scenarioName)
}
