package atktools

type Cipherscan struct {
	parts []string
}

func (cipherscan Cipherscan) BuildAtkCommand() []string {
	cipherscan.parts = []string{"cipherscan", "-v", "ugent.be"}
	return cipherscan.parts
}
