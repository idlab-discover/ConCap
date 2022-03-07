package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
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

type scnMeta struct {
	inputDir     string
	outputDir    string
	captureDir   string
	transformDir string
}

var scnMap = map[string]*scnMeta{}

type FlagStore struct {
	MountLoc  string `short:"m" description:"The mount path on the host"`
	Selection string `short:"s" description:"The selection of the scenario's to run, default=all" optional:"true" default:"all"`
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

	fh, err := os.Open(scnMap[filename].inputDir + "/" + filename + ".yaml")
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

	os.MkdirAll(scnMap[scn.UUID.String()].transformDir, 0777)
	os.MkdirAll(scnMap[scn.UUID.String()].captureDir, 0777)

	podspec := scenario.ScenarioPod(scn, scnMap[scn.UUID.String()].captureDir)
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
			scenario.WriteScenario(scn, scnMap[scn.UUID.String()].inputDir+"/"+scn.UUID.String()+".yaml")
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
	kubeapi.ExecShellInContainer("default", "joy", "joy", "./joy retain=1 bidir=1 num_pkts=200 dist=1 cdist=none entropy=1 wht=0 example=0 dns=1 ssh=1 tls=1 dhcp=1 dhcpv6=1 http=1 ike=1 payload=1 salt=0 ppi=0 fpx=0 verbosity=4 threads=4 "+scnMap[scenarioUUID].captureDir+"/"+scenarioUUID+".pcap"+" | gunzip > "+scnMap[scenarioUUID].transformDir+"/"+scenarioUUID+".joy.json")

}

// cicProcessing is called after an experiment pod with scenarioUUID is done to extract features from the captured .pcap file
// it sends the command to the long-living joy container
// the shell argument should not be modified
func cicProcessing(scenarioUUID string) {
	fmt.Println("CIC: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", "cicflowmeter", "cicflowmeter", "./cfm "+scnMap[scenarioUUID].captureDir+"/"+scenarioUUID+".pcap "+scnMap[scenarioUUID].transformDir)
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

	if _, err := os.Stat(mountLoc + "/containercap-scenarios"); os.IsNotExist(err) {
		log.Fatal("Scenario directory does not exist")
		return
	}
	err := filepath.WalkDir(mountLoc+"/containercap-scenarios", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			parentPath := strings.Replace(filepath.Dir(path), mountLoc+"/containercap-scenarios/", "", -1)
			filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))

			if strings.Contains(GetFlags().Selection, parentPath) || GetFlags().Selection == "all" {
				scnMap[filename] = &scnMeta{
					inputDir:     mountLoc + "/containercap-scenarios/" + parentPath,
					outputDir:    mountLoc + "/containercap-completed/" + parentPath + "/" + filename,
					captureDir:   mountLoc + "/containercap-captures/" + parentPath + "/" + filename,
					transformDir: mountLoc + "/containercap-transformed/" + parentPath + "/" + filename,
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Println(err.Error())
		return
	}

	fmt.Println("Number of files read", len(scnMap))
	scenariosChan := make(chan *scenario.Scenario, len(scnMap))

	go func() {
		defer close(scenariosChan)
		var wgReadExp sync.WaitGroup
		for scnUUID := range scnMap {
			wgReadExp.Add(1)
			fmt.Println("Fx:", scnUUID)
			go loadScenarios(scnUUID, scenariosChan, &wgReadExp)
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
			_, errs[0] = os.Stat(scnMap[scene].inputDir + "/" + scene + ".yaml")
			_, errs[1] = os.Stat(scnMap[scene].captureDir + "/" + scene + ".pcap")
			_, errs[2] = os.Stat(scnMap[scene].transformDir + "/" + scene + ".pcap_Flow.csv")
			_, errs[3] = os.Stat(scnMap[scene].transformDir + "/" + scene + ".joy.json")

			for i, err := range errs {
				if err != nil {
					fmt.Println(errs[i].Error())
					return
				}
			}

			if err := os.MkdirAll(scnMap[scene].outputDir, 0700); err != nil {
				fmt.Println(err.Error())
			} else {
				errs[0] = os.Rename(scnMap[scene].inputDir+"/"+scene+".yaml", scnMap[scene].outputDir+"/"+scene+".yaml")
				errs[1] = os.Rename(scnMap[scene].captureDir+"/"+scene+".pcap", scnMap[scene].outputDir+"/"+scene+".pcap")
				errs[2] = os.Rename(scnMap[scene].transformDir+"/"+scene+".pcap_Flow.csv", scnMap[scene].outputDir+"/"+scene+".pcap_Flow.csv")
				errs[3] = os.Rename(scnMap[scene].transformDir+"/"+scene+".joy.json", scnMap[scene].outputDir+"/"+scene+".joy.json")
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
