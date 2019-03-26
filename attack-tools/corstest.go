package atktools

type Corstest struct {
	parts []string
}

func (corstest Corstest) BuildAtkCommand() []string {
	corstest.parts = []string{"corstest", "-v"}
	return corstest.parts
}
