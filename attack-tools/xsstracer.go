package atktools

import "strconv"

type Xsstracer struct {
	parts      []string
	targetPort int
}

func (xsstracer Xsstracer) BuildAtkCommand() []string {
	xsstracer.parts = []string{"xsstracer", "127.0.0.1", strconv.Itoa(xsstracer.targetPort)}
	return xsstracer.parts
}
