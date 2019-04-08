package atktools

type Cipherscan struct {
	Weight int
	parts  []string
}

func (cipherscan Cipherscan) BuildAtkCommand() []string {
	cipherscan.parts = []string{"cipherscan", "-v", "ugent.be"}
	cipherscan.Weight = 1
	return cipherscan.parts
}
