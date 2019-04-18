package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Enumshares struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewEnumshares() Enumshares {
	return Enumshares{weight: 1, scenarioType: scenario.Scanning}
}

func (enumshares Enumshares) BuildAtkCommand() []string {
	enumshares.parts = []string{"printf \"yes\n\"", "|", "enum-shares", "-w", "-t", "localhost", "-u", "root", "-p", "root"}
	return enumshares.parts
}

func (enumshares Enumshares) Weight() int {
	return enumshares.weight
}

func (enumshares Enumshares) ScenarioType() scenario.ScenarioType {
	return enumshares.scenarioType
}
