package atktools

import (
	"math/rand"
	"time"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Hulk struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	targetDomain string
}

func NewHulk() Hulk {
	hulk := Hulk{weight: 10, scenarioType: scenario.Scanning, targetDomain: "http://localhost"}
	return hulk
}

// Form more flags: https://github.com/grafov/hulk
func (hulk Hulk) BuildAtkCommand() []string {
	rand.Seed(time.Now().UnixNano())
	hulk.parts = []string{"python3", "hulk.py"}
	hulk.parts = append(hulk.parts, "-site", hulk.targetDomain, "2>/dev/null")
	if rand.Float32() < 0.33 {
		hulk.parts = append(hulk.parts, "-data", "peakaboo")
	}
	return hulk.parts
}

func (hulk Hulk) Weight() int {
	return hulk.weight
}

func (hulk Hulk) ScenarioType() scenario.ScenarioType {
	return hulk.scenarioType
}
