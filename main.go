package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/jessevdk/go-flags"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ccap"
)

type FlagStore struct {
	Directory       string `short:"d" long:"dir" description:"The mount path on the host" required:"true"`
	Scenario        string `short:"s" long:"scenario" description:"The scenario's to run, default=all" optional:"true" default:"all"`
	NumberOfWorkers int    `short:"w" long:"workers" description:"The number of concurrent workers that will execute scenarios" default:"1"`
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

	dirEntries, err := os.ReadDir(scenarioDir)
	if err != nil {
		log.Fatalf("Error reading scenario directory: %s", err.Error())
	}

	var filepaths []string
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			filepaths = append(filepaths, filepath.Join(scenarioDir, entry.Name()))
		}
	}

	if len(filepaths) == 0 {
		log.Println("No scenarios found.")
	} else {
		log.Println("Number of scenarios found: " + fmt.Sprint(len(filepaths)))
		// Create the flow extraction pods
		ccap.DeployFlowExtractionPods()

		// Create a channel to schedule scenarios
		scenarioChannel := make(chan ccap.ScenarioScheduleRequest)

		// Create a waitgroup to wait for all scenarios to finish
		wg := sync.WaitGroup{}

		// Start the concurrent goroutines to schedule scenarios
		log.Printf("Starting %d scenario workers", flagstore.NumberOfWorkers)
		for i := 0; i < flagstore.NumberOfWorkers; i++ {
			wg.Add(1)
			go ccap.ScheduleScenarioWorker(scenarioChannel, &wg)
		}

		// Send the scenarios to be executed to the channel
		for _, scenarioPath := range filepaths {
			scenarioChannel <- ccap.ScenarioScheduleRequest{ScenarioPath: scenarioPath, OutputDir: completedDir}
		}

		// Close the channel to signal that all scenarios have been sent, this will cause the goroutines to exit their receive loop
		close(scenarioChannel)

		// Wait for all scenarios to finish
		wg.Wait()
		log.Println("All scenarios have finished. ")
	}
}
