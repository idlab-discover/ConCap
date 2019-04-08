package atktools

import "math/rand"

type A2sv struct {
	weight  int
	parts   []string
	modules []string
}

func NewA2sv() A2sv {
	a2sv := A2sv{}
	a2sv.weight = 10
	a2sv.modules = []string{"anonymous", "crime", "heart", "ccs", "poodle", "freak", "logjam", "drown"}
	return a2sv
}

func (a2sv A2sv) BuildAtkCommand() []string {
	a2sv.parts = []string{"a2sv", "-t", "127.0.0.1", "-o", "Y", "-m"}
	a2sv.parts = append(a2sv.parts, a2sv.modules[rand.Intn(len(a2sv.modules))])
	return a2sv.parts
}

func (a2sv A2sv) Weight() int {
	return a2sv.weight
}
