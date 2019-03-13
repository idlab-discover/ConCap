package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	atktools "gitlab.ilabt.imec.be/lpdhooge/containercap/attack-tools"
	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

func main() {
	fmt.Println("Containercap")
	var wg sync.WaitGroup
	files, err := ioutil.ReadDir("testcases")
	if err != nil {
		log.Fatal(err)
	}
	scenarios := make(chan scenario.Scenario)
	for _, file := range files {
		go func() {
			fh, err := os.Open(file.Name())
			defer fh.Close()
			if err != nil {
				log.Println("Couldn't read file", file.Name())
			}
			scn := scenario.ReadScenario(fh)
			ledger.Register(scn)
		}()
	}

	//ledger.Dump()
	podspec := scenario.PodTemplateBuilder(scn)
	kubeapi.CreatePod(podspec)
	podStates := make(chan bool, 100)
	go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
	for msg := range podStates {
		if msg {
			wg.Add(1)
			go func() {
				nmap := atktools.NewNmap()
				scn.Attacker.AtkCommand = strings.Join(nmap.BuildAtkCommand(), " ")
				fmt.Println("launched: ", scn.Attacker.AtkCommand)
				scn.StartTime = time.Now()
				kubeapi.ExecShellInContainer("default", scn.UUID.String(), scn.Attacker.Name, scn.Attacker.AtkCommand)
				kubeapi.DeletePod(scn.UUID.String())
				scn.StopTime = time.Now()
				scenario.WriteScenario(scn, file.Name())
				wg.Done()
			}()
			wg.Wait()
		} else {
			fmt.Print("Check again\n")
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
		}
	}

	// file, err = os.Open("scenario2-test.yaml")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// scn = scenario.ReadScenario(file)
	// ledger.Register(scn)
	// ledger.Dump()
	// podspec = scenario.PodTemplateBuilder(scn)
	// kubeapi.CreatePod(podspec)
	// scn.Attacker.AtkCommand = strings.Join(nmap.BuildAtkCommand(), " ")
	// fmt.Println("launched: ", scn.Attacker.AtkCommand)
	// scn.StartTime = time.Now()
	// kubeapi.ExecShellInContainer("default", scn.UUID.String(), scn.Attacker.Name, scn.Attacker.AtkCommand)
	// kubeapi.DeletePod(scn.UUID.String())
	// scn.StopTime = time.Now()
	// scenario.WriteScenario(scn, file.Name())
}
