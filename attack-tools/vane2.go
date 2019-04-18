package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Vane2 struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewVane2() Vane2 {
	return Vane2{weight: 1, scenarioType: scenario.Scanning}
}

func (vane2 Vane2) BuildAtkCommand() []string {
	// TODO wordpress domains, preferably vulnerable
	vane2.parts = []string{"vane import-data; vane scan -pv --url http://chefilan.com/blog/"}
	return vane2.parts
}

func (vane2 Vane2) Weight() int {
	return vane2.weight
}

func (vane2 Vane2) ScenarioType() scenario.ScenarioType {
	return vane2.scenarioType
}
