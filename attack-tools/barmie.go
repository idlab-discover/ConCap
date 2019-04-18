package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Barmie struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewBarmie() Barmie {
	return Barmie{weight: 1, scenarioType: scenario.Scanning}
}

func (barmie Barmie) BuildAtkCommand() []string {
	barmie.parts = []string{"barmie", "-attack", "127.0.0.1"}
	return barmie.parts
}

func (barmie Barmie) Weight() int {
	return barmie.weight
}

func (barmie Barmie) ScenarioType() scenario.ScenarioType {
	return barmie.scenarioType
}
