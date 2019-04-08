package atktools

type Subfinder struct {
	Weight int
	parts  []string
}

func (subfinder Subfinder) BuildAtkCommand() []string {
	// TODO domain expansion
	subfinder.parts = []string{"subfinder", "-t 25", "-r 8.8.8.8,1.1.1.1", "-d", "ugent.be"}
	subfinder.Weight = 1
	return subfinder.parts
}
