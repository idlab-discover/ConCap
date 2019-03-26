package atktools

var topPorts TopPorts

func init() {
	topPorts = NewTopPorts()
}

func SelectAttacker(name string) AttackCommandBuilder {
	switch name {
	case "xsstracer":
		return Xsstracer{}
	case "topera":
		return NewTopera()
	case "verbal":
		return Verbal{}
	case "a2sv":
		return NewA2sv()
	case "allthevhosts":
		return Allthevhosts{}
	case "amass":
		return NewAmass()
	case "netdomains":
		return NewNetdomains()
	case "barmie":
		return Barmie{}
	case "cangibrina":
		return Cangibrina{}
	case "corstest":
		return Corstest{}
	case "cipherscan":
		return Cipherscan{}
	case "bluto":
		return Bluto{}
	case "n3map":
		return NewN3Map()
	case "nmap":
		return NewNmap()
	case "vulscan":
		return Vulscan{}
	case "autonse":
		return AutoNSE{}
	case "zmap":
		return NewZmap()
	default:
		return nil
	}
}

type AttackCommandBuilder interface {
	BuildAtkCommand() []string
}

type TopPorts struct {
	TCP []uint32
	UDP []uint32
}

func NewTopPorts() TopPorts {
	return TopPorts{
		TCP: []uint32{7, 9, 13, 21, 22, 23, 25, 26, 37, 53, 79, 80, 81, 88, 106, 110, 111, 113, 119, 135, 139, 143, 144, 179, 199, 389, 427, 443, 444, 445, 465, 513, 514, 515, 543, 544, 548, 554, 587, 631, 646, 873, 990, 993, 995, 1025, 1026, 1027, 1028, 1029, 1110, 1433, 1720, 1723, 1755, 1900, 2000, 2001, 2049, 2121, 2717, 3000, 3128, 3306, 3389, 3986, 4899, 5000, 5009, 5051, 5060, 5101, 5190, 5357, 5432, 5631, 5666, 5800, 5900, 6000, 6001, 6646, 7070, 8000, 8008, 8009, 8080, 8081, 8443, 8888, 9100, 9999, 10000, 32768, 49152, 49153, 49154, 49155, 49156, 49157},
		UDP: []uint32{7, 9, 17, 19, 49, 53, 67, 68, 69, 80, 88, 111, 120, 123, 135, 136, 137, 138, 139, 158, 161, 162, 177, 427, 443, 445, 497, 500, 514, 515, 518, 520, 593, 623, 626, 631, 996, 997, 998, 999, 1022, 1023, 1025, 1026, 1027, 1029, 1030, 1433, 1434, 1645, 1646, 1701, 1718, 1719, 1812, 1813, 1900, 2000, 2048, 2049, 2222, 2223, 3283, 3456, 3703, 4444, 4500, 5000, 5060, 5353, 5632, 9200, 10000, 17185, 20031, 30718, 31337, 32768, 32769, 32771, 32815, 33281, 49152, 49153, 49154, 49156, 49181, 49182, 49185, 49186, 49188, 49190, 49191, 49192, 49193, 49194, 49200, 49201, 65024},
	}
}
