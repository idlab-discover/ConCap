package atktools

type Enumshares struct {
	parts []string
}

func (enumshares Enumshares) BuildAtkCommand() []string {
	enumshares.parts = []string{"printf \"yes\n\"", "|", "enum-shares", "-w", "-t", "localhost", "-u", "root", "-p", "root"}
	return enumshares.parts
}
