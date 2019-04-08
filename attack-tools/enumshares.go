package atktools

type Enumshares struct {
	Weight int
	parts  []string
}

func (enumshares Enumshares) BuildAtkCommand() []string {
	enumshares.parts = []string{"printf \"yes\n\"", "|", "enum-shares", "-w", "-t", "localhost", "-u", "root", "-p", "root"}
	enumshares.Weight = 1
	return enumshares.parts
}
