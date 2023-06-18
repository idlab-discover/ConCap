package atktools

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Rudy struct {
	scenarioType                       scenario.ScenarioType
	weight                             int
	parts                              []string
	targetDomain                       string
	length, numberOfConnections, delay []int32
	method                             []string
}

func NewRudy() Rudy {
	rudy := Rudy{weight: 10, scenarioType: scenario.Scanning, targetDomain: "http://{{.TargetAddress}}:8080"}
	rudy.length = []int32{1048576, 62500, 31250, 2097152, 15625}
	rudy.numberOfConnections = []int32{100, 200, 250, 500, 1000, 750}
	rudy.delay = []int32{2, 3, 4, 5, 6, 7, 8, 9}
	rudy.method = []string{"POST", "GET"}
	return rudy
}

// For more flags: https://github.com/gkbrk/rudy
func (rudy Rudy) BuildAtkCommand() []string {
	rand.Seed(time.Now().UnixNano())
	rudy.parts = []string{"rudy"}
	rudy.parts = append(rudy.parts, "-t", rudy.targetDomain)

	if rand.Float32() < 0.33 {
		rudy.parts = append(rudy.parts, "-l", fmt.Sprint(rudy.length[rand.Intn(len(rudy.length))]))
		return rudy.parts
	}
	if rand.Float32() < 0.33 {
		rudy.parts = append(rudy.parts, "-n", fmt.Sprint(rudy.numberOfConnections[rand.Intn(len(rudy.numberOfConnections))]))
	}
	if rand.Float32() < 0.33 {
		rudy.parts = append(rudy.parts, "-d", fmt.Sprint(rudy.delay[rand.Intn(len(rudy.delay))]))
	}

	return rudy.parts
}

func (rudy Rudy) Weight() int {
	return rudy.weight
}

func (rudy Rudy) ScenarioType() scenario.ScenarioType {
	return rudy.scenarioType
}
