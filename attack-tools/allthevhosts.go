package atktools

import "gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"

// Allthevhosts https://github.com/sidaf/scripts/blob/master/allthevhosts.py
// There are a number of ways to own a webapp. In a shared environment, an attacker can enumerate all the applications accessible and target the weakest one to root the server and with it all the webapps on the box.
// To try and emulate this approach on a pentest, we have to find ALL THE VHOSTS.
type Allthevhosts struct {
	scenarioType scenario.ScenarioType
	weight       int
	parts        []string
}

// NewAllthevhosts sets basic information for this tool
func NewAllthevhosts() Allthevhosts {
	allthevhosts := Allthevhosts{weight: 1, scenarioType: scenario.Scanning}
	return allthevhosts
}

// BuildAtkCommand generates the default command for allthevhosts to use
func (allthevhosts Allthevhosts) BuildAtkCommand() []string {
	allthevhosts.parts = []string{"allthevhosts", "127.0.0.1"}
	return allthevhosts.parts
}

func (allthevhosts Allthevhosts) Weight() int {
	return allthevhosts.weight
}

func (allthevhosts Allthevhosts) ScenarioType() scenario.ScenarioType {
	return allthevhosts.scenarioType
}
