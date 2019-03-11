package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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
	scn.StartTime = time.Now()
	kubeapi.ExecShellInContainer("default", scn.UUID.String(), "nmap", scn.Attacker.AtkCommand)
	kubeapi.DeletePod(scn.UUID.String())
	scn.StopTime = time.Now()
	scenario.WriteScenario(scn, file.Name())

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
	scn.StartTime = time.Now()
	kubeapi.ExecShellInContainer("default", scn.UUID.String(), "nmap", scn.Attacker.AtkCommand)
	kubeapi.DeletePod(scn.UUID.String())
	scn.StopTime = time.Now()
	scenario.WriteScenario(scn, file.Name())

	//kubeapi.WatchPod()
	//kubeapi.ListPod()
	//kubeapi.UpdatePod()
}
