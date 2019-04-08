package atktools

type Enumshares struct {
	weight int
	parts  []string
}

func (enumshares Enumshares) BuildAtkCommand() []string {
	enumshares.parts = []string{"printf \"yes\n\"", "|", "enum-shares", "-w", "-t", "localhost", "-u", "root", "-p", "root"}
	enumshares.weight = 1
	return enumshares.parts
}

func (enumshares Enumshares) Weight() int {
	return enumshares.weight
}
