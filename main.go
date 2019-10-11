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
var mountLoc string

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()
	sugar = logger.Sugar()
	flag.StringVar(&mountLoc, "m", "/mnt/L/kube/", "The mount path on the host")
	// on the live kube cluster this flag will be /groups/wall2-ilabt-iminds-be/cybersecurity/L/kube/
	flag.Parse()
}

func loadScenarios(filename string, scns chan *scenario.Scenario, wg *sync.WaitGroup) {
	defer wg.Done()
	fh, err := os.Open(mountLoc + "containercap-scenarios/" + filename)
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
			scenario.WriteScenario(scn, mountLoc+"containercap-scenarios/"+scn.UUID.String()+".yaml")
		} else {
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
		}
	}
}

func joyProcessing(scenarioUUID string) {
	fmt.Println("JOY: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", "joy", "joy",
		"./joy retain=1 bidir=1 num_pkts=200 dist=1 cdist=none entropy=1 wht=0 example=0 dns=1 ssh=1 tls=1 dhcp=1 dhcpv6=1 http=1 ike=1 payload=1 salt=0 ppi=0 fpx=0 verbosity=4 threads=4 "+"/mnt/L/kube/containercap-captures/"+scenarioUUID+".pcap"+" | gunzip > /mnt/L/kube/containercap-transformed/"+scenarioUUID+".joy.json")
}

func cicProcessing(scenarioUUID string) {
	fmt.Println("CIC: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", "cicflowmeter", "cicflowmeter", "./cfm /mnt/L/kube/containercap-captures/"+scenarioUUID+".pcap"+" /mnt/L/kube/containercap-transformed/")
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

	files, err := ioutil.ReadDir(mountLoc + "containercap-scenarios/")
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
			_, errs[0] = os.Stat(mountLoc + "containercap-scenarios/" + scene + ".yaml")
			_, errs[1] = os.Stat(mountLoc + "containercap-captures/" + scene + ".pcap")
			_, errs[2] = os.Stat(mountLoc + "containercap-transformed/" + scene + ".pcap_Flow.csv")
			_, errs[3] = os.Stat(mountLoc + "containercap-transformed/" + scene + ".joy.json")

			for i, err := range errs {
				if err != nil {
					fmt.Println(errs[i].Error())
					return
				}
			}

			if err := os.MkdirAll(mountLoc+"containercap-completed/"+scene, 0700); err != nil {
				fmt.Println(err.Error())
				return
			} else {
				errs[0] = os.Rename(mountLoc+"containercap-scenarios/"+scene+".yaml", mountLoc+"containercap-completed/"+scene+"/"+scene+".yaml")
				errs[1] = os.Rename(mountLoc+"containercap-captures/"+scene+".pcap", mountLoc+"containercap-completed/"+scene+"/"+scene+".pcap")
				errs[2] = os.Rename(mountLoc+"containercap-transformed/"+scene+".pcap_Flow.csv", mountLoc+"containercap-completed/"+scene+"/"+scene+".pcap_Flow.csv")
				errs[3] = os.Rename(mountLoc+"containercap-transformed/"+scene+".joy.json", mountLoc+"containercap-completed/"+scene+"/"+scene+".joy.json")
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
