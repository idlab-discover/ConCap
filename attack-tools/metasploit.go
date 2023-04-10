package atktools

import (
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Metasploit struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

func NewMetasploit() Metasploit {
	metasploit := Metasploit{weight: 50, scenarioType: scenario.Scanning}
	return metasploit
}

func (metasploit Metasploit) BuildAtkCommand() []string {
	metasploit.parts = []string{"msfconsole"}
	metasploit.parts = append(metasploit.parts, "-q")
	metasploit.parts = append(metasploit.parts, "-x")
	metasploit.parts = append(metasploit.parts, "'")
	metasploit.parts = append(metasploit.parts, "load db_autopwn;")
	metasploit.parts = append(metasploit.parts, "workspace -a Case01;")
	metasploit.parts = append(metasploit.parts, "workspace Case01;")
	metasploit.parts = append(metasploit.parts, "db_nmap")
	metasploit.parts = append(metasploit.parts, "localhost")
	metasploit.parts = append(metasploit.parts, "-p- -sV -O;")
	metasploit.parts = append(metasploit.parts, "db_autopwn -p -R great -e -q")
	metasploit.parts = append(metasploit.parts, "127.0.0.1;")
	metasploit.parts = append(metasploit.parts, "exit -y '")
	metasploit.parts = append(metasploit.parts, "> /Containercap/containercap-logs/metasploit-logs.txt")

	return metasploit.parts

}

func (metasploit Metasploit) Weight() int {
	return metasploit.weight
}

func (metasploit Metasploit) ScenarioType() scenario.ScenarioType {
	return metasploit.scenarioType
}
