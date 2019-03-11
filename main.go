package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	atktools "gitlab.ilabt.imec.be/lpdhooge/containercap/attack-tools"
	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

func main() {
	fmt.Println("Containercap")
	file, err := os.Open("scenario-test.yaml")
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	scn := scenario.ReadScenario(file)
	ledger.Register(scn)
	ledger.Dump()
	podspec := scenario.PodTemplateBuilder(scn)
	kubeapi.CreatePod(podspec)

	nmap := atktools.NewNmap()
	scn.Attacker.AtkCommand = strings.Join(nmap.BuildAtkCommand(), " ")
	fmt.Println("launched: ", scn.Attacker.AtkCommand)
	kubeapi.ExecShellInContainer("default", scn.UUID.String(), "nmap", scn.Attacker.AtkCommand)
	kubeapi.DeletePod(scn.UUID.String())

	file, err = os.Open("scenario2-test.yaml")
	if err != nil {
		log.Fatal(err)
	}
	scn = scenario.ReadScenario(file)
	ledger.Register(scn)
	ledger.Dump()
	podspec = scenario.PodTemplateBuilder(scn)
	kubeapi.CreatePod(podspec)
	scn.Attacker.AtkCommand = strings.Join(nmap.BuildAtkCommand(), " ")
	fmt.Println("launched: ", scn.Attacker.AtkCommand)
	kubeapi.ExecShellInContainer("default", scn.UUID.String(), "nmap", scn.Attacker.AtkCommand)
	kubeapi.DeletePod(scn.UUID.String())

	//kubeapi.WatchPod()
	//kubeapi.ListPod()
	//kubeapi.UpdatePod()
}
