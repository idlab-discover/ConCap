package atktools

import (
	"math/rand"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Cangibrina struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewCangibrina() Cangibrina {
	return Cangibrina{weight: 1, scenarioType: scenario.Scanning}
}

func (cangibrina Cangibrina) BuildAtkCommand() []string {
	// TODO domain selection
	cangibrina.parts = []string{"printf \"y\n\"", "|", "cangibrina", "-t 20", "-u ugent.be"}
	if rand.Float32() < 0.2 {
		cangibrina.parts = append(cangibrina.parts, "--sub-domain")
	} else {
		cangibrina.parts = append(cangibrina.parts, "-w /usr/share/cangibrina/wordlists/wl_big")
	}
	return cangibrina.parts
}

func (cangibrina Cangibrina) Weight() int {
	return cangibrina.weight
}

func (cangibrina Cangibrina) ScenarioType() scenario.ScenarioType {
	return cangibrina.scenarioType
}
