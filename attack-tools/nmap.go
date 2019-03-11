package atktools

import "math/rand"

type Nmap struct {
	discovery, port, scanTypes, srvDetect, osDetect, timing, script, evasion, parts []string
}

func NewNmap() Nmap {
	nmap := Nmap{}
	nmap.discovery = []string{"-sL", "-sn", "-Pn", "-PU", "PR"}
	nmap.port = []string{"--top-ports 50", "--top-ports 100", "--top-ports 500", "--top-ports 1000", "--top-ports 2000", "-p-"}
	nmap.scanTypes = []string{"-sS", "-sT", "-sU", "-sA", "-sW", "-sW", "-SM"}
	nmap.srvDetect = []string{"-A", "-sV", "-sV --version-light", "-sV --version-all"}
	nmap.osDetect = []string{"-O", "-O --osscan-limit", "-O --osscan-guess"}
	nmap.timing = []string{"-T3", "-T4", "-T5"}
	nmap.script = []string{"-sC", "--script default", "--script=http*", "--script \"not intrusive\""}
	nmap.evasion = []string{"-f", "--mtu 32", "--mtu 512", "-g 53", "-g 8080", "--data-length 200"}
	return nmap
}

func (nmap Nmap) BuildAtkCommand() []string {
	nmap.parts = []string{"nmap", "localhost"}
	if rand.Float32() < 0.05 {
		nmap.parts = append(nmap.parts, nmap.discovery[rand.Intn(len(nmap.discovery))])
		return nmap.parts
	}

	nmap.parts = append(nmap.parts, nmap.scanTypes[rand.Intn(len(nmap.scanTypes))])
	if rand.Float32() < 0.33 {
		nmap.parts = append(nmap.parts, nmap.port[rand.Intn(len(nmap.port))])
	}
	if rand.Float32() < 0.33 {
		nmap.parts = append(nmap.parts, nmap.srvDetect[rand.Intn(len(nmap.srvDetect))])
	}
	if rand.Float32() < 0.33 {
		nmap.parts = append(nmap.parts, nmap.osDetect[rand.Intn(len(nmap.osDetect))])
	}
	nmap.parts = append(nmap.parts, nmap.timing[rand.Intn(len(nmap.timing))])
	// if rand.Float32() < 0.33 {
	// 	parts = append(parts, script[rand.Intn(len(script))])
	// }
	if rand.Float32() < 0.33 {
		nmap.parts = append(nmap.parts, nmap.evasion[rand.Intn(len(nmap.evasion))])
	}
	return nmap.parts
}
