package scenariobuilder

import (
	"flag"

	atktools "gitlab.ilabt.imec.be/lpdhooge/containercap/attack-tools"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

var (
	count     = flag.Int("c", 0, "How many scenarios need to be made?")
	verbose   = flag.Bool("v", false, "Verbose output")
	outfolder = flag.String("o", "./testcases/", "Where should the scenarios be stored?")
	tool      = flag.String("t", "", "Should we create scenarios for a specific tool?")
	category  = flag.String("c", "", "Should we build scenarios for a collection of tools from a specific attack type")
)

func main() {
	flag.Parse()

}

func selectTools(category, name string) (map[atktools.AttackCommandBuilder]int, error) {
	var selection = map[atktools.AttackCommandBuilder]int{}
	if category != "" && name != "" {
		selection[atktools.SelectAttacker(category, name)] = 
	}

}

func generateScenario(scenarioType string) *scenario.Scenario {
	S := scenario.Scenario{}
	return &S
}
