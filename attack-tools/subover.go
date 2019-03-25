package atktools

import "math/rand"

type SubOver struct {
	parts []string
}

func (subover SubOver) BuildAtkCommand() []string {
	subover.parts = []string{"subover", "-t 50"}
	if rand.Float32() < 0.5 {
		subover.parts = append(subover.parts, "-https")
	}

	return subover.parts
}
