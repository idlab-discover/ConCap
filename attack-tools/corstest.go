package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Corstest struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewCorstest() Corstest {
	return Corstest{weight: 1, scenarioType: scenario.Scanning}
}

func (corstest Corstest) BuildAtkCommand() []string {
	corstest.parts = []string{"corstest", "-v"}
	return corstest.parts
}

func (corstest Corstest) Weight() int {
	return corstest.weight
}

func (corstest Corstest) ScenarioType() scenario.ScenarioType {
	return corstest.scenarioType
}
