package atktools

import "math/rand"

type Cangibrina struct {
	weight int
	parts  []string
}

func (cangibrina Cangibrina) BuildAtkCommand() []string {
	// TODO domain selection
	cangibrina.parts = []string{"printf \"y\n\"", "|", "cangibrina", "-t 20", "-u ugent.be"}
	if rand.Float32() < 0.2 {
		cangibrina.parts = append(cangibrina.parts, "--sub-domain")
	} else {
		cangibrina.parts = append(cangibrina.parts, "-w /usr/share/cangibrina/wordlists/wl_big")
	}
	cangibrina.weight = 1
	return cangibrina.parts
}
