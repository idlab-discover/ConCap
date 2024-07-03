package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jessevdk/go-flags"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ccap"
)

type FlagStore struct {
	Directory string `short:"d" long:"dir" description:"The mount path on the host" required:"true"`
	Scenario  string `short:"s" long:"scenario" description:"The scenario's to run, default=all" optional:"true" default:"all"`
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

		for _, scenarioPath := range filepaths {
			// TODO make this concurrent
			ccap.ScheduleScenario(scenarioPath, completedDir)
		}

		log.Println("All scenarios have finished. ")
	}
}
