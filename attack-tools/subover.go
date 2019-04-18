package atktools

import (
	"math/rand"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Subover struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewSubover() Subover {
	return Subover{weight: 1, scenarioType: scenario.Scanning}
}

func (subover Subover) BuildAtkCommand() []string {
	subover.parts = []string{"subover", "-t 50"}
	if rand.Float32() < 0.5 {
		subover.parts = append(subover.parts, "-https")
	}
	return subover.parts
}

func (subover Subover) Weight() int {
	return subover.weight
}

func (subover Subover) ScenarioType() scenario.ScenarioType {
	return subover.scenarioType
}
