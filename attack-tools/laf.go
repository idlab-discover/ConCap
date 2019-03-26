package atktools

type Laf struct {
	parts, sys []string
}

func NewLaf() Laf {
	laf := Laf{}
	laf.sys = []string{"dirs", "php", "cfm", "asp", "pl", "html", "pma"}
	return laf
}

// todo add port specification via host:port notation which works
func (laf Laf) BuildAtkCommand() []string {
	laf.parts = []string{"laf", "-d", "localhost", "-u", "admin", "-p", "admin"}
	return laf.parts
}
