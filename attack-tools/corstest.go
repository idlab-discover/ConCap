package atktools

type Corstest struct {
	Weight int
	parts  []string
}

func (corstest Corstest) BuildAtkCommand() []string {
	corstest.parts = []string{"corstest", "-v"}
	corstest.Weight = 1
	return corstest.parts
}
