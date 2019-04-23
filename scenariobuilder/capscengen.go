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
	category  = flag.String("cat", "", "Should we build scenarios for a collection of tools from a specific attack type")
)

func main() {
	flag.Parse()
	targets, _ := selectTargets("webserver", "X")
	for i := 0; i < *count; i++ {
		scn := generateScenario()
		selected, _ := selectAttackers(scenario.ScenarioType(*category), *tool)
		for _, v := range selected {
			scn.Attacker = v
		}
		scn.Target = targets[i%5]
		scn.CaptureEngine = scenario.CaptureEngine{Name: "tcpdump", Image: "gitlab.ilabt.imec.be:4567/lpdhooge/containercap-imagery/tcpdump:latest", Interface: "lo", Filter: ""}
		scn.ScenarioType = scn.Attacker.Category
		scn.Tag = ""
		scenario.WriteScenario(scn, "testcases/"+scn.UUID.String()+".yaml")
	}
}

func selectAttackers(category scenario.ScenarioType, tool string) ([]scenario.Attacker, error) {
	var selection []scenario.Attacker
	if category != "" && tool != "" {
		attacker := *atktools.SelectAttacker(category, tool)
		selection = append(selection, scenario.Attacker{
			Category:   category,
			Name:       tool,
			Image:      "gitlab.ilabt.imec.be:4567/lpdhooge/containercap-imagery/" + tool + ":latest",
			AtkCommand: strings.Join(attacker.BuildAtkCommand(), " "),
		})

	} else if category != "" && tool == "" {
		//attackers := *atktools.SelectAttackers(category)
	}
	return selection, nil
}

func selectTargets(category, target string) ([]scenario.Target, error) {
	return []scenario.Target{
		{Category: "webserver", Name: "httpd", Image: "httpd:2.4.38", Ports: []int32{8080}},
		{Category: "webserver", Name: "nginx", Image: "nginx:1:12", Ports: []int32{80}},
		{Category: "webserver", Name: "node", Image: "node:latest", Ports: []int32{21}},
		{Category: "webserver", Name: "jetty", Image: "jetty:latest", Ports: []int32{22}},
		{Category: "webserver", Name: "tomcat", Image: "tomcat:latest", Ports: []int32{8081}},
	}, nil
}

func generateScenario() *scenario.Scenario {
	S := scenario.Scenario{}
	S.UUID = uuid.New()

	return &S
}
