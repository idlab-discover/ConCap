package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os/signal"
	"syscall"

	"log"
	"net"
	"os"

	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	"github.com/radovskyb/watcher"
	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
	apiv1 "k8s.io/api/core/v1"
)

// Constants
const (
	containercapScenarios = "/containercap-scenarios/"
	containercapCompleted = "/containercap-completed/"
	containercapCaptures  = "/containercap-captures/"
	containercapTransform = "/containercap-transformed/"
	FolderWatchInterval   = 1 // seconds
	PodWaitInterval       = 5 // seconds
)

var FlowExtractionPods = []string{"cicflowmeter", "rustiflow"} // We can Add other Processing Pods Like Rustiflow, Argus, nProbe, etc.
var once sync.Once

// var mountLoc string
var joyPod kubeapi.PodSpec
var cicPod kubeapi.PodSpec

type scnMeta struct {
	inputDir     string
	outputDir    string
	captureDir   string
	transformDir string
	done         bool
	started      bool
}

var scnMap = map[uuid.UUID]*scnMeta{}

// TODO add option to disbable waiting for new scenarios
type FlagStore struct {
	MountLoc  string `short:"m" long:"mount-path" description:"The mount path on the host" required:"true"`
	Selection string `short:"s" long:"scenario" description:"The selection of the scenario's to run, default=all" optional:"true" default:"all"`
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

	// if err := os.MkdirAll(scnMap[scn.UUID].transformDir, 0777); err != nil {
	// 	log.Println(err.Error())
	// }
	// if err := os.MkdirAll(scnMap[scn.UUID].captureDir, 0777); err != nil {
	// 	log.Println(err.Error())
	// }

	//######################################################################//
	//					CREATING ATTACK AND TARGET PODS						//
	//######################################################################//

	var attackpod kubeapi.PodSpec
	var targetpod kubeapi.PodSpec
	var targetpodspec apiv1.Pod
	// supportpod := make([]kubeapi.PodSpec, len(scn.Support))
	// var supportpodspec []*apiv1.Pod
	var supportIPs []net.IP

	// Create WaitGroup for spawning pods
	var Podwg sync.WaitGroup
	ledger.UpdateState(scn.UUID, ledger.LedgerEntry{State: ledger.STARTING, Time: time.Now()})

	// Create Support Pods if present
	// if len(scn.Support) > 0 {
	// 	log.Println(fmt.Sprint(len(scn.Support)) + " support pods found")

	// 	var mu sync.Mutex
	// 	supportpodspec = scenario.SupportPods(scn, scnMap[scn.UUID].captureDir, targetpod.PodIP)

	// 	// Create Support Pods
	// 	for index, helperpod := range supportpodspec {
	// 		Podwg.Add(1)
	// 		go func(index int, helperpod *apiv1.Pod) {
	// 			defer Podwg.Done()
	// 			helper, _, _ := kubeapi.CreateRunningPod(helperpod, false)
	// 			fmt.Println(" Created support pod: " + helper.Name + " with IP: " + helper.PodIP + "\n")

	// 			mu.Lock()
	// 			supportpod[index] = helper
	// 			supportIPs = append(supportIPs, net.ParseIP(helper.PodIP))
	// 			mu.Unlock()
	// 			time.Sleep(2 * time.Second)
	// 			if scn.Support[index].Category == "fail2ban" {
	// 				stdo, stde := kubeapi.ExecShellInContainer("default", supportpod[index].Uuid, scn.Support[index].Name, scn.Support[index].SupCommand)

	// 				if stde != "" {
	// 					fmt.Println(scn.UUID.String() + " : " + scn.Support[index].Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
	// 				}
	// 			}

	// 		}(index, helperpod)

	// 	}
	// }

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
		// if scn.Target.Category == "custom" {
		// 	stdo, stde := kubeapi.ExecShellInContainer("default", targetpod.Uuid, scn.Target.Name, "sudo service rsyslog restart")
		// 	if stde != "" {
		// 		fmt.Println(scn.UUID.String() + " : " + scn.Target.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
		// 	}
		// }
	}()

	// Wait for all pods to be running
	Podwg.Wait()
	log.Println("All pods running")

	// var attack string
	// if scn.Attacker.AtkCommand == "" {
	// 	attack = atksetter.GenerateAttackCommand(scn)
	// 	scn.Attacker.AtkCommand = attack
	// }

	//######################################################################//
	//							CHECKING POD STATUS							//
	//######################################################################//

	// podStates := make(chan bool, 64)
	// ready := false
	// var pods []string

	// pods = append(pods, attackpod.Uuid, targetpod.Uuid)
	// for _, pod := range supportpod {
	// 	pods = append(pods, pod.Uuid)
	// }

	// go kubeapi.CheckPodsStatus(podStates, pods...)
	// for msg := range podStates {
	// 	if msg {
	// 		ready = ready || msg
	// 	} else {
	// 		ready = ready && msg
	// 		time.Sleep(10 * time.Second)
	// 		go kubeapi.CheckPodsStatus(podStates, pods...)
	// 	}
	// }

	//######################################################################//
	//				STARTING ATTACK	  + 	CREATING PCAP FILE				//
	//######################################################################//

	// go capengi.PcapCreatorWithSupport(scn, scnMap[scn.UUID].captureDir+"/"+scn.UUID.String()+".pcap", attackpod, targetpod, supportpod...)
	// fmt.Println("Loading GoPacket...")
	ledger.UpdateState(scn.UUID, ledger.LedgerEntry{State: ledger.RUNNING, Time: time.Now()})

	// Parses attack command and changes the IP address to the target pod's IP address
	buf := new(bytes.Buffer)
	attackIP := IPAddress{net.ParseIP(targetpod.PodIP), supportIPs}
	tmpl, err := template.New("test").Parse(scn.Attacker.AtkCommand)
	if err != nil {
		fmt.Println("Something went wrong while implementing the attack command: " + err.Error())
	}
	err = tmpl.Execute(buf, attackIP)
	if err != nil {
		fmt.Println("Something went wrong while implementing the attack command (2): " + err.Error())
	}

	// bufSupport := new(bytes.Buffer)
	// support := IPAddress{net.ParseIP(targetpod.PodIP), supportIPs}

	// for _, sups := range scn.Support {
	// 	supporttmpl, err := template.New("test").Parse(sups.SupCommand)
	// 	if err != nil {
	// 		fmt.Println("Something went wrong while implementing the support command...")
	// 	}
	// 	err = supporttmpl.Execute(bufSupport, support)
	// 	if err != nil {
	// 		fmt.Println("Something went wrong while implementing the support command... (2)")
	// 	}

	// }

	log.Println("Launching attack command: " + buf.String()) // + "\t and " + bufSupport.String() + "\n")

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

	// for _, podspec := range supportpodspec {
	// 	err = kubeapi.DeletePod(podspec.ObjectMeta.Name)
	// 	if err != nil {
	// 		fmt.Println(err.Error())
	// 	} else {
	// 		scenario.MinusSupportPodCount()
	// 	}
	// }

	kubeapi.AddLabelToRunningPod("idle", "true", attackpod.Uuid)
	ledger.UpdateState(scn.UUID, ledger.LedgerEntry{State: ledger.COMPLETED, Time: time.Now()})
	targetName := targetpodspec.ObjectMeta.Name

	// Download the pcap file from the target pod to local and upload to analyse pcap file
	kubeapi.CopyFileFromPod(scn.UUID.String()+"-target", "tcpdump", "/data/dump.pcap", filepath.Join(outputDir, "/dump.pcap"), true)
	kubeapi.CopyFileToPod("cicflowmeter", "cicflowmeter", filepath.Join(outputDir, "/dump.pcap"), filepath.Join("/data/pcap", scn.UUID.String()+".pcap"))

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

// zipSource is a helper function which is used to zip the final folder in containercap-completed/<scenario-UUID>
// as way to clean things up as well as overcoming the problem with overlapping .pcap files.
// Source: https://gosamples.dev/zip-file/
func zipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}
		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(headerWriter, f)
		return err
	})
}

// Does not work like it should be. Nested values in the json file can't convert => use a wrapped python script
func convertJSONToCSV(inputFile, outputFile string) error {
	jsonFile, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	scanner := bufio.NewScanner(jsonFile)
	scanner.Scan() // Skip the first line

	fieldnames := []string{}
	dataList := []map[string]interface{}{}

	for scanner.Scan() {
		line := scanner.Text()
		var data map[string]interface{}
		err := json.Unmarshal([]byte(line), &data)
		if err != nil {
			return err
		}

		for key := range data {
			found := false
			for _, field := range fieldnames {
				if field == key {
					found = true
					break
				}
			}
			if !found {
				fieldnames = append(fieldnames, key)
			}
		}
		dataList = append(dataList, data)
	}

	csvFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	err = writer.Write(fieldnames)
	if err != nil {
		return err
	}

	for _, data := range dataList {
		row := make([]string, len(fieldnames))
		for i, field := range fieldnames {
			if value, ok := data[field]; ok {
				row[i] = fmt.Sprint(value)
			} else {
				row[i] = ""
			}
		}
		err := writer.Write(row)
		if err != nil {
			return err
		}
	}

	return nil
}

// The converted json file should not be written to the zip file.
func bundling(scene *scenario.Scenario) {
	gotError := false
	errs := [4]error{}
	_, errs[0] = os.Stat(scnMap[scene.UUID].inputDir + "/" + scene.UUID.String() + ".yaml")
	_, errs[1] = os.Stat(scnMap[scene.UUID].captureDir + "/" + scene.UUID.String() + ".pcap")
	_, errs[2] = os.Stat(scnMap[scene.UUID].transformDir + "/" + scene.UUID.String() + ".pcap_Flow.csv")
	_, errs[3] = os.Stat(scnMap[scene.UUID].transformDir + "/" + scene.UUID.String() + ".joy.json")

	for i, err := range errs {
		if err != nil {
			fmt.Println("Error 1 bundling: " + errs[i].Error())
			gotError = true
		}
	}

	if err := os.MkdirAll(scnMap[scene.UUID].outputDir, 0777); err != nil {
		fmt.Println(err.Error())
	} else {
		errs[0] = os.Rename(scnMap[scene.UUID].inputDir+"/"+scene.UUID.String()+".yaml", scnMap[scene.UUID].outputDir+"/"+scene.UUID.String()+".yaml")
		errs[1] = os.Rename(scnMap[scene.UUID].captureDir+"/"+scene.UUID.String()+".pcap", scnMap[scene.UUID].outputDir+"/"+scene.UUID.String()+".pcap")
		errs[2] = os.Rename(scnMap[scene.UUID].transformDir+"/"+scene.UUID.String()+".pcap_Flow.csv", scnMap[scene.UUID].outputDir+"/"+scene.UUID.String()+".pcap_Flow.csv")
		jsonFile := scnMap[scene.UUID].transformDir + "/" + scene.UUID.String() + ".joy.json"
		csvFile := scnMap[scene.UUID].outputDir + "/" + scene.UUID.String() + ".joy.csv"
		errs[3] = convertJSONToCSV(jsonFile, csvFile)
		if errs[3] == nil {
			_ = os.Remove(jsonFile) // Delete the original JSON file after conversion
		}

	}

	for i, err := range errs {
		if err != nil {
			fmt.Println("Error 2 bundling: " + errs[i].Error())
			gotError = true
		}
	}

	// This function is usable but commented out.

	name := time.Now().Format("02-01-2006") + "_" + string(scene.ScenarioType) + "_" + string(scene.Attacker.Category) + "_" + scene.UUID.String()
	if err := zipSource(scnMap[scene.UUID].outputDir, flagstore.MountLoc+"/containercap-completed/"+name+".zip"); err != nil {
		fmt.Println("Error 3 bundling (zipping): " + err.Error())
	} else {
		if err := os.RemoveAll(scnMap[scene.UUID].outputDir); err != nil {
			fmt.Println("Error 3 bundling (removing files): " + err.Error())
		}
	}

	ledger.UpdateState(scene.UUID, ledger.LedgerEntry{State: ledger.BUNDLED, Time: time.Now()})
	if !gotError {
		fmt.Println("Completed bundling for scenario => " + scene.UUID.String())
	} else {
		fmt.Println("Completed bundling (with error) for scenario => " + scene.UUID.String())
	}
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

	for _, podName := range FlowExtractionPods {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			exists, err := kubeapi.PodExists(podName)
			if err != nil {
				log.Printf("Error checking if pod %s exists: %v\n", podName, err)
				return
			}
			if !exists {
				log.Printf("Creating Pod %s\n", podName)
				podSpec, err := scenario.LoadPodSpecFromYaml(filepath.Join("kubernetes-resources", podName+".yaml"))
				if err != nil {
					log.Fatalf("Error loading processing pod: %v", err)
				}
				_, _, err = kubeapi.CreateRunningPod(podSpec, true)
				if err != nil {
					log.Fatalf("Error running processing pod: %v", err)
				}
				log.Printf("Pod %s created\n", podName)
			} else {
				log.Printf("Pod %s already exists\n", podName)
			}
		}(podName)
	}
	wg.Wait()
}

func analysePcapFile(scenario *scenario.Scenario, outputDir string) {
	// var wg sync.WaitGroup
	// wg.Add(1)
	// go capengi.JoyProcessing(scnMap[uuid].captureDir, scnMap[uuid].transformDir, &wg, joyPod, uuid.String())
	stdo, stde, err := kubeapi.ExecShellInContainer("default", "cicflowmeter", "cicflowmeter", "/CICFlowMeter/bin/cfm /data/pcap/"+scenario.UUID.String()+".pcap /data/flow/")
	if err != nil {
		log.Println(scenario.UUID.String() + " : " + scenario.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
	}
	if stde != "" {
		log.Println(scenario.UUID.String() + " : " + scenario.Attacker.Name + " : error: " + err.Error())
	}
	log.Println("Flow reconstruction & feature extraction completed")
	// Copy analysis results to local and remove file from pod
	kubeapi.CopyFileFromPod("cicflowmeter", "cicflowmeter", "/data/flow/"+scenario.UUID.String()+".pcap_flow.csv", filepath.Join(outputDir, "cic-flows.csv"), true)
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
	absMountLoc, _ := filepath.Abs(flagstore.MountLoc)
	scenarioDir := filepath.Join(absMountLoc, "scenarios")
	completedDir := filepath.Join(absMountLoc, "completed")
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
	once.Do(createFlowExtractionPods)

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
	// go scenarioWatcher(scenarioDir, completedDir)
	// log.Println("Waiting for new scenarios...")
	// log.Println("Press Ctrl+C to exit")
	// // Block main from exiting
	// select {}
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
				filename := strings.TrimSuffix(filepath.Base(event.FileInfo.Name()), filepath.Ext(filepath.Base(event.FileInfo.Name())))
				log.Println("A new scenario is found: " + filename)
				go scheduleScenario(filename, outputDir)
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
