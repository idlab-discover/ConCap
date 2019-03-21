package atktools

type Bluto struct {
	parts        []string
	targetDomain string
}

func (bluto Bluto) BuildAtkCommand() []string {
	bluto.parts = []string{"bluto", "-e"}
	bluto.parts = append(bluto.parts, "-d", bluto.targetDomain)
	return bluto.parts
}
