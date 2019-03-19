package atktools

type Vulscan struct {
	parts []string
}

func (vulscan Vulscan) BuildAtkCommand() []string {
	vulscan.parts = []string{"nmap", "-sV", "--script=vulscan/vulscan.nse", "--script-args", "vulscanoutput=details", "localhost"}
	return vulscan.parts
}
