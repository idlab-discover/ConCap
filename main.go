package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()
	sugar = logger.Sugar()
}

func loadScenarios(filename string, scns chan *scenario.Scenario, wg *sync.WaitGroup) {
	defer wg.Done()
	fh, err := os.Open("autogen-cases/" + filename)
	defer fh.Close()
	if err != nil {
		log.Println("Couldn't read file", filename)
	}
	scn := scenario.ReadScenario(fh)
	scn.UUID, err = uuid.Parse(strings.Split(filename, ".")[0])
	if err != nil {
		log.Println("File had incorrect UUID filename")
	}
	ledger.Register(scn)
	scns <- scn
	fmt.Println("loaded scenario onto channel")
}

func startScenario(scn *scenario.Scenario, wg *sync.WaitGroup) {
	defer wg.Done()
	podspec := scenario.ScenarioPod(scn)
	kubeapi.CreatePod(podspec)
	ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: "CREATED", Scenario: scn})
	podStates := make(chan bool, 100)
	go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
	for msg := range podStates {
		if msg {
			//attacker := *atktools.SelectAttacker(scn.Attacker.Category, scn.Attacker.Name)
			//scn.Attacker.AtkCommand = strings.Join(attacker.BuildAtkCommand(), " ")
			fmt.Println("launched: ", scn.Attacker.AtkCommand)
			scn.StartTime = time.Now()
			ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: "IN PROGRESS", Scenario: scn})
			kubeapi.ExecShellInContainer("default", scn.UUID.String(), scn.Attacker.Name, scn.Attacker.AtkCommand)
			kubeapi.DeletePod(scn.UUID.String())
			ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: "COMPLETED", Scenario: scn})
			scn.StopTime = time.Now()
			scenario.WriteScenario(scn, "autogen-cases/"+scn.UUID.String()+".yaml")
		} else {
			fmt.Print("Check again\n")
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
		}
	}
}

func flowProcessing() {
	podspecJoy := scenario.FlowProcessPod("joy")
	kubeapi.CreatePod(podspecJoy)
	podspecCIC := scenario.FlowProcessPod("cicflowmeter")
	kubeapi.CreatePod(podspecCIC)
	podspecYaf := scenario.FlowProcessPod("yaf")
	kubeapi.CreatePod(podspecYaf)

}

func main() {
	fmt.Println("Containercap")
	files, err := ioutil.ReadDir("autogen-cases")
	fmt.Println("Number of files read", len(files))
	if err != nil {
		log.Fatal(err)
	}

	scenarios := make(chan *scenario.Scenario, len(files))

	go func() {
		defer close(scenarios)
		var wgReadExp sync.WaitGroup
		for _, file := range files {
			wgReadExp.Add(1)
			fmt.Println("Fx:", file.Name())
			go loadScenarios(file.Name(), scenarios, &wgReadExp)
		}
		wgReadExp.Wait()
		fmt.Printf("\033[2K\r")
		ledger.Repr()
	}()

	var wgExecExp sync.WaitGroup
	for scene := range scenarios {
		wgExecExp.Add(1)
		go startScenario(scene, &wgExecExp)
	}
	wgExecExp.Wait()

}
