package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Cipherscan struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewCipherscan() Cipherscan {
	return Cipherscan{weight: 1, scenarioType: scenario.Scanning}
}

func (cipherscan Cipherscan) BuildAtkCommand() []string {
	cipherscan.parts = []string{"cipherscan", "-v", "ugent.be"}
	return cipherscan.parts
}

func (cipherscan Cipherscan) Weight() int {
	return cipherscan.weight
}

func (cipherscan Cipherscan) ScenarioType() scenario.ScenarioType {
	return cipherscan.scenarioType
}
