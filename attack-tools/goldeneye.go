package atktools

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Goldeneye struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	targetDomain string
	method       []string
	nossl        []string
}

func NewGoldeneye() Goldeneye {
	goldeneye := Goldeneye{weight: 10, scenarioType: scenario.Scanning, targetDomain: "http://{{.TargetAddress}}"}
	goldeneye.method = []string{"get", "post", "random"}
	goldeneye.nossl = []string{"True", "False"}
	return goldeneye
}

// For more flags: https://github.com/jseidl/GoldenEye
func (goldeneye Goldeneye) BuildAtkCommand() []string {
	rand.Seed(time.Now().UnixNano())
	goldeneye.parts = []string{"goldeneye"}
	goldeneye.parts = append(goldeneye.parts, goldeneye.targetDomain, "-w", fmt.Sprint(rand.Intn(100)+10), "-s", fmt.Sprint(rand.Intn(500)+300))

	if rand.Float32() < 0.33 {
		goldeneye.parts = append(goldeneye.parts, "-m", fmt.Sprint(goldeneye.method[rand.Intn(len(goldeneye.method))]))
	}
	if rand.Float32() < 0.33 {
		goldeneye.parts = append(goldeneye.parts, "-n", fmt.Sprint(goldeneye.nossl[rand.Intn(len(goldeneye.nossl))]))
	}
	if rand.Float32() < 0.33 {
		goldeneye.parts = append(goldeneye.parts, "-https")
	}
	return goldeneye.parts
}

func (goldeneye Goldeneye) Weight() int {
	return goldeneye.weight
}

func (goldeneye Goldeneye) ScenarioType() scenario.ScenarioType {
	return goldeneye.scenarioType
}
