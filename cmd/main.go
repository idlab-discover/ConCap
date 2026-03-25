package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/idlab-discover/concap/internal/controller"
	kubeapi "github.com/idlab-discover/concap/internal/kubernetes"
	"github.com/jessevdk/go-flags"
)

type FlagStore struct {
	Directory       string `short:"d" long:"dir" description:"The mount path on the host" required:"true"`
	Scenario        string `short:"s" long:"scenario" description:"The scenario's to run, default=all" default:"all"`
	NumberOfWorkers int    `short:"w" long:"workers" description:"The number of concurrent workers that will execute scenarios. If NumberOfWorkers is greater than the number of scenarios, a maximum of 1 worker per scenario will be spawned." default:"1"`
}

var flagstore FlagStore

func parseFlags() error {
	_, err := flags.Parse(&flagstore)
	if err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	return nil
}

func main() {
	if err := parseFlags(); err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("Shutdown complete: %v", err)
			os.Exit(1)
		}
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if err := kubeapi.Init(ctx); err != nil {
		return fmt.Errorf("initialize Kubernetes client: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	watcherErrCh := make(chan error, 1)
	go func() {
		for err := range kubeapi.WatchErrors() {
			if err == nil {
				continue
			}
			select {
			case watcherErrCh <- fmt.Errorf("pod watcher failed: %w", err):
			default:
			}
			cancel()
			return
		}
	}()

	outputDirAbsPath, err := filepath.Abs(flagstore.Directory)
	if err != nil {
		return fmt.Errorf("resolve output directory %s: %w", flagstore.Directory, err)
	}
	scenarioDir := filepath.Join(outputDirAbsPath, "scenarios")
	processingDir := filepath.Join(outputDirAbsPath, "processingpods")
	completedDir := filepath.Join(outputDirAbsPath, "completed")

	if _, err := os.Stat(scenarioDir); os.IsNotExist(err) {
		return fmt.Errorf("scenario directory does not exist: %s", scenarioDir)
	}

	processingPodPaths, err := readDir(processingDir)
	if err != nil {
		return fmt.Errorf("read processing pod directory %s: %w", processingDir, err)
	}
	if len(processingPodPaths) == 0 {
		return fmt.Errorf("no processing pods found in %s", processingDir)
	}

	scenarioPaths, err := readDir(scenarioDir)
	if err != nil {
		return fmt.Errorf("read scenario directory %s: %w", scenarioDir, err)
	}
	if len(scenarioPaths) == 0 {
		return fmt.Errorf("no scenarios found")
	}

	if flagstore.Scenario != "all" {
		scenarioPath := filepath.Join(scenarioDir, flagstore.Scenario)
		log.Printf("Scenario %s selected to be run", flagstore.Scenario)
		if _, err := os.Stat(scenarioPath); err != nil {
			return fmt.Errorf("open specified scenario %s: %w", scenarioPath, err)
		}
		scenarioPaths = []string{scenarioPath}
	}

	log.Printf("Number of scenarios found: %d", len(scenarioPaths))
	if err := controller.DeployFlowExtractionPods(runCtx, processingPodPaths); err != nil {
		return fmt.Errorf("deploy flow extraction pods: %w", err)
	}

	scenarioChannel := make(chan controller.ScenarioScheduleRequest)
	scenarioResults := make(chan error, len(scenarioPaths))

	var wg sync.WaitGroup
	numWorkers := min(flagstore.NumberOfWorkers, len(scenarioPaths))
	log.Printf("Starting %d scenario workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go controller.ScheduleScenarioWorker(runCtx, scenarioChannel, scenarioResults, &wg)
	}

	sendErr := enqueueScenarios(runCtx, scenarioChannel, scenarioPaths, completedDir)
	wg.Wait()
	close(scenarioResults)

	var errs []error
	if sendErr != nil {
		errs = append(errs, sendErr)
	}
	for err := range scenarioResults {
		errs = append(errs, err)
	}
	select {
	case err := <-watcherErrCh:
		errs = append(errs, err)
	default:
	}
	if err := controller.JoinErrors(errs); err != nil {
		return fmt.Errorf("one or more scenarios failed: %w", err)
	}

	log.Println("All scenarios have finished.")
	return nil
}

func enqueueScenarios(ctx context.Context, scenarioChannel chan<- controller.ScenarioScheduleRequest, scenarioPaths []string, outputDir string) error {
	defer close(scenarioChannel)

	for _, scenarioPath := range scenarioPaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case scenarioChannel <- controller.ScenarioScheduleRequest{
			ScenarioPath: scenarioPath,
			OutputDir:    outputDir,
		}:
		}
	}

	return nil
}

func readDir(dir string) ([]string, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var filepaths []string
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			filepaths = append(filepaths, filepath.Join(dir, entry.Name()))
		}
	}

	return filepaths, nil
}
