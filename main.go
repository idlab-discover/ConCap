package main

import (
	"fmt"
	"log"
	"os"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

func main() {
	fmt.Println("Containercap")
	file, err := os.Open("scenario-test.yaml")
	if err != nil {
		log.Fatal(err)
	}
	scn := scenario.BuildScenario(file)
	ledger.Register(scn)
	ledger.Dump()
	podspec := scenario.PodTemplateBuilder(scn)
	kubeapi.CreatePod(podspec)

	var nmap atk.Nmap
	scn.Attacker.AtkCommand = nmap.BuildAtkCommand()
	kubeapi.ExecShellInContainer("default", scn.UUID.String(), "nmap", "nmap -sS -A -T5 localhost")
	kubeapi.DeletePod(scn.UUID.String())

	file, err = os.Open("scenario2-test.yaml")
	if err != nil {
		log.Fatal(err)
	}
	scn = scenario.BuildScenario(file)
	ledger.Register(scn)
	ledger.Dump()
	podspec = scenario.PodTemplateBuilder(scn)
	kubeapi.CreatePod(podspec)
	kubeapi.ExecShellInContainer("default", scn.UUID.String(), "nmap", "nmap -sS -A -T5 localhost")

	//kubeapi.WatchPod()
	//kubeapi.ListPod()
	//kubeapi.UpdatePod()
	kubeapi.DeletePod(scn.UUID.String())
}
