package atktools

import "math/rand"

type Verbal struct {
	weight int
	parts  []string
}

func (verbal Verbal) BuildAtkCommand() []string {
	verbal.parts = []string{"verbal", "-A", "-u"}
	if rand.Float32() < 0.5 {
		verbal.parts = append(verbal.parts, "http://127.0.0.1")
	} else {
		verbal.parts = append(verbal.parts, "https://127.0.0.1")
	}
	verbal.weight = 1
	return verbal.parts
}

func (verbal Verbal) Weight() int {
	return verbal.weight
}
