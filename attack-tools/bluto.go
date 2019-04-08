package atktools

type Bluto struct {
	Weight       int
	parts        []string
	targetDomain string
}

func (bluto Bluto) BuildAtkCommand() []string {
	bluto.parts = []string{"bluto", "-e"}
	bluto.parts = append(bluto.parts, "-d", bluto.targetDomain)
	bluto.Weight = 1
	return bluto.parts
}
