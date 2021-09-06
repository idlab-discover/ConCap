package atktools

import (
	"math/rand"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

// A2sv is an SSL vulnerability scanner https://github.com/hahwul/a2sv
type A2sv struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	modules      []string
}

// NewA2sv sets basic information for this attack
func NewA2sv() A2sv {
	a2sv := A2sv{weight: 10, scenarioType: scenario.Scanning}
	a2sv.modules = []string{"anonymous", "crime", "heart", "ccs", "poodle", "freak", "logjam", "drown"}
	return a2sv
}

// BuildAtkCommand generates the default command for a2sv to use
func (a2sv A2sv) BuildAtkCommand() []string {
	a2sv.parts = []string{"a2sv", "-t", "127.0.0.1", "-o", "Y", "-m"}
	a2sv.parts = append(a2sv.parts, a2sv.modules[rand.Intn(len(a2sv.modules))])
	return a2sv.parts
}

func (a2sv A2sv) Weight() int {
	return a2sv.weight
}

func (a2sv A2sv) ScenarioType() scenario.ScenarioType {
	return a2sv.scenarioType
}
