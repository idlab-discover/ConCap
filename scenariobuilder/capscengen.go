package scenariobuilder

import (
	"flag"
	"strings"

	"github.com/google/uuid"
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
	selected, _ := selectTools(scenario.ScenarioType(*category), *tool)
	for i := range selected {

	}
	scn := generateScenario()

}

func selectTools(category scenario.ScenarioType, tool string) ([]scenario.Attacker, error) {
	var selection []scenario.Attacker
	if category != "" && tool != "" {
		attacker := *atktools.SelectAttacker(category, tool)
		selection = append(selection, scenario.Attacker{
			Category:   category,
			Name:       tool,
			AtkCommand: strings.Join(attacker.BuildAtkCommand(), " "),
		})

	} else if category != "" && tool == "" {
		attackers := *atktools.SelectAttackers(category)

	}
	return selection, nil
}

func generateScenario() *scenario.Scenario {
	S := scenario.Scenario{}
	S.UUID = uuid.New()

	return &S
}
