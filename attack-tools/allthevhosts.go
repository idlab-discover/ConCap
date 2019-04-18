package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Allthevhosts struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewAllthevhosts() Allthevhosts {
	allthevhosts := Allthevhosts{weight: 1, scenarioType: scenario.Scanning}
	return allthevhosts
}

func (allthevhosts Allthevhosts) BuildAtkCommand() []string {
	allthevhosts.parts = []string{"allthevhosts", "127.0.0.1"}
	return allthevhosts.parts
}

func (allthevhosts Allthevhosts) Weight() int {
	return allthevhosts.weight
}

func (allthevhosts Allthevhosts) ScenarioType() scenario.ScenarioType {
	return allthevhosts.scenarioType
}
