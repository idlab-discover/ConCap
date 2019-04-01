package atktools

type Vane2 struct {
	parts []string
}

func (vane2 Vane2) BuildAtkCommand() []string {
	vane2.parts = []string{"vane import-data; vane scan -pv --url http://chefilan.com/blog/"}
	return vane2.parts
}
