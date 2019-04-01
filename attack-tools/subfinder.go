package atktools

type Subfinder struct {
	parts []string
}

func (subfinder Subfinder) BuildAtkCommand() []string {
	subfinder.parts = []string{"subfinder", "-t 25", "-r 8.8.8.8,1.1.1.1", "-d", "ugent.be"}
	return subfinder.parts
}
