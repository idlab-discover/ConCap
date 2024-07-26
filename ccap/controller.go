package ccap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type ScenarioScheduleRequest struct {
	ScenarioPath string
	OutputDir    string
}

var (
	processingPods []*ProcessingPod
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
			processingPod, err := ReadProcessingPod(path)
			if err != nil {
				log.Fatalf("Error reading processing pod: %v", err)
			}
			err = processingPod.DeployPod()
			if err != nil {
				log.Fatalf("Error deploying processing pod: %v", err)
			}

			// Lock the mutex before accessing the shared slice
			mutex.Lock()
			processingPods = append(processingPods, processingPod)
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
		scen, err := ScheduleScenario(sceneRequest.ScenarioPath, sceneRequest.OutputDir)
		if err != nil {
			log.Printf("Error running scenario: %v\n", err)
			if scen != nil {
				// Clean up the failed scenario
				scen.DeletePods()
			}
		}
	}
}

// This function is run asynchronously, to allow for simultaneous execution of multiple scenarios at once.
func ScheduleScenario(scenarioPath string, outputDir string) (*Scenario, error) {
	scene, err := ReadScenario(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read scenario: " + scenarioPath)
	}
	log.Printf("Scenario loaded: %s\n", scene.Name)

	scenarioOutputFolder, err := mkdirScenarioOutput(outputDir, scene.Name)
	if err != nil {
		return scene, fmt.Errorf("error creating scenario output folder: %v", err)
	}
	scene.OutputDir = scenarioOutputFolder

	log.Printf("Scenario starting: %s\n", scene.Name)
	err = scene.Execute()
	if err != nil {
		return scene, fmt.Errorf("error executing scenario: %v", err)
	}

	// Process the pcap file concurrently by all the processing pods
	log.Printf("Analyzing traffic for scenario %v...", scene.Name)
	var wg sync.WaitGroup
	for _, pod := range processingPods {
		wg.Add(1)
		go func(pod *ProcessingPod, scene *Scenario) {
			defer wg.Done()
			err = pod.ProcessPcap(filepath.Join(scene.OutputDir, "dump.pcap"), scene)
			if err != nil {
				log.Printf("error analysing the pcap at processing pod %v: %v", pod.Name, err)
			}
		}(pod, scene)
	}
	wg.Wait()
	log.Println("Traffic analysis completed for scenario: ", scene.Name)

	log.Printf("Scenario finished: %s\n", scene.Name)
	return scene, nil
}

func mkdirScenarioOutput(outputDir string, scenarioName string) (string, error) {
	scenarioOutputFolder := filepath.Join(outputDir, scenarioName)
	if err := os.MkdirAll(scenarioOutputFolder, 0777); err != nil {
		return "", err
	}
	return scenarioOutputFolder, nil
}
