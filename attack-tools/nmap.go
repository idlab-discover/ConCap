package atk

import "math/rand"

type Nmap struct{}

func (nmap Nmap) BuildAtkCommand() []string {
	discovery := [...]string{"-sL", "-sn", "-Pn", "-PU", "PR"}
	port := [...]string{"--top-ports 50", "--top-ports 100", "--top-ports 500", "--top-ports 1000", "--top-ports 2000", "-p-"}
	scanTypes := [...]string{"-sS", "-sT", "-sU", "-sA", "-sW", "-sW", "-SM"}
	srvDetect := [...]string{"-A", "-sV", "-sV --version-light", "-sV --version-all"}
	osDetect := [...]string{"-O", "-O --osscan-limit", "-O --osscan-guess"}
	timing := [...]string{"-T3", "-T4", "-T5"}
	//script := [...]string{"-sC", "--script default", "--script=http*", "--script \"not intrusive\""}
	evasion := [...]string{"-f", "--mtu 32", "--mtu 512", "-g 53", "-g 8080", "--data-length 200"}

	parts := []string{"nmap", "localhost"}
	if rand.Float32() < 0.05 {
		parts = append(parts, discovery[rand.Intn(len(discovery))])
		return parts
	}

	parts = append(parts, scanTypes[rand.Intn(len(scanTypes))])
	if rand.Float32() < 0.33 {
		parts = append(parts, port[rand.Intn(len(port))])
	}
	if rand.Float32() < 0.33 {
		parts = append(parts, srvDetect[rand.Intn(len(srvDetect))])
	}
	if rand.Float32() < 0.33 {
		parts = append(parts, osDetect[rand.Intn(len(osDetect))])
	}
	parts = append(parts, timing[rand.Intn(len(timing))])
	// if rand.Float32() < 0.33 {
	// 	parts = append(parts, script[rand.Intn(len(script))])
	// }
	if rand.Float32() < 0.33 {
		parts = append(parts, evasion[rand.Intn(len(evasion))])
	}
	return parts
}
