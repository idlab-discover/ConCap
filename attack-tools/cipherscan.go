package atktools

type Cipherscan struct {
	weight int
	parts  []string
}

func (cipherscan Cipherscan) BuildAtkCommand() []string {
	cipherscan.parts = []string{"cipherscan", "-v", "ugent.be"}
	cipherscan.weight = 1
	return cipherscan.parts
}

func (cipherscan Cipherscan) Weight() int {
	return cipherscan.weight
}
