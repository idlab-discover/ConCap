package scenariobuilder

import (
	"flag"

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
	atks, _ := selectTools(*category, *tool)
	scn := generateScenario()

}

func selectTools(category, tool string) (map[atktools.AttackCommandBuilder]int, error) {
	var selection = map[atktools.AttackCommandBuilder]int{}
	if category != "" && tool != "" {
		attacker := *atktools.SelectAttacker(category, tool)
		selection[attacker] = attacker.Weight()
	} else if category != "" && tool == "" {
		attackers := *atktools.SelectAttackers(category)
		for _, v := range attackers {
			selection[v] = v.Weight()
		}
	}
	return selection, nil
}

func generateScenario(typ scenario.ScenarioType) *scenario.Scenario {
	S := scenario.Scenario{}
	S.UUID = uuid.New()
	S.ScenarioType = typ

	return &S
}
