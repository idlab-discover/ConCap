package atktools

import (
	"math/rand"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Verbal struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewVerbal() Verbal {
	return Verbal{weight: 1, scenarioType: scenario.Scanning}
}

func (verbal Verbal) BuildAtkCommand() []string {
	verbal.parts = []string{"verbal", "-A", "-u"}
	if rand.Float32() < 0.5 {
		verbal.parts = append(verbal.parts, "http://127.0.0.1")
	} else {
		verbal.parts = append(verbal.parts, "https://127.0.0.1")
	}
	return verbal.parts
}

func (verbal Verbal) Weight() int {
	return verbal.weight
}

func (verbal Verbal) ScenarioType() scenario.ScenarioType {
	return verbal.scenarioType
}
