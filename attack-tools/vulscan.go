package atktools

type Vulscan struct {
	weight int
	parts  []string
}

func (vulscan Vulscan) BuildAtkCommand() []string {
	vulscan.parts = []string{"nmap", "-sV", "--script=vulscan/vulscan.nse", "--script-args", "vulscanoutput=details", "localhost"}
	vulscan.weight = 1
	return vulscan.parts
}

func (vulscan Vulscan) Weight() int {
	return vulscan.weight
}
