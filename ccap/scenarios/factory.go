package ccap

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// ScenarioTypeConfig is used to determine the scenario type from YAML
type ScenarioTypeConfig struct {
	Type string `yaml:"type"`
}

// CreateScenario creates a scenario of the appropriate type based on the YAML file
func CreateScenario(filePath string) (ScenarioInterface, error) {
	// First, read the file to determine the scenario type
	fileHandler, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer fileHandler.Close()

	b, err := io.ReadAll(fileHandler)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML: %w", err)
	}

	// Parse the type field to determine the scenario type
	var typeConfig ScenarioTypeConfig
	err = yaml.Unmarshal(b, &typeConfig)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML to determine type: %w", err)
	}

	// Create the appropriate scenario type based on the type field
	var scenario ScenarioInterface
	switch strings.ToLower(typeConfig.Type) {
	case SingleTargetType:
		scenario = &SingleTargetScenario{}
	case MultiTargetType:
		scenario = &MultiTargetScenario{}
	default:
		// Default to single target if type is not specified or unknown
		scenario = &SingleTargetScenario{}
	}

	// Parse the YAML file into the scenario
	err = scenario.FromYAML(filePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML into scenario: %w", err)
	}

	return scenario, nil
}
