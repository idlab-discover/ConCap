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

func loadScenarios(filename string, scns chan *scenario.Scenario, wg *sync.WaitGroup) {
	defer wg.Done()
	fh, err := os.Open("testcases/" + filename)
	defer fh.Close()
	if err != nil {
		log.Println("Couldn't read file", filename)
	}
	scn := scenario.ReadScenario(fh)
	ledger.Register(scn)
	scns <- scn
	fmt.Println("loaded scenario onto channel")
}

func main() {
	fmt.Println("Containercap")
	files, err := ioutil.ReadDir("testcases")
	fmt.Println("Number of files read", len(files))
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	scenarios := make(chan *scenario.Scenario, len(files))

	for _, file := range files {
		wg.Add(1)
		fmt.Println("Fx:", file.Name())
		go loadScenarios(file.Name(), scenarios, &wg)
	}
	wg.Wait()
	close(scenarios)

	for scene := range scenarios {
		wg.Add(1)
		go func(scn *scenario.Scenario, wg *sync.WaitGroup) {
			defer wg.Done()
			podspec := scenario.PodTemplateBuilder(scn)
			kubeapi.CreatePod(podspec)
			podStates := make(chan bool, 100)
			go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
			for msg := range podStates {
				if msg {
					nmap := atktools.NewNmap()
					scn.Attacker.AtkCommand = strings.Join(nmap.BuildAtkCommand(), " ")
					fmt.Println("launched: ", scn.Attacker.AtkCommand)
					scn.StartTime = time.Now()
					kubeapi.ExecShellInContainer("default", scn.UUID.String(), scn.Attacker.Name, scn.Attacker.AtkCommand)
					kubeapi.DeletePod(scn.UUID.String())
					scn.StopTime = time.Now()
					//scenario.WriteScenario(scn, "testcases/"+scn.UUID.String()+".yaml")
				} else {
					fmt.Print("Check again\n")
					time.Sleep(10 * time.Second)
					go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
				}
			}
		}(scene, &wg)
	}
	fmt.Println("Waiting for wait group to end")
	wg.Wait()
}
