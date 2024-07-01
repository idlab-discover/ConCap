// Package scenario handles our scenario specification through a number of structs.
// It also contains the podbuilder which fills in the Kubernetes templates for pods with the appropriate information
// These scenario files are exported to yaml and serve as the starting point of every experiment
package scenario

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type ScenarioType string

const (
	Scanning   ScenarioType = "scan"
	Exploit    ScenarioType = "exploit"
	Bruteforce ScenarioType = "bruteforce"
	Dos        ScenarioType = "dos"
	DDoS       ScenarioType = "ddos"
)

const EmptyAttackDuration = ""

type Scenario struct {
	UUID          uuid.UUID
	Name          string
	ScenarioType  ScenarioType  `yaml:"scenarioType"`
	StartTime     time.Time     `yaml:"startTime"`
	StopTime      time.Time     `yaml:"stopTime"`
	Attacker      Attacker      `yaml:"attacker"`
	Target        Target        `yaml:"target"`
	Support       []Support     `yaml:"support"`
	CaptureEngine CaptureEngine `yaml:"captureEngine"`
	Tag           string        `yaml:"tag"`
}

type Attacker struct {
	Category   ScenarioType `yaml:"category"`
	Name       string       `yaml:"name"`
	Image      string       `yaml:"image"`
	AtkCommand string       `yaml:"atkCommand"`
	AtkTime    string       `yaml:"atkTime"`
}

type Target struct {
	Category string  `yaml:"category"`
	Name     string  `yaml:"name"`
	Image    string  `yaml:"image"`
	Ports    []int32 `yaml:"ports"`
}

type Support struct {
	Category   string  `yaml:"category"`
	Name       string  `yaml:"name"`
	Image      string  `yaml:"image"`
	SupCommand string  `yaml:"supCommand"`
	Ports      []int32 `yaml:"ports"`
}

type CaptureEngine struct {
	Name      string `yaml:"name"`
	Image     string `yaml:"image"`
	Interface string `yaml:"interface"`
	Filter    string `yaml:"filter"`
}

// ReadScenario will unmarshall the yaml back into the in-memory Scenario representation
func ReadScenario(filePath string) (*Scenario, error) {
	s := Scenario{}

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

	err = yaml.UnmarshalStrict(b, &s)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	// TODO more checking needed here
	if s.Attacker.Image == "" {
		return nil, fmt.Errorf("no attack-image provided for attack: '%s'", s.Attacker.Name)
	}

	atkTime, err := parseToSeconds(s.Attacker.AtkTime)
	if err != nil {
		s.Attacker.AtkTime = EmptyAttackDuration
	} else {
		s.Attacker.AtkTime = atkTime
	}

	s.UUID = uuid.New()
	s.Name = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSuffix(filepath.Base(fileHandler.Name()), filepath.Ext(fileHandler.Name())), "_", "-"), " ", "-")
	return &s, nil
}

// WriteScenario will marshall the in-memory Scenario to valid yaml and write it to disk
func WriteScenario(s *Scenario, outputDir string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return os.WriteFile(filepath.Join(outputDir, "scenario.yaml"), b, 0644)
}

type Container struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetTimeout converts a time string (e.g., "10s", "2m", "1h") to a standardized
// string representation of seconds (e.g., "600s" for "10m").
// It returns an error if the input is in an unsupported format.
func parseToSeconds(s string) (string, error) {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return EmptyAttackDuration, err
	}

	seconds := int(duration.Seconds())
	return fmt.Sprint(seconds) + "s", nil
}
