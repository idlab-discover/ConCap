package atktools

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type SlowHTTPTest struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
	targetDomain string
	modes        []string
	verbs        []string
}

func NewSlowHTTPTest() SlowHTTPTest {
	slowHTTPTest := SlowHTTPTest{weight: 10, scenarioType: scenario.Scanning, targetDomain: "http://{{.TargetAddress}}/"}
	slowHTTPTest.parts = []string{"slowhttptest"}
	slowHTTPTest.modes = []string{"-H", "-B", "-R", "-X"}
	slowHTTPTest.verbs = []string{"GET", "POST"}
	return slowHTTPTest
}

// For more flags: https://manpages.debian.org/unstable/slowhttptest/slowhttptest.1.en.html
func (slowHTTPTest SlowHTTPTest) BuildAtkCommand() []string {
	rand.Seed(time.Now().UnixNano())

	// Randomly choose a mode from -H, -B, -R, -X
	slowHTTPTest.parts = append(slowHTTPTest.parts, slowHTTPTest.modes[rand.Intn(len(slowHTTPTest.modes))])

	slowHTTPTest.parts = append(slowHTTPTest.parts, "-c", fmt.Sprint(rand.Intn(100)+50))
	slowHTTPTest.parts = append(slowHTTPTest.parts, "-i", fmt.Sprint(rand.Intn(20)))
	slowHTTPTest.parts = append(slowHTTPTest.parts, "-r", fmt.Sprint(rand.Intn(100)+30))
	slowHTTPTest.parts = append(slowHTTPTest.parts, "-u", slowHTTPTest.targetDomain)

	if rand.Float32() < 0.33 {
		slowHTTPTest.parts = append(slowHTTPTest.parts, "-t", slowHTTPTest.verbs[rand.Intn(len(slowHTTPTest.verbs))]) // replace with actual proxy host and port
	}
	return slowHTTPTest.parts
}

func (slowHTTPTest SlowHTTPTest) Weight() int {
	return slowHTTPTest.weight
}

func (slowHTTPTest SlowHTTPTest) ScenarioType() scenario.ScenarioType {
	return slowHTTPTest.scenarioType
}
