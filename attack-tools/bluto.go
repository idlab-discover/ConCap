package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Bluto struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	targetDomain string
}

func NewBluto() Bluto {
	return Bluto{weight: 1, scenarioType: scenario.Scanning}
}

func (bluto Bluto) BuildAtkCommand() []string {
	bluto.parts = []string{"bluto", "-e"}
	bluto.parts = append(bluto.parts, "-d", bluto.targetDomain)
	return bluto.parts
}

func (bluto Bluto) Weight() int {
	return bluto.weight
}

func (bluto Bluto) ScenarioType() scenario.ScenarioType {
	return bluto.scenarioType
}
