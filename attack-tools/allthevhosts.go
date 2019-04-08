package atktools

type Allthevhosts struct {
	Weight int
	parts  []string
}

func (allthevhosts Allthevhosts) BuildAtkCommand() []string {
	allthevhosts.parts = []string{"allthevhosts", "127.0.0.1"}
	allthevhosts.Weight = 1
	return allthevhosts.parts
}
