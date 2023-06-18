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
	"strconv"
	"strings"
	"time"
	"unicode"

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
func ReadScenario(r io.Reader) *Scenario {
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
	var atkTime = GetTimeout(s.Attacker.AtkTime)
	if atkTime == "" {
		s.Attacker.AtkTime = "60s"
	} else {
		s.Attacker.AtkTime = atkTime
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

// Tested and works but only with latest version
func SearchImage(attackerName string) (string, error) {
	git, _ := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), gitlab.WithBaseURL("https://gitlab.ilabt.imec.be/api/v4"))
	projectID := 880

	options := &gitlab.ListRegistryRepositoriesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}
	registryRepos, _, err := git.ContainerRegistry.ListProjectRegistryRepositories(projectID, options)
	if err != nil {
		fmt.Printf("Error fetching registry repositories: %s\n", err)
		return "", err
	}
	fmt.Println("There are " + fmt.Sprint(len(registryRepos)) + " repositories")
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

// Tested and works
func GetTimeout(input string) string {
	var seconds int

	letterIndex := strings.IndexFunc(input, func(c rune) bool {
		return !unicode.IsNumber(c)
	})

	if letterIndex == -1 {
		return ""
	}
	if len(input) != letterIndex+1 {
		return ""
	}
	// Extract the number and the letter
	numberStr := input[:letterIndex]
	letter := input[letterIndex]

	// Parse the number
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return ""
	}

	// Convert the number to seconds based on the letter
	switch letter {
	case 's':
		seconds = number
	case 'm':
		seconds = number * 60
	case 'h':
		seconds = number * 3600
	default:
		return ""
	}

	return fmt.Sprint(seconds) + "s"
}
