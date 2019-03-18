package scenario

import (
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type Scenario struct {
	UUID          uuid.UUID
	StartTime     time.Time `yaml:"startTime"`
	StopTime      time.Time `yaml:"stopTime"`
	Attacker      Attacker
	Target        Target
	CaptureEngine CaptureEngine `yaml:"captureEngine"`
	Tag           string
}

type Attacker struct {
	Name       string
	Image      string
	AtkCommand string `yaml: atkCommand`
}

type Target struct {
	Name  string
	Image string
	Ports []int32
}

type CaptureEngine struct {
	Name      string
	Image     string
	Interface string
	Filter    string
}

func ReadScenario(r io.Reader) *Scenario {
	S := Scenario{}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = yaml.UnmarshalStrict(b, &S)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	S.UUID = uuid.New()
	//fmt.Printf("Scenario struct %+v\n", S)

	return &S
}

func WriteScenario(s *Scenario, f string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return ioutil.WriteFile(f, b, 0644)
}
