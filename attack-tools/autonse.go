package atktools

type AutoNSE struct {
	parts []string
}

func (autonse AutoNSE) BuildAtkCommand() []string {
	autonse.parts = []string{"printf", "\"n\nlocalhost\n\"", "|", "autonse"}
	return autonse.parts
}
