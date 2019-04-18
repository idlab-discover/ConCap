package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type AutoNSE struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewAutoNSE() AutoNSE {
	return AutoNSE{weight: 1, scenarioType: scenario.Scanning}
}

func (autonse AutoNSE) BuildAtkCommand() []string {
	autonse.parts = []string{"printf", "\"n\nlocalhost\n\"", "|", "autonse"}
	return autonse.parts
}

func (autonse AutoNSE) Weight() int {
	return autonse.weight
}

func (autonse AutoNSE) ScenarioType() scenario.ScenarioType {
	return autonse.scenarioType
}
