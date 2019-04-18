package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

type Sslscan struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewSslscan() Sslscan {
	return Sslscan{weight: 1, scenarioType: scenario.Scanning}
}

func (sslscan Sslscan) BuildAtkCommand() []string {
	sslscan.parts = []string{"sslscan --no-failed --renegotiation --bugs 127.0.0.1:443"}
	return sslscan.parts
}

func (sslscan Sslscan) Weight() int {
	return sslscan.weight
}

func (sslscan Sslscan) ScenarioType() scenario.ScenarioType {
	return sslscan.scenarioType
}
