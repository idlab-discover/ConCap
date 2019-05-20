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
	fh, err := os.Open("/home/dhoogla/PhD/containercap-scenarios" + filename)
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
	podStates := make(chan bool, 64)
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
			scenario.WriteScenario(scn, "/home/dhoogla/Phd/containercap-scenarios/"+scn.UUID.String()+".yaml")
		} else {
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
		}
	}
}

func joyProcessing(scenarioUUID string) {
	kubeapi.ExecShellInContainer("default", "joy", "joy",
		"./joy retain=1 bidir=1 num_pkts=200 dist=1 cdist=none entropy=1 wht=0 example=0 dns=1 ssh=1 tls=1 dhcp=1 dhcpv6=1 http=1 ike=1 payload=1 salt=0 ppi=0 fpx=0 verbosity=4 threads=4 "+scenarioUUID+"	> /tmp/containercap-transformed/"+scenarioUUID+".joy")
}

func cicProcessing(scenarioUUID string) {
	kubeapi.ExecShellInContainer("default", "cicflowmeter", "cicflowmeter", "./cfm /tmp/containercap-captures/"+scenarioUUID+" /tmp/containercap-transformed/"+scenarioUUID)
}

func main() {
	fmt.Println("Containercap")
	podspecJoy := scenario.FlowProcessPod("joy")
	kubeapi.CreatePod(podspecJoy)
	podspecCIC := scenario.FlowProcessPod("cicflowmeter")
	kubeapi.CreatePod(podspecCIC)

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
		ledger.Repr()
	}()

	// fmt.Printf("\033[2K\r")
	var wgExecExp sync.WaitGroup
	for scene := range scenarios {
		wgExecExp.Add(1)
		go startScenario(scene, &wgExecExp)
	}
	wgExecExp.Wait()

	var wgProcessing sync.WaitGroup
	wgProcessing.Add(2)
	go func() {
		for scene := range scenarios {
			joyProcessing(scene.UUID.String())
		}
	}()

	go func() {
		for scene := range scenarios {
			cicProcessing(scene.UUID.String())
		}
	}()
	wgProcessing.Wait()

	var wgBundle sync.WaitGroup
	for scene := range scenarios {
		wgBundle.Add(1)
		go func(sc *scenario.Scenario) {
			uuid := sc.UUID.String()
			errs := [4]error{}
			_, errs[0] = os.Stat("/home/dhoogla/PhD/containercap-scenarios/" + uuid + ".yaml")
			_, errs[1] = os.Stat("/home/dhoogla/PhD/containercap-captures/" + uuid + ".pcap")
			_, errs[2] = os.Stat("/home/dhoogla/PhD/containercap-transformed/" + uuid + ".pcap_Flow.csv")
			_, errs[3] = os.Stat("/home/dhoogla/PhD/containercap-transformed/" + uuid + ".joy")

			for i, err := range errs {
				if err != nil {
					fmt.Println(errs[i].Error())
					return
				}
			}

			if err := os.MkdirAll("/home/dhoogla/PhD/containercap-completed/"+uuid, os.ModeDir); err != nil {
				errs[0] = os.Rename("/home/dhoogla/PhD/containercap-scenarios/"+uuid+".yaml", "/home/dhoogla/PhD/containercap-completed/"+uuid+"/"+uuid+".yaml")
				errs[1] = os.Rename("/home/dhoogla/PhD/containercap-scenarios/"+uuid+".pcap", "/home/dhoogla/PhD/containercap-completed/"+uuid+"/"+uuid+".pcap")
				errs[2] = os.Rename("/home/dhoogla/PhD/containercap-transformed/"+uuid+".pcap_Flow.csv", "/home/dhoogla/PhD/containercap-completed/"+uuid+"/"+uuid+".pcap_Flow.csv")
				errs[3] = os.Rename("/home/dhoogla/PhD/containercap-transformed/"+uuid+".joy", "/home/dhoogla/PhD/containercap-completed/"+uuid+"/"+uuid+".joy")
			} else {
				fmt.Println(err.Error())
				return
			}

			for i, err := range errs {
				if err != nil {
					fmt.Println(errs[i].Error())
					return
				}
			}
			fmt.Println("Completed bundling for scenario => ", uuid)
		}(scene)

	}
	wgBundle.Wait()
}
