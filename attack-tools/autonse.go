package atktools

type AutoNSE struct {
	weight int
	parts  []string
}

func (autonse AutoNSE) BuildAtkCommand() []string {
	autonse.parts = []string{"printf", "\"n\nlocalhost\n\"", "|", "autonse"}
	autonse.weight = 1
	return autonse.parts
}

func (autonse AutoNSE) Weight() int {
	return autonse.weight
}
