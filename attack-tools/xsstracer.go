package atktools

import "strconv"

type Xsstracer struct {
	weight     int
	parts      []string
	targetPort int
}

func (xsstracer Xsstracer) BuildAtkCommand() []string {
	xsstracer.parts = []string{"xsstracer", "127.0.0.1", strconv.Itoa(xsstracer.targetPort)}
	xsstracer.weight = 1
	return xsstracer.parts
}
