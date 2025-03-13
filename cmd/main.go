package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/idlab-discover/concap/internal/controller"
	"github.com/jessevdk/go-flags"
)

type FlagStore struct {
	Directory       string `short:"d" long:"dir" description:"The mount path on the host" required:"true"`
	Scenario        string `short:"s" long:"scenario" description:"The scenario's to run, default=all" default:"all"`
	NumberOfWorkers int    `short:"w" long:"workers" description:"The number of concurrent workers that will execute scenarios. If NumberOfWorkers is greater than the number of scenarios, a maximum of 1 worker per scenario will be spawned." default:"1"`
}

var flagstore FlagStore

// the init function of main will parse provided flags before running the main, gracefully stop running if parsing fails.
func init() {
	_, err := flags.Parse(&flagstore)
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}
}

func main() {
	outputDirAbsPath, _ := filepath.Abs(flagstore.Directory)
	scenarioDir := filepath.Join(outputDirAbsPath, "scenarios")
	processingDir := filepath.Join(outputDirAbsPath, "processingpods")
	completedDir := filepath.Join(outputDirAbsPath, "completed")

	// Setup channel to listen for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Wait for interrupt signal
	go func() {
		<-quit
		fmt.Println("Shutdown signal received, cleaning up...")
		// Perform cleanup operations here
		// TODO: Add cleanup operations
		os.Exit(0) // Exit after cleanup
	}()

	if _, err := os.Stat(scenarioDir); os.IsNotExist(err) {
		log.Fatalf("Scenario directory does not exist: %s", scenarioDir)
	}

	// Get all the processing pods in the processing directory
	processingPodPaths := readDir(processingDir)
	if len(processingPodPaths) == 0 {
		log.Fatalf("No processing pods found in %s", processingDir)
	}

	// Get all the scenarios in the scenario directory
	scenarioPaths := readDir(scenarioDir)

	if len(scenarioPaths) == 0 {
		log.Fatalf("No scenarios found.")
	} else {
		if flagstore.Scenario != "all" {
			// Specific scenario is given as parameter
			scenario_path := filepath.Join(scenarioDir, flagstore.Scenario)
			log.Printf("Scenarion %s selected to be run", flagstore.Scenario)
			_, err := os.Stat(scenario_path)
			if err != nil {
				log.Fatalf("Error opening specified scenario %s", scenario_path)
			}
			scenarioPaths = []string{scenario_path}
		}
		log.Println("Number of scenarios found: " + fmt.Sprint(len(scenarioPaths)))
		// Create the flow extraction pods
		controller.DeployFlowExtractionPods(processingPodPaths)

		// Create a channel to schedule scenarios
		scenarioChannel := make(chan controller.ScenarioScheduleRequest)

		// Create a waitgroup to wait for all scenarios to finish
		wg := sync.WaitGroup{}

		// Start the concurrent goroutines to schedule scenarios
		numWorkers := min(flagstore.NumberOfWorkers, len(scenarioPaths))
		log.Printf("Starting %d scenario workers", numWorkers)
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go controller.ScheduleScenarioWorker(scenarioChannel, &wg)
		}

		// Send the scenarios to be executed to the channel
		for _, scenarioPath := range scenarioPaths {
			scenarioChannel <- controller.ScenarioScheduleRequest{ScenarioPath: scenarioPath, OutputDir: completedDir}
		}

		// Close the channel to signal that all scenarios have been sent, this will cause the goroutines to exit their receive loop
		close(scenarioChannel)

		// Wait for all scenarios to finish
		wg.Wait()
		log.Println("All scenarios have finished. ")
	}
}

func readDir(dir string) []string {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Error reading directory: %s", err.Error())
	}

	var filepaths []string
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			filepaths = append(filepaths, filepath.Join(dir, entry.Name()))
		}
	}

	return filepaths
}
