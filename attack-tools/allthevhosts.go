package atktools

type Allthevhosts struct {
	parts []string
}

func (allthevhosts Allthevhosts) BuildAtkCommand() []string {
	allthevhosts.parts = []string{"allthevhosts", "127.0.0.1"}
	return allthevhosts.parts
}
