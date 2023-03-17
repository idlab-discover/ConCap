// Package scenario handles our scenario specification through a number of structs.
// It also contains the podbuilder which fills in the Kubernetes templates for pods with the appropriate information
// These scenario files are exported to yaml and serve as the starting point of every experiment
package scenario

import (
	"io"
	"io/ioutil"
	"log"
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

type Scenario struct {
	UUID          uuid.UUID
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
func ReadScenario(r io.Reader) *Scenario {
	S := Scenario{}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("error1: %v", err.Error())
	}

	err = yaml.UnmarshalStrict(b, &S)
	if err != nil {
		log.Fatalf("error2: %v", err.Error())
	}
	//fmt.Printf("Scenario struct %+v\n", S)
	return &S
}

// WriteScenario will marshall the in-memory Scenario to valid yaml and write it to disk
func WriteScenario(s *Scenario, f string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return ioutil.WriteFile(f, b, 0644)
}
