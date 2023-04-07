package atktools

import (
	"math/rand"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Hydra struct {
	scenarioType                                                                        scenario.ScenarioType
	weight                                                                              int
	parts, protocol, task, user, pass, options, misc, bruteForceEngine, additionalFlags []string
}

func NewHydra() Hydra {
	hydra := Hydra{weight: 50, scenarioType: scenario.Scanning}
	hydra.protocol = []string{"ssh://", "ftp://", "telnet://", "http://", "https://", "smtp://", "pop3://", "imap://"}
	hydra.task = []string{"-t 1", "-t 2", "-t 4", "-t 8", "-t 16", "-t 32"}
	hydra.user = []string{"-l user", "-L userlist.txt"}
	hydra.pass = []string{"-p pass", "-P passlist.txt"}
	hydra.options = []string{"-s 22", "-o output.txt"}
	hydra.misc = []string{"-v", "-V", "-q", "-f", "-w 30", "-W 0", "-4", "-6"}
	hydra.bruteForceEngine = []string{"-e ns", "-e nsr", "-e n"}
	hydra.additionalFlags = []string{"-M", "-R"}

	return hydra
}

func (hydra Hydra) BuildAtkCommand() []string {

	hydra.parts = []string{"hydra"}
	hydra.parts = append(hydra.parts, hydra.task[rand.Intn(len(hydra.task))])
	hydra.parts = append(hydra.parts, hydra.user[rand.Intn(len(hydra.user))])
	hydra.parts = append(hydra.parts, hydra.pass[rand.Intn(len(hydra.pass))])

	if rand.Float32() < 0.33 {
		hydra.parts = append(hydra.parts, hydra.options[rand.Intn(len(hydra.options))])
	}
	if rand.Float32() < 0.33 {
		hydra.parts = append(hydra.parts, hydra.misc[rand.Intn(len(hydra.misc))])
	}
	if rand.Float32() < 0.33 {
		hydra.parts = append(hydra.parts, hydra.bruteForceEngine[rand.Intn(len(hydra.bruteForceEngine))])
	}
	if rand.Float32() < 0.33 {
		hydra.parts = append(hydra.parts, hydra.additionalFlags[rand.Intn(len(hydra.additionalFlags))])
	}
	hydra.parts = append(hydra.parts, hydra.protocol[rand.Intn(len(hydra.protocol))])
	hydra.parts = append(hydra.parts, "localhost")

	return hydra.parts
}

func (hydra Hydra) Weight() int {
	return hydra.weight
}

func (hydra Hydra) ScenarioType() scenario.ScenarioType {
	return hydra.scenarioType
}
