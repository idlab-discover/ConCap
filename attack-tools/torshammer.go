package atktools

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type TorsHammer struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	targetDomain string
}

func NewTorshammer() TorsHammer {
	torshammer := TorsHammer{weight: 10, scenarioType: scenario.Scanning, targetDomain: "localhost"}
	return torshammer
}

// For more flags: https://github.com/Karlheinzniebuhr/torshammer/blob/master/torshammer.py
func (torshammer TorsHammer) BuildAtkCommand() []string {
	rand.Seed(time.Now().UnixNano())
	torshammer.parts = []string{"python", "torshammer.py"}
	torshammer.parts = append(torshammer.parts, "-t", torshammer.targetDomain, "--threads", fmt.Sprint(rand.Intn(100000)+256), "-T")
	return torshammer.parts
}

func (torshammer TorsHammer) Weight() int {
	return torshammer.weight
}

func (torshammer TorsHammer) ScenarioType() scenario.ScenarioType {
	return torshammer.scenarioType
}
