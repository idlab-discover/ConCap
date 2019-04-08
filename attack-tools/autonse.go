package atktools

type AutoNSE struct {
	Weight int
	parts  []string
}

func (autonse AutoNSE) BuildAtkCommand() []string {
	autonse.parts = []string{"printf", "\"n\nlocalhost\n\"", "|", "autonse"}
	autonse.Weight = 1
	return autonse.parts
}
