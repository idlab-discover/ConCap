package scenarios

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// ParseScenario determines the type of scenario and calls the appropriate parser
func ParseScenario(filePath string) (ScenarioInterface, error) {
	// First, open the file and determine the scenario type
	fileHandler, err := os.Open(filePath)
	if err != nil {
		log.Println("Failed to open file " + filePath + " : " + err.Error())
		return nil, err
	}
	defer fileHandler.Close()

	b, err := io.ReadAll(fileHandler)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML: %w", err)
	}

	// Unmarshal just enough to determine the scenario type
	var scenarioType ScenarioType
	err = yaml.Unmarshal(b, &scenarioType)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML to determine scenario type: %w", err)
	}

	// Check the scenario type and create the appropriate scenario
	switch scenarioType.Type {
	case MultiTargetType:
		var scenario MultiTargetScenario
		err = scenario.FromYAML(filePath)
		if err != nil {
			return nil, err
		}
		return &scenario, nil
	case SingleTargetType: // Default to SingleTarget if type is missing
		var scenario SingleTargetScenario
		err = scenario.FromYAML(filePath)
		if err != nil {
			return nil, err
		}
		return &scenario, nil
	default:
		return nil, fmt.Errorf("unknown scenario type: %s", scenarioType.Type)
	}
}

// WriteScenario marshals the scenario to YAML and writes it to disk
func WriteScenario(scenario ScenarioInterface, outputDir string) error {
	b, err := yaml.Marshal(scenario)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return os.WriteFile(filepath.Join(outputDir, "scenario.yaml"), b, 0644)
}

// ParseToSeconds converts a time string (e.g., "10s", "2m", "1h") to a standardized
// string representation of seconds (e.g., "600s" for "10m").
func ParseToSeconds(s string) (string, error) {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return EmptyAttackDuration, err
	}

	seconds := int(duration.Seconds())
	return fmt.Sprint(seconds) + "s", nil
}

// CleanPodName returns a cleaned up version of the pod name in lowercase with spaces and underscores replaced by dashes
func CleanPodName(name string) string {
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", ":", "-")
	return strings.ToLower(replacer.Replace(name))
}
