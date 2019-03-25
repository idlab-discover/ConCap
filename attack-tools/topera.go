package atktools

import "math/rand"

type Topera struct {
	parts, modes []string
}

func NewTopera() Topera {
	topera := Topera{}
	topera.modes = []string{"topera_tcp_scan", "topera_loris"}
	return topera
}

func (topera Topera) BuildAtkCommand() []string {
	topera.parts = []string{"topera", "-t ::1", "-M"}
	topera.parts = append(topera.parts, topera.modes[rand.Intn(len(topera.modes))])
	return topera.parts
}
