package atktools

type Vane2 struct {
	Weight int
	parts  []string
}

func (vane2 Vane2) BuildAtkCommand() []string {
	// TODO wordpress domains, preferably vulnerable
	vane2.parts = []string{"vane import-data; vane scan -pv --url http://chefilan.com/blog/"}
	vane2.Weight = 1
	return vane2.parts
}
