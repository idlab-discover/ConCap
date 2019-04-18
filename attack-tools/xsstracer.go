package atktools

import (
	"strconv"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Xsstracer struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	targetPort   int
}

func NewXsstracer() Xsstracer {
	return Xsstracer{weight: 1, scenarioType: scenario.Scanning}
}

func (xsstracer Xsstracer) BuildAtkCommand() []string {
	xsstracer.parts = []string{"xsstracer", "127.0.0.1", strconv.Itoa(xsstracer.targetPort)}
	return xsstracer.parts
}

func (xsstracer Xsstracer) Weight() int {
	return xsstracer.weight
}

func (xsstracer Xsstracer) ScenarioType() scenario.ScenarioType {
	return xsstracer.scenarioType
}
