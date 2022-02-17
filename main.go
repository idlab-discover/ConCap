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
	"github.com/jessevdk/go-flags"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger
var once sync.Once

type FlagStore struct {
	MountLoc string `short:"m" description:"The mount path on the host"`
}

var flagstore FlagStore

func GetFlags() FlagStore {
	once.Do(func() {
		_, err := flags.Parse(&flagstore)
		if err != nil {
			panic(err)
		}
	})
	return flagstore
}

// the init function of main will
// instantiate the sugared version of the Zap logger (https://github.com/uber-go/zap)
// I recommend using zap, but logging in this project is currently not used much and certainly not consistently
// (re)set the mount location for the persistent storage
// parse the other flags
func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()
	sugar = logger.Sugar()
}

// loadScenarios will read scenarios from disk and put their Scenario representation in a channel
// this function is not exported and it is used as a goroutine for parallel & async scenario loading
func loadScenarios(filename string, scns chan *scenario.Scenario, wg *sync.WaitGroup) {
	defer wg.Done()
	fh, err := os.Open(GetFlags().MountLoc + "/containercap-scenarios/" + filename)
	if err != nil {
		log.Println("Couldn't read file", filename)
	}
	defer fh.Close()
	scn := scenario.ReadScenario(fh)
	scn.UUID, err = uuid.Parse(strings.Split(filename, ".")[0])
	if err != nil {
		log.Println("File had incorrect UUID filename")
	}
	ledger.Register(scn)
	scns <- scn
	fmt.Println("loaded scenario onto channel")
}

// startScenario will interface with our wrappers around the k8s api to
// It also shows how the ledger is intended to be used.
// Through the api, the pod status is checked once every 10s through a goroutine
// If a pod's containers are all ready, then the attack command is executed
// When the attack finishes, the pod is cleaned up and the scenario is amended to include the exact stop time
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
			scenario.WriteScenario(scn, GetFlags().MountLoc+"/containercap-scenarios/"+scn.UUID.String()+".yaml")
		} else {
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodStatus(scn.UUID.String(), podStates)
		}
	}
}

// joyProcessing is called after an experiment pod with scenarioUUID is done to extract features from the captured .pcap file
// it sends the command to the long-living joy container
// the shell argument is very specific and should not be modified
func joyProcessing(scenarioUUID string) {
	fmt.Println("JOY: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", "joy", "joy",
		"./joy retain=1 bidir=1 num_pkts=200 dist=1 cdist=none entropy=1 wht=0 example=0 dns=1 ssh=1 tls=1 dhcp=1 dhcpv6=1 http=1 ike=1 payload=1 salt=0 ppi=0 fpx=0 verbosity=4 threads=4 /storage/nfs/L/kube/containercap-captures/"+scenarioUUID+".pcap"+" | gunzip > /storage/nfs/L/kube/containercap-transformed/"+scenarioUUID+".joy.json")
}

// cicProcessing is called after an experiment pod with scenarioUUID is done to extract features from the captured .pcap file
// it sends the command to the long-living joy container
// the shell argument should not be modified
func cicProcessing(scenarioUUID string) {
	fmt.Println("CIC: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", "cicflowmeter", "cicflowmeter", "./cfm /storage/nfs/L/kube/containercap-captures/"+scenarioUUID+".pcap /storage/nfs/L/kube/containercap-transformed/")
}

// main ties everything together
// 0. flag parsing
// 1. setup and teardown (at function end with defer) of the long-living feature processing pods
// 2. reading a folder which contains all new scenario (experiment) YAML definitions
//    this includes a trick with an anonymous go routine to load the scenarios asynchronously
//    the pods are not created here just yet, only the scenario specs
// 3. start all scenarios in separate goroutines, startScenario(), defined earlier will poll and run when everything is ready, then clean itself up
// 4. after all scenarios are done, start feature processing in 2 goroutines that are started simultaneously for CICFlowmeter and Joy, however both tools churn through the scenario pcaps one by one.
//    this may be scaled so that there are multiple flow processing pods for CICFlowmeter and Joy
// 5. 4 checks happen, scenario yaml, scenario pcap, CICFlowmeter CSV and Joy JSON should be present for the experiment. These are the bundled in their own folder and moved to the other completed scenarios
//    this indexing is basic, because the folders are UUIDs, a better implementation would leverage the currently available metadata (attack type, attack tool, ...) to store the results in a better structure
//    we will probably add a NoSQL database to store and later publish finished experiments
// NOTE: currently there are several synchronization points i.e. WaitGroup instance.Wait()
// These can probably be relaxed, especially the sync point to complete all scenario runs before processing pcaps into features
func main() {
	// flag.Parse()
	fmt.Println(GetFlags().MountLoc)
	fmt.Println("Containercap")
	defer kubeapi.DeletePod("joy")
	defer kubeapi.DeletePod("cicflowmeter")
	podspecJoy := scenario.FlowProcessPod("joy")
	kubeapi.CreatePod(podspecJoy)
	podspecCIC := scenario.FlowProcessPod("cicflowmeter")
	kubeapi.CreatePod(podspecCIC)

	time.Sleep(2 * time.Second)

	var mountLoc = GetFlags().MountLoc

	files, err := ioutil.ReadDir(mountLoc + "/containercap-scenarios/")
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
	}()

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
			_, errs[0] = os.Stat(mountLoc + "/containercap-scenarios/" + scene + ".yaml")
			_, errs[1] = os.Stat(mountLoc + "/containercap-captures/" + scene + ".pcap")
			_, errs[2] = os.Stat(mountLoc + "/containercap-transformed/" + scene + ".pcap_Flow.csv")
			_, errs[3] = os.Stat(mountLoc + "/containercap-transformed/" + scene + ".joy.json")

			for i, err := range errs {
				if err != nil {
					fmt.Println(errs[i].Error())
					return
				}
			}

			if err := os.MkdirAll(mountLoc+"/containercap-completed/"+scene, 0700); err != nil {
				fmt.Println(err.Error())
			} else {
				errs[0] = os.Rename(mountLoc+"/containercap-scenarios/"+scene+".yaml", mountLoc+"/containercap-completed/"+scene+"/"+scene+".yaml")
				errs[1] = os.Rename(mountLoc+"/containercap-captures/"+scene+".pcap", mountLoc+"/containercap-completed/"+scene+"/"+scene+".pcap")
				errs[2] = os.Rename(mountLoc+"/containercap-transformed/"+scene+".pcap_Flow.csv", mountLoc+"/containercap-completed/"+scene+"/"+scene+".pcap_Flow.csv")
				errs[3] = os.Rename(mountLoc+"/containercap-transformed/"+scene+".joy.json", mountLoc+"/containercap-completed/"+scene+"/"+scene+".joy.json")
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
