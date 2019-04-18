package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Subfinder struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewSubfinder() Subfinder {
	return Subfinder{weight: 1, scenarioType: scenario.Scanning}
}

func (subfinder Subfinder) BuildAtkCommand() []string {
	// TODO domain expansion
	subfinder.parts = []string{"subfinder", "-t 25", "-r 8.8.8.8,1.1.1.1", "-d", "ugent.be"}
	return subfinder.parts
}

func (subfinder Subfinder) Weight() int {
	return subfinder.weight
}

func (subfinder Subfinder) ScenarioType() scenario.ScenarioType {
	return subfinder.scenarioType
}
