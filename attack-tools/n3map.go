package atktools

import (
	"math/rand"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type N3Map struct {
	scenarioType                     scenario.ScenarioType
	weight                           int
	parts, enumeration, nsec, output []string
}

func NewN3map() N3Map {
	n3map := N3Map{weight: 5, scenarioType: scenario.Scanning}
	n3map.enumeration = []string{"--auto", "--nsec3", "--nsec"}
	n3map.nsec = []string{"--query-mode=mixed", "--query-mode=A", "--query-mode=NSEC"}
	// n3map.output = []string{-o -}
	return n3map
}

func (n3map N3Map) BuildAtkCommand() []string {
	n3map.parts = []string{"n3map", "-v", "-f 10"}
	n3map.parts = append(n3map.parts, n3map.enumeration[rand.Intn(len(n3map.enumeration))])
	n3map.parts = append(n3map.parts, n3map.nsec[rand.Intn(len(n3map.nsec))])
	return n3map.parts
}

func (n3map N3Map) Weight() int {
	return n3map.weight
}

func (n3map N3Map) ScenarioType() scenario.ScenarioType {
	return n3map.scenarioType
}
