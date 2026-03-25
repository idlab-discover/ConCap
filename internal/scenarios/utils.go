package scenarios

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// WriteScenario marshals the scenario to YAML and writes it to disk
func WriteScenario(scenario ScenarioInterface, outputDir string) error {
	b, err := yaml.Marshal(scenario)
	if err != nil {
		return fmt.Errorf("marshal scenario YAML: %w", err)
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

func copyLabels(labels map[string]string) map[string]string {
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}

func ConvertToStringKeys(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[fmt.Sprint(k)] = ConvertToStringKeys(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = ConvertToStringKeys(v)
		}
	}
	return i
}
