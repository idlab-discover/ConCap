package main

import (
	"flag"
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
	fh, err := os.Open("/mnt/containercap-scenarios/" + filename)
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
			fmt.Println("launched: ", scn.Attacker.AtkCommand)
			scn.StartTime = time.Now()
			ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: "IN PROGRESS", Scenario: scn})
			kubeapi.ExecShellInContainer("default", scn.UUID.String(), scn.Attacker.Name, scn.Attacker.AtkCommand)
			kubeapi.DeletePod(scn.UUID.String())
			ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: "COMPLETED", Scenario: scn})
			scn.StopTime = time.Now()
			scenario.WriteScenario(scn, "/mnt/containercap-scenarios/"+scn.UUID.String()+".yaml")
		} else {
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
		}
	}
}

func joyProcessing(scenarioUUID string) {
	fmt.Println("JOY: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", "joy", "joy",
		"./joy retain=1 bidir=1 num_pkts=200 dist=1 cdist=none entropy=1 wht=0 example=0 dns=1 ssh=1 tls=1 dhcp=1 dhcpv6=1 http=1 ike=1 payload=1 salt=0 ppi=0 fpx=0 verbosity=4 threads=4 "+"/mnt/containercap-captures/"+scenarioUUID+".pcap"+" | gunzip > /mnt/containercap-transformed/"+scenarioUUID+".joy.json")
}

func cicProcessing(scenarioUUID string) {
	fmt.Println("CIC: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", "cicflowmeter", "cicflowmeter", "./cfm /mnt/containercap-captures/"+scenarioUUID+".pcap"+" /mnt/containercap-transformed/")
}

func main() {
	flag.Parse()
	fmt.Println("Containercap")
	defer kubeapi.DeletePod("joy")
	defer kubeapi.DeletePod("cicflowmeter")
	podspecJoy := scenario.FlowProcessPod("joy")
	kubeapi.CreatePod(podspecJoy)
	podspecCIC := scenario.FlowProcessPod("cicflowmeter")
	kubeapi.CreatePod(podspecCIC)

	files, err := ioutil.ReadDir("/mnt/containercap-scenarios/")
	fmt.Println("Number of files read", len(files))
	if err != nil {
		log.Println(err.Error())
		return
	}

	scenariosChan := make(chan *scenario.Scenario, len(files))

	go func() {
		defer close(scenariosChan)
		var wgReadExp sync.WaitGroup
		for _, file := range files {
			wgReadExp.Add(1)
			fmt.Println("Fx:", file.Name())
			go loadScenarios(file.Name(), scenariosChan, &wgReadExp)
		}
		wgReadExp.Wait()
		ledger.Repr()
	}()

	// fmt.Printf("\033[2K\r")
	var wgExecExp sync.WaitGroup
	for scene := range scenariosChan {
		wgExecExp.Add(1)
		go startScenario(scene, &wgExecExp)
	}
	wgExecExp.Wait()

	var wgProcessing sync.WaitGroup
	scenarios := ledger.Keys()
	wgProcessing.Add(2)
	go func() {
		defer wgProcessing.Done()
		for _, scene := range scenarios {
			joyProcessing(scene)
		}
	}()

	go func() {
		defer wgProcessing.Done()
		for _, scene := range scenarios {
			cicProcessing(scene)
		}
	}()
	wgProcessing.Wait()

	var wgBundle sync.WaitGroup
	for _, scn := range scenarios {
		wgBundle.Add(1)
		go func(scene string) {
			defer wgBundle.Done()
			errs := [4]error{}
			_, errs[0] = os.Stat("/mnt/containercap-scenarios/" + scene + ".yaml")
			_, errs[1] = os.Stat("/mnt/containercap-captures/" + scene + ".pcap")
			_, errs[2] = os.Stat("/mnt/containercap-transformed/" + scene + ".pcap_Flow.csv")
			_, errs[3] = os.Stat("/mnt/containercap-transformed/" + scene + ".joy.json")

			for i, err := range errs {
				if err != nil {
					fmt.Println(errs[i].Error())
					return
				}
			}

			// errs[0] = os.Chown("/mnt/containercap-scenarios/"+scene+".yaml", 1000, 1000)
			// errs[1] = os.Chown("/mnt/containercap-captures/"+scene+".pcap", 1000, 1000)
			// errs[2] = os.Chown("/mnt/containercap-transformed/"+scene+".pcap_Flow.csv", 1000, 1000)
			// errs[3] = os.Chown("/mnt/containercap-transformed/"+scene+".joy.json", 1000, 1000)

			// for i, err := range errs {
			// 	if err != nil {
			// 		fmt.Println(errs[i].Error())
			// 		return
			// 	}
			// }

			if err := os.MkdirAll("/mnt/containercap-completed/"+scene, 0700); err != nil {
				fmt.Println(err.Error())
				return
			} else {
				errs[0] = os.Rename("/mnt/containercap-scenarios/"+scene+".yaml", "/mnt/containercap-completed/"+scene+"/"+scene+".yaml")
				errs[1] = os.Rename("/mnt/containercap-captures/"+scene+".pcap", "/mnt/containercap-completed/"+scene+"/"+scene+".pcap")
				errs[2] = os.Rename("/mnt/containercap-transformed/"+scene+".pcap_Flow.csv", "/mnt/containercap-completed/"+scene+"/"+scene+".pcap_Flow.csv")
				errs[3] = os.Rename("/mnt/containercap-transformed/"+scene+".joy.json", "/mnt/containercap-completed/"+scene+"/"+scene+".joy.json")
			}

			for i, err := range errs {
				if err != nil {
					fmt.Println(errs[i].Error())
					return
				}
			}
			fmt.Println("Completed bundling for scenario => ", scene)
		}(scn)

	}
	wgBundle.Wait()
}
