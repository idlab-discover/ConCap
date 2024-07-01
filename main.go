package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os/signal"
	"syscall"

	"log"
	"net"
	"os"

	"path/filepath"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/radovskyb/watcher"
	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
	apiv1 "k8s.io/api/core/v1"
)

// Constants
const (
	FolderWatchInterval = 1 // seconds
	PodWaitInterval     = 5 // seconds
)

type ProcessingPod struct {
	Name  string
	Image string
}

var ProcessingPods = []ProcessingPod{
	{"cicflowmeter", "mielverkerken/cicflowmeter:latest"},
	{"rustiflow", "ghcr.io/matissecallewaert/rustiflow:slim"},
}

type FlagStore struct {
	Directory string `short:"d" long:"dir" description:"The mount path on the host" required:"true"`
	Scenario  string `short:"s" long:"scenario" description:"The scenario's to run, default=all" optional:"true" default:"all"`
	Watch     bool   `short:"w" long:"watch" description:"Watch for new scenarios in the directory" optional:"true"`
}

var flagstore FlagStore

// the init function of main will parse provided flags before running the main, gracefully stop running if parsing fails.
func init() {
	_, err := flags.Parse(&flagstore)
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}
}

type IPAddress struct {
	TargetAddress net.IP
	SupportIP     []net.IP
}

// startScenario will interface with our wrappers around the k8s api
// It also shows how the ledger is intended to be used.
// Through the api, the pod status is checked once every 10s through a goroutine
// If a pod's containers are all ready, then the attack command is executed
// When the attack finishes, the pod is cleaned up and the scenario is amended to include the exact stop time
func startScenario(scn *scenario.Scenario, outputDir string) {

	//######################################################################//
	//					CREATING ATTACK AND TARGET PODS						//
	//######################################################################//

	var attackpod kubeapi.PodSpec
	var targetpod kubeapi.PodSpec
	var targetpodspec apiv1.Pod

	// Create WaitGroup for spawning pods
	var Podwg sync.WaitGroup
	ledger.UpdateState(scn.UUID, ledger.LedgerEntry{State: ledger.STARTING, Time: time.Now()})

	// Create Attack Pod
	Podwg.Add(1)
	go func() {
		defer Podwg.Done()
		var err error
		attackpod, err = CreateAttackPod(scn)
		if err != nil {
			log.Printf("Error creating attack pod: %v", err)
			return
		}
	}()
	// Create Target Pod
	Podwg.Add(1)
	go func() {
		defer Podwg.Done()
		var err error
		targetpodspec, targetpod, err = CreateTargetPod(scn)
		if err != nil {
			log.Printf("Error creating target pod: %v", err)
			return
		}
	}()

	// Wait for all pods to be running
	Podwg.Wait()
	log.Println("All pods running")

	//######################################################################//
	//				STARTING ATTACK	  + 	CREATING PCAP FILE				//
	//######################################################################//
	ledger.UpdateState(scn.UUID, ledger.LedgerEntry{State: ledger.RUNNING, Time: time.Now()})

	// Parses attack command and changes the IP address to the target pod's IP address
	buf := new(bytes.Buffer)
	attackIP := IPAddress{net.ParseIP(targetpod.PodIP), []net.IP{}}
	tmpl, err := template.New("test").Parse(scn.Attacker.AtkCommand)
	if err != nil {
		fmt.Println("Something went wrong while implementing the attack command: " + err.Error())
	}
	err = tmpl.Execute(buf, attackIP)
	if err != nil {
		fmt.Println("Something went wrong while implementing the attack command (2): " + err.Error())
	}

	log.Println("Launching attack command: " + buf.String())

	command := buf.String()
	if scn.Attacker.AtkTime != scenario.EmptyAttackDuration {
		log.Println("The attack will last max " + scn.Attacker.AtkTime)
		command = "timeout " + scn.Attacker.AtkTime + " " + command
	}
	scn.StartTime = time.Now()
	stdo, stde, err := kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, attackpod.Uuid, attackpod.ContainerName, command)

	if err != nil {
		log.Println(scn.UUID.String() + " : " + scn.Attacker.Name + " : error: " + err.Error())
	}
	if stde != "" {
		log.Println(scn.UUID.String() + " : " + scn.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
	}

	//######################################################################//
	//								STOP ATTACK								//
	//######################################################################//

	scn.StopTime = time.Now()
	log.Println(scn.UUID.String() + ": Attack finished")

	kubeapi.AddLabelToRunningPod("idle", "true", attackpod.Uuid)
	ledger.UpdateState(scn.UUID, ledger.LedgerEntry{State: ledger.COMPLETED, Time: time.Now()})
	targetName := targetpodspec.ObjectMeta.Name

	// Download the pcap file from the target pod to local and upload to analyse pcap file
	kubeapi.CopyFileFromPod(scn.UUID.String()+"-target", "tcpdump", "/data/dump.pcap", filepath.Join(outputDir, "/dump.pcap"), true)
	kubeapi.CopyFileToPod("cicflowmeter", "cicflowmeter", filepath.Join(outputDir, "/dump.pcap"), filepath.Join("/data/pcap", scn.UUID.String()+".pcap"))
	kubeapi.CopyFileToPod("rustiflow", "rustiflow", filepath.Join(outputDir, "/dump.pcap"), filepath.Join("/data/pcap", scn.UUID.String()+".pcap"))

	err = kubeapi.DeletePod(targetName)
	if err != nil {
		log.Println(err.Error())
	} else {
		log.Println("Deleted Pod: " + targetName)
		scenario.MinusTargetPodCount()
	}

	time.Sleep(5 * time.Second)
}

// Can be combined with CreateTargetPod function
func CreateAttackPod(scn *scenario.Scenario) (kubeapi.PodSpec, error) {

	attackpodspec := scenario.AttackPod(scn)

	attackpod, reused, _ := kubeapi.CreateRunningPod(attackpodspec, true)

	if reused {
		log.Println(" Attackerpod " + attackpod.Name + " with IP: " + attackpod.PodIP + " will be reused")
	} else {
		log.Println(" Created attack pod: " + attackpod.Name + " with IP: " + attackpod.PodIP)
	}
	return attackpod, nil
}

func CreateTargetPod(scn *scenario.Scenario) (apiv1.Pod, kubeapi.PodSpec, error) {

	targetpodspec := scenario.TargetPod(scn)
	targetpod, _, _ := kubeapi.CreateRunningPod(targetpodspec, false)
	log.Println(" Created target pod: " + targetpod.Name + " with IP: " + targetpod.PodIP)
	return *targetpodspec, targetpod, nil
}

func waitForPodAvailability() {
	if scenario.ExceedingMaxRunningPods() {
		log.Print("Maximum amount of pods reached, deleting idle pods...")
		scenario.DeleteIdlePods()
		ticker := time.NewTicker(time.Duration(PodWaitInterval))
		defer ticker.Stop()
		for range ticker.C {
			if scenario.ExceedingMaxRunningPods() {
				scenario.DeleteIdlePods()
			} else {
				log.Print("Pod availability restored. Resuming pod spawning.")
				break
			}
		}
	}
}

func createFlowExtractionPods() {
	var wg sync.WaitGroup
	log.Print("Creating flow extraction pods")

	for _, processingPod := range ProcessingPods {
		wg.Add(1)
		go func(processingPod ProcessingPod) {
			defer wg.Done()

			exists, err := kubeapi.PodExists(processingPod.Name)
			if err != nil {
				log.Printf("Error checking if pod %s exists: %v\n", processingPod.Name, err)
				return
			}
			if !exists {
				log.Printf("Creating Pod %s\n", processingPod.Name)
				podSpec := scenario.ProcessingPodSpec(processingPod.Name, processingPod.Image)
				_, _, err = kubeapi.CreateRunningPod(podSpec, true)
				if err != nil {
					log.Fatalf("Error running processing pod: %v", err)
				}
				log.Printf("Pod %s created\n", processingPod.Name)
			} else {
				log.Printf("Pod %s already exists\n", processingPod.Name)
			}
		}(processingPod)
	}
	wg.Wait()
}

func analysePcapFile(scenario *scenario.Scenario, outputDir string) {
	// var wg sync.WaitGroup
	// wg.Add(1)
	// go capengi.JoyProcessing(scnMap[uuid].captureDir, scnMap[uuid].transformDir, &wg, joyPod, uuid.String())
	stdo, stde, err := kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, "cicflowmeter", "cicflowmeter", "/CICFlowMeter/bin/cfm /data/pcap/"+scenario.UUID.String()+".pcap /data/flow/")
	if err != nil {
		log.Println(scenario.UUID.String() + " : " + scenario.Attacker.Name + " : error: " + err.Error())
	}
	if stde != "" {
		log.Println(scenario.UUID.String() + " : " + scenario.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
	}
	stdo, stde, err = kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, "rustiflow", "rustiflow", "rustiflow pcap cic-flow 120 /data/pcap/"+scenario.UUID.String()+".pcap csv /data/flow/"+scenario.UUID.String()+".csv")
	if err != nil {
		log.Println(scenario.UUID.String() + " : " + scenario.Attacker.Name + " : error: " + err.Error())
	}
	// TODO fix rustiflow to not output logging to stderr
	// if stde != "" {
	// 	log.Println(scenario.UUID.String() + " : " + scenario.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
	// }
	log.Println("Flow reconstruction & feature extraction completed")
	// Copy analysis results to local and remove file from pod
	kubeapi.CopyFileFromPod("cicflowmeter", "cicflowmeter", "/data/flow/"+scenario.UUID.String()+".pcap_flow.csv", filepath.Join(outputDir, "cic-flows.csv"), true)
	kubeapi.CopyFileFromPod("rustiflow", "rustiflow", "/data/flow/"+scenario.UUID.String()+".csv", filepath.Join(outputDir, "rustiflow.csv"), true)
	// Remove the pcap file from the pod
	// _, stde, err = kubeapi.ExecShellInContainer("default", "cicflowmeter", "cicflowmeter", "rm", "/data/pcap/"+scenario.UUID.String()+".pcap")
	// if stde != "" {
	// 	log.Println(scenario.UUID.String() + " : " + scenario.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
	// }
	log.Println("Flows downloaded")
	// wg.Wait()
}

// Per-Scenario function: all necessary actions are bundled here, from loading the scenario to bundling the results.
// First, a check is run whether the (hard-coded) limit of usable pods for containercap is respected.
// Otherwise waiting 10 seconds til an OK is received.
// Afterwards, processing pods are being defined and the scenario gets instantiated in scnMap (var scnMap = map[string]*scnMeta{}).
// The scenario UUID should be unique, a scenario can only be run once (per execution of the program).
// Then the actual processing of the scenario starts: loading, starting, processing and bundling.
// This function is run asynchronously, to allow for simultaneous execution of multiple scenarios at once.
func scheduleScenario(scenarioPath string, outputDir string) {
	scene, err := scenario.ReadScenario(scenarioPath)
	if err != nil {
		log.Println("Failed to read scenario: " + scenarioPath)
	}
	ledger.Register(scene.UUID)

	// TODO: Take into account reusable pods (both for max running pods and deleting idle pods)
	waitForPodAvailability()

	log.Printf("Starting scenario: %s\n", scene.UUID)
	// if len(scene.Support) > 0 {
	// 	startScenarioWithSupport(scene)

	// } else {
	// 	startScenario(scene)
	// }
	scenarioOutputFolder := filepath.Join(outputDir, scene.UUID.String())
	if err := os.MkdirAll(scenarioOutputFolder, 0777); err != nil {
		log.Println(err.Error())
	}
	startScenario(scene, scenarioOutputFolder)
	log.Printf("Scenario finished: %s\n", scene.UUID)
	err = scenario.WriteScenario(scene, scenarioOutputFolder)
	if err != nil {
		log.Fatalf("error writing scenario file: %v", err)
	}

	analysePcapFile(scene, scenarioOutputFolder)
	// time.Sleep(5 * time.Second)

	// bundling(scene)
	// scnMap[scene.UUID].done = true
}

// main ties everything together
//  0. flag parsing
//  1. setup and teardown (at function end with defer) of the long-living feature processing pods
//  2. reading a folder which contains all new scenario (experiment) YAML definitions
//     this includes a trick with an anonymous go routine to load the scenarios asynchronously
//     the pods are not created here just yet, only the scenario specs
//  3. start all scenarios in separate goroutines, startScenario(), defined earlier will poll and run when everything is ready, then clean itself up
//  4. after all scenarios are done, start feature processing in 2 goroutines that are started simultaneously for CICFlowmeter and Joy, however both tools churn through the scenario pcaps one by one.
//     this may be scaled so that there are multiple flow processing pods for CICFlowmeter and Joy
//  5. 4 checks happen, scenario yaml, scenario pcap, CICFlowmeter CSV and Joy JSON should be present for the experiment. These are the bundled in their own folder and moved to the other completed scenarios
//     this indexing is basic, because the folders are UUIDs, a better implementation would leverage the currently available metadata (attack type, attack tool, ...) to store the results in a better structure
//     we will probably add a NoSQL database to store and later publish finished experiments
//
// NOTE: currently there are several synchronization points i.e. WaitGroup instance.Wait()
// These can probably be relaxed, especially the sync point to complete all scenario runs before processing pcaps into features
func main() {
	absPathDirectory, _ := filepath.Abs(flagstore.Directory)
	scenarioDir := filepath.Join(absPathDirectory, "scenarios")
	completedDir := filepath.Join(absPathDirectory, "completed")
	// Setup channel to listen for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Wait for interrupt signal
	go func() {
		<-quit
		fmt.Println("Shutdown signal received, cleaning up...")
		// Perform cleanup operations here
		// TODO: Add cleanup operations
		os.Exit(0) // Exit after cleanup
	}()

	if _, err := os.Stat(scenarioDir); os.IsNotExist(err) {
		log.Fatalf("Scenario directory does not exist: %s", scenarioDir)
	}

	dirEntries, err := os.ReadDir(scenarioDir)
	if err != nil {
		log.Fatalf("Error reading scenario directory: %s", err.Error())
	}

	// Create the flow extraction pods
	createFlowExtractionPods()

	var filepaths []string
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			filepaths = append(filepaths, filepath.Join(scenarioDir, entry.Name()))
		}
	}

	if len(filepaths) == 0 {
		log.Println("No scenarios found.")
	} else {
		log.Println("Number of scenarios found: " + fmt.Sprint(len(filepaths)))
		var runningScenariosWaitGroup sync.WaitGroup
		for _, path := range filepaths {
			runningScenariosWaitGroup.Add(1)
			go func(scenarioPath string) {
				defer runningScenariosWaitGroup.Done()
				scheduleScenario(scenarioPath, completedDir)
			}(path)
		}
		runningScenariosWaitGroup.Wait()

		log.Println("\nAll scenarios have finished. ")
	}

	// Wait here for new scenarios until CTRL+C or other term signal is received.
	if flagstore.Watch {
		go scenarioWatcher(scenarioDir, completedDir)
		log.Println("Waiting for new scenarios...")
		log.Println("Press Ctrl+C to exit")
		// Block main from exiting
		select {}
	}
}

// scenarioWatcher will watch a folder, checking for newly created/added files.
// New files will be handled as scenarios and the scheduleScenario will get triggered.
func scenarioWatcher(folder string, outputDir string) {
	w := watcher.New()
	w.FilterOps(watcher.Create)

	go func() {
		for {
			select {
			case event := <-w.Event:
				log.Println("A new scenario is found: " + event.FileInfo.Name())
				go scheduleScenario(event.Path, outputDir)
			case err := <-w.Error:
				log.Println("Error scenariowatcher: " + err.Error())
			case <-w.Closed:
				return
			}

		}
	}()

	// Watch this folder for changes.
	if err := w.Add(folder); err != nil {
		log.Fatalln(err)
	}

	// Start the watching process - it'll check for changes every X ms.
	if err := w.Start(time.Duration(FolderWatchInterval)); err != nil {
		log.Fatalln(err)
	}
}
