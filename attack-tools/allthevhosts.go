package atktools

type Allthevhosts struct {
	weight int
	parts  []string
}

func (allthevhosts Allthevhosts) BuildAtkCommand() []string {
	allthevhosts.parts = []string{"allthevhosts", "127.0.0.1"}
	allthevhosts.weight = 1
	return allthevhosts.parts
}
