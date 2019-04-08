package atktools

type Vane2 struct {
	weight int
	parts  []string
}

func (vane2 Vane2) BuildAtkCommand() []string {
	// TODO wordpress domains, preferably vulnerable
	vane2.parts = []string{"vane import-data; vane scan -pv --url http://chefilan.com/blog/"}
	vane2.weight = 1
	return vane2.parts
}
