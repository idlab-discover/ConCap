package atktools

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Slowloris struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	targetDomain string
}

func NewSlowloris() Slowloris {
	slowloris := Slowloris{weight: 10, scenarioType: scenario.Scanning, targetDomain: "localhost"}
	return slowloris
}

// For more flags: https://github.com/gkbrk/slowloris
func (slowloris Slowloris) BuildAtkCommand() []string {
	rand.Seed(time.Now().UnixNano())
	slowloris.parts = []string{"python3", "slowloris.py"}
	slowloris.parts = append(slowloris.parts, "--sockets", fmt.Sprint(rand.Intn(100)+150), slowloris.targetDomain)
	return slowloris.parts
}

func (slowloris Slowloris) Weight() int {
	return slowloris.weight
}

func (slowloris Slowloris) ScenarioType() scenario.ScenarioType {
	return slowloris.scenarioType
}
