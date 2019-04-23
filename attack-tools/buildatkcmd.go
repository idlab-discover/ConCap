package atktools

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

var (
	topPorts   TopPorts
	topDomains TopDomains
)

var attackers = map[scenario.ScenarioType]map[string]AttackCommandBuilder{
	scenario.Scanning: map[string]AttackCommandBuilder{},
}

func init() {
	topPorts = *NewTopPorts()
	//topDomains = *NewTopDomains("filtered-1m.list")
}

type AttackCommandBuilder interface {
	BuildAtkCommand() []string
	Weight() int
	ScenarioType() scenario.ScenarioType
}

func fetchAttacker(category scenario.ScenarioType, name string) (*AttackCommandBuilder, error) {
	if val, ok := attackers[category][name]; ok {
		return &val, nil
	} else {
		var a AttackCommandBuilder
		switch name {
		case "xsstracer":
			a = NewXsstracer()
		case "topera":
			a = NewTopera()
		case "verbal":
			a = NewVerbal()
		case "a2sv":
			a = NewA2sv()
		case "allthevhosts":
			a = NewAllthevhosts()
		case "amass":
			a = NewAmass()
		case "netdomains":
			a = NewNetdomains()
		case "barmie":
			a = NewBarmie()
		case "cangibrina":
			a = NewCangibrina()
		case "corstest":
			a = NewCorstest()
		case "cipherscan":
			a = NewCipherscan()
		case "bluto":
			a = NewBluto()
		case "enumshares":
			a = NewEnumshares()
		case "laf":
			a = NewLaf()
		case "sslscan":
			a = NewSslscan()
		case "subfinder":
			a = NewSubfinder()
		case "subover":
			a = NewSubover()
		case "vane2":
			a = NewVane2()
		case "n3map":
			a = NewN3map()
		case "nmap":
			a = NewNmap()
		case "vulscan":
			a = NewVulscan()
		case "autonse":
			a = NewAutoNSE()
		case "zmap":
			a = NewZmap()
		default:
			return nil, errors.New("Attacker not recognized")
		}
		attackers[category][name] = a
		return &a, nil
	}
}

func SelectAttacker(category scenario.ScenarioType, name string) *AttackCommandBuilder {
	val, err := fetchAttacker(category, name)
	if err != nil {
		log.Fatalln(err)
	}
	return val
}

func SelectAttackers(category scenario.ScenarioType) *map[string]AttackCommandBuilder {
	if val, found := attackers[category]; found {
		return &val
	}
	return &map[string]AttackCommandBuilder{}
}

type TopDomains struct {
	domains []string
}

func NewTopDomains(domainlist string) *TopDomains {
	file, err := os.Open(domainlist)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	td := TopDomains{
		domains: lines,
	}
	return &td
}

type CustomDomains struct {
	domains []string
}

type TopPorts struct {
	TCP []uint32
	UDP []uint32
}

func NewTopPorts() *TopPorts {
	return &TopPorts{
		TCP: []uint32{7, 9, 13, 21, 22, 23, 25, 26, 37, 53, 79, 80, 81, 88, 106, 110, 111, 113, 119, 135, 139, 143, 144, 179, 199, 389, 427, 443, 444, 445, 465, 513, 514, 515, 543, 544, 548, 554, 587, 631, 646, 873, 990, 993, 995, 1025, 1026, 1027, 1028, 1029, 1110, 1433, 1720, 1723, 1755, 1900, 2000, 2001, 2049, 2121, 2717, 3000, 3128, 3306, 3389, 3986, 4899, 5000, 5009, 5051, 5060, 5101, 5190, 5357, 5432, 5631, 5666, 5800, 5900, 6000, 6001, 6646, 7070, 8000, 8008, 8009, 8080, 8081, 8443, 8888, 9100, 9999, 10000, 32768, 49152, 49153, 49154, 49155, 49156, 49157},
		UDP: []uint32{7, 9, 17, 19, 49, 53, 67, 68, 69, 80, 88, 111, 120, 123, 135, 136, 137, 138, 139, 158, 161, 162, 177, 427, 443, 445, 497, 500, 514, 515, 518, 520, 593, 623, 626, 631, 996, 997, 998, 999, 1022, 1023, 1025, 1026, 1027, 1029, 1030, 1433, 1434, 1645, 1646, 1701, 1718, 1719, 1812, 1813, 1900, 2000, 2048, 2049, 2222, 2223, 3283, 3456, 3703, 4444, 4500, 5000, 5060, 5353, 5632, 9200, 10000, 17185, 20031, 30718, 31337, 32768, 32769, 32771, 32815, 33281, 49152, 49153, 49154, 49156, 49181, 49182, 49185, 49186, 49188, 49190, 49191, 49192, 49193, 49194, 49200, 49201, 65024},
	}
}
