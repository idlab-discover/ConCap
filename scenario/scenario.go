// Package scenario handles our scenario specification through a number of structs.
// It also contains the podbuilder which fills in the Kubernetes templates for pods with the appropriate information
// These scenario files are exported to yaml and serve as the starting point of every experiment
package scenario

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xanzy/go-gitlab"
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

func ReadScenario2(r io.Reader) *Scenario {
	s := Scenario{}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("error reading YAML: %v", err)
	}

	err = yaml.UnmarshalStrict(b, &s)
	if err != nil {
		log.Fatalf("error unmarshaling YAML: %v", err)
	}

	// handle case where image field in Attacker is empty
	if s.Attacker.Image == "" {

		fmt.Println("Attacker Image was not given, so checking for Image for given attack")
		fmt.Println()

		image, err := SearchImage(s.Attacker.Name)
		if image != "" {
			s.Attacker.Image = image
			fmt.Println("Found Image for attack: " + s.Attacker.Name)
			fmt.Println()
		} else {
			log.Fatalf("error finding the Image for " + err.Error())
		}

	}

	return &s
}

// WriteScenario will marshall the in-memory Scenario to valid yaml and write it to disk
func WriteScenario(s *Scenario, f string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return ioutil.WriteFile(f, b, 0644)
}

type Container struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func SearchImage(attackerName string) (string, error) {
	git, _ := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), gitlab.WithBaseURL("https://gitlab.ilabt.imec.be/api/v4"))
	projectID := 880

	registryRepos, _, err := git.ContainerRegistry.ListProjectRegistryRepositories(projectID, &gitlab.ListRegistryRepositoriesOptions{})
	if err != nil {
		fmt.Printf("Error fetching registry repositories: %s\n", err)
		return "", err
	}

	// Create a list to store the containers
	var containers []string

	// Iterate through the repositories and append the container names to the list
	for _, repo := range registryRepos {
		containers = append(containers, repo.Path)
	}

	for _, container := range containers {
		if strings.Contains(container, attackerName) {
			image := "gitlab.ilabt.imec.be:4567/lpdhooge/containercap-imagery/" + attackerName + ":latest"
			return image, nil
		}
	}
	return "", err
}
