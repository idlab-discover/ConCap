package atktools

import (
	"strconv"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type WordpressScan struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewWordpressscan() WordpressScan {
	return WordpressScan{weight: 1, scenarioType: scenario.Scanning}
}

func (wpsscan WordpressScan) BuildAtkCommand() []string {
	wpsscan.parts = []string{"wordpresscan", "-u", "http://127.0.0.1/wordpress", "--fuzz", "--random-agent", "--threads", strconv.Itoa(50)}
	return wpsscan.parts
}

func (wpsscan WordpressScan) Weight() int {
	return wpsscan.weight
}

func (wpsscan WordpressScan) ScenarioType() scenario.ScenarioType {
	return wpsscan.scenarioType
}
