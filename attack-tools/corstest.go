package atktools

type Corstest struct {
	weight int
	parts  []string
}

func (corstest Corstest) BuildAtkCommand() []string {
	corstest.parts = []string{"corstest", "-v"}
	corstest.weight = 1
	return corstest.parts
}
