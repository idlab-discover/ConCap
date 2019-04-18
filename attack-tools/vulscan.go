package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Vulscan struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewVulscan() Vulscan {
	return Vulscan{weight: 1, scenarioType: scenario.Scanning}
}

func (vulscan Vulscan) BuildAtkCommand() []string {
	vulscan.parts = []string{"nmap", "-sV", "--script=vulscan/vulscan.nse", "--script-args", "vulscanoutput=details", "localhost"}
	return vulscan.parts
}

func (vulscan Vulscan) Weight() int {
	return vulscan.weight
}

func (vulscan Vulscan) ScenarioType() scenario.ScenarioType {
	return vulscan.scenarioType
}
