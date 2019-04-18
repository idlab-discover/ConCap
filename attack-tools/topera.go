package atktools

import (
	"math/rand"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Topera struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts, modes []string
}

func NewTopera() Topera {
	topera := Topera{weight: 5, scenarioType: scenario.Scanning}
	topera.modes = []string{"topera_tcp_scan", "topera_loris"}
	return topera
}

func (topera Topera) BuildAtkCommand() []string {
	topera.parts = []string{"topera", "-t ::1", "-M"}
	topera.parts = append(topera.parts, topera.modes[rand.Intn(len(topera.modes))])
	topera.weight = 5
	return topera.parts
}

func (topera Topera) Weight() int {
	return topera.weight
}

func (topera Topera) ScenarioType() scenario.ScenarioType {
	return topera.scenarioType
}
