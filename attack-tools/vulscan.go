package atktools

type Vulscan struct {
	Weight int
	parts  []string
}

func (vulscan Vulscan) BuildAtkCommand() []string {
	vulscan.parts = []string{"nmap", "-sV", "--script=vulscan/vulscan.nse", "--script-args", "vulscanoutput=details", "localhost"}
	vulscan.Weight = 1
	return vulscan.parts
}
