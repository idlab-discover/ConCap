package atktools

type Bluto struct {
	weight       int
	parts        []string
	targetDomain string
}

func (bluto Bluto) BuildAtkCommand() []string {
	bluto.parts = []string{"bluto", "-e"}
	bluto.parts = append(bluto.parts, "-d", bluto.targetDomain)
	bluto.weight = 1
	return bluto.parts
}

func (bluto Bluto) Weight() int {
	return bluto.weight
}
