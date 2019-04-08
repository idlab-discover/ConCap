package atktools

import "math/rand"

type SubOver struct {
	weight int
	parts  []string
}

func (subover SubOver) BuildAtkCommand() []string {
	subover.parts = []string{"subover", "-t 50"}
	if rand.Float32() < 0.5 {
		subover.parts = append(subover.parts, "-https")
	}
	subover.weight = 1
	return subover.parts
}
