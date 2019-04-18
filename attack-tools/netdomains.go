package atktools

import (
	"math/rand"
	"strconv"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Netdomains struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	asns         []int
}

func NewNetdomains() Netdomains {
	netdomains := Netdomains{weight: 10, scenarioType: scenario.Scanning}
	netdomains.asns = []int{5432, 6848, 2611, 47377, 12392, 44944, 34762, 5488, 48517, 35219, 49964, 8201, 9208, 8368, 12942, 42160, 6774, 31713, 60436, 28707}
	return netdomains
}

func (netdomains Netdomains) BuildAtkCommand() []string {
	netdomains.parts = []string{"amass.netdomains", "-p 80,443,8080,8443", "-whois", "-asn"}
	netdomains.parts = append(netdomains.parts, strconv.Itoa(netdomains.asns[rand.Intn(len(netdomains.asns))]))
	return netdomains.parts
}

func (netdomains Netdomains) Weight() int {
	return netdomains.weight
}

func (netdomains Netdomains) ScenarioType() scenario.ScenarioType {
	return netdomains.scenarioType
}
