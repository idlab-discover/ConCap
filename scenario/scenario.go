package scenario

import (
	"fmt"
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
	Tags          []string
}

type Attacker struct {
	Name  string
	Image string
}

type Target struct {
	Name  string
	Image string
	Ports []int32
}

type CaptureEngine struct {
	Name   string
	Image  string
	Filter string
}

func BuildScenario(r io.Reader) *Scenario {
	S := Scenario{}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.UnmarshalStrict(b, &S)
	if err != nil {
		log.Fatal(err)
	}
	S.UUID = uuid.New()
	fmt.Printf("Scenario struct %+v\n", S)

	return &S
}
