package ccap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type ScenarioScheduleRequest struct {
	ScenarioPath string
	OutputDir    string
}

// DeployFlowExtractionPods creates the flow extraction pods if they do not exist yet.
func DeployFlowExtractionPods() {
	var wg sync.WaitGroup
	log.Print("Creating flow extraction pods")

	for _, processingPod := range scenario.ProcessingPods {
		wg.Add(1)
		go func(processingPod scenario.ProcessingPod) {
			defer wg.Done()

			exists, err := kubeapi.PodExists(processingPod.Name)
			if err != nil {
				log.Fatalf("Error checking if pod %s exists: %v\n", processingPod.Name, err)
			}
			if !exists {
				log.Printf("Creating Pod %s\n", processingPod.Name)
				podSpec := scenario.ProcessingPodSpec(processingPod.Name, processingPod.Image)
				_, err = kubeapi.CreateRunningPod(podSpec)
				if err != nil {
					log.Fatalf("Error running processing pod: %v", err)
				}
				log.Printf("Pod %s created\n", processingPod.Name)
			} else {
				log.Printf("Pod %s already exists\n", processingPod.Name)
			}
		}(processingPod)
	}
	wg.Wait()
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
func ScheduleScenario(scenarioPath string, outputDir string) (*scenario.Scenario, error) {
	scene, err := scenario.ReadScenario(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read scenario: " + scenarioPath)
	}
	log.Printf("Scenario loaded: %s\n", scene.Name)

	scenarioOutputFolder, err := mkdirScenarioOutput(outputDir, scene.Name)
	if err != nil {
		return scene, fmt.Errorf("error creating scenario output folder: %v", err)
	}
	scene.OutputDir = scenarioOutputFolder

	// TODO implement maximum concurrent scenarios
	log.Printf("Scenario starting: %s\n", scene.Name)
	err = scene.Execute()
	if err != nil {
		return scene, fmt.Errorf("error executing scenario: %v", err)
	}

	// Upload the pcap file to the processing pods
	log.Printf("Uploading pcap to processingpods for scenario %v...", scene.Name)
	kubeapi.CopyFileToPod("cicflowmeter", "cicflowmeter", filepath.Join(scene.OutputDir, "/dump.pcap"), filepath.Join("/data/pcap", scene.Name+".pcap"))
	kubeapi.CopyFileToPod("rustiflow", "rustiflow", filepath.Join(scene.OutputDir, "/dump.pcap"), filepath.Join("/data/pcap", scene.Name+".pcap"))

	// Perform flow reconstruction and feature extraction
	err = scene.AnalysePcap()
	if err != nil {
		return scene, fmt.Errorf("error analysing pcap file: %v", err)
	}

	// Copy analysis results to local and remove file from pod
	log.Printf("Downloading flows for scenario %v...", scene.Name)
	_ = kubeapi.CopyFileFromPod("cicflowmeter", "cicflowmeter", "/data/flow/"+scene.Name+".pcap_flow.csv", filepath.Join(scene.OutputDir, "cic-flows.csv"), false)
	_ = kubeapi.CopyFileFromPod("rustiflow", "rustiflow", "/data/flow/"+scene.Name+".csv", filepath.Join(scene.OutputDir, "rustiflow.csv"), false)

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
