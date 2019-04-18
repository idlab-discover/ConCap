package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Laf struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts, sys   []string
}

func NewLaf() Laf {
	laf := Laf{weight: 5, scenarioType: scenario.Scanning}
	laf.sys = []string{"dirs", "php", "cfm", "asp", "pl", "html", "pma"}
	return laf
}

// todo add port specification via host:port notation which works
func (laf Laf) BuildAtkCommand() []string {
	laf.parts = []string{"laf", "-d", "localhost", "-u", "admin", "-p", "admin"}
	return laf.parts
}

func (laf Laf) Weight() int {
	return laf.weight
}

func (laf Laf) ScenarioType() scenario.ScenarioType {
	return laf.scenarioType
}
