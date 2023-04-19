package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"

	"log"
	"net"
	"net/http"
	"os"

	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	"github.com/radovskyb/watcher"
	atksetter "gitlab.ilabt.imec.be/lpdhooge/containercap/atkcommandsetter"
	capengi "gitlab.ilabt.imec.be/lpdhooge/containercap/capture-engines"
	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
)

// Constants
const (
	containercapScenarios = "/containercap-scenarios/"
	containercapCompleted = "/containercap-completed/"
	containercapCaptures  = "/containercap-captures/"
	containercapTransform = "/containercap-transformed/"
)

var sugar *zap.SugaredLogger
var once sync.Once
var mountLoc string
var scenarioDir string
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

	//meta.NewMeta()
	//time.Sleep(100 * time.Second)
}

type IPAddress struct {
	TargetAddress net.IP
	//StartTargetRange net.IP
	//EndTargetRange net.IP
	SupportIP []net.IP
	//StartSupportRange net.IP
	//EndSupportRange net.IP
}

// loadScenarios will read scenarios from disk and put their Scenario representation in a channel
// this function is not exported and it is used as a goroutine for parallel & async scenario loading
func loadScenarios(filename string, wg *sync.WaitGroup) *scenario.Scenario {
	defer wg.Done()

	fh, err := os.Open(scnMap[filename].inputDir + "/" + filename + ".yaml")
	if err != nil {

		log.Println("Couldn't read file" + scnMap[filename].inputDir + "/" + filename + ".yaml")
	}
	defer fh.Close()
	scn := scenario.ReadScenario2(fh)
	scn.UUID, err = uuid.Parse(strings.Split(filename, ".")[0])
	if err != nil {
		log.Println("File had incorrect UUID filename")
	}
	ledger.Register(scn.UUID.String())
	fmt.Println("loaded scenario onto channel")
	return scn
}

// startScenario will interface with our wrappers around the k8s api to
// It also shows how the ledger is intended to be used.
// Through the api, the pod status is checked once every 10s through a goroutine
// If a pod's containers are all ready, then the attack command is executed
// When the attack finishes, the pod is cleaned up and the scenario is amended to include the exact stop time
func startScenario(scn *scenario.Scenario, wg *sync.WaitGroup) {
	defer wg.Done()

	//syscall.Umask(0) // https://stackoverflow.com/questions/14249467/os-mkdir-and-os-mkdirall-permissions
	if err := os.MkdirAll(scnMap[scn.UUID.String()].transformDir, 0777); err != nil {
		fmt.Println(err.Error())
	}
	//syscall.Umask(0)
	if err := os.MkdirAll(scnMap[scn.UUID.String()].captureDir, 0777); err != nil {
		fmt.Println(err.Error())
	}

	//######################################################################//
	//					CREATING ATTACK AND TARGET PODS						//
	//######################################################################//

	var attackpod kubeapi.PodSpec
	var targetpod kubeapi.PodSpec
	var targetpodspec apiv1.Pod

	// Create WaitGroup for 2 pods
	var Podwg sync.WaitGroup
	Podwg.Add(2)
	// Create Attack Pod
	go func() {
		defer Podwg.Done()
		attackpod = CreateAttackPod(scn, scnMap[scn.UUID.String()].captureDir)

	}()
	// Create Target Pod
	go func() {
		defer Podwg.Done()
		targetpodspec, targetpod = CreateTargetPod(scn, scnMap[scn.UUID.String()].captureDir, &targetpodspec)
		//time.Sleep(2 * time.Second)
		if scn.Target.Category == "ssh" {
			stdo, stde := kubeapi.ExecShellInContainer("default", targetpod.Uuid, scn.Target.Name, "sudo service rsyslog stop && sudo service rsyslog restart")
			//stdo, stde := kubeapi.ExecShellInContainer("default", targetpod.Uuid, scn.Target.Name, "sudo service rsyslog stop && sudo service rsyslog restart && sleep 4 && cat var/log/auth.log")
			if stde != "" {
				fmt.Println(scn.UUID.String() + " : " + scn.Target.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
			}
			fmt.Println(scn.UUID.String() + " : " + scn.Target.Name + " : stderr: " + stde)
		}
	}()

	// Wait for Both Pods to be Running
	Podwg.Wait()

	var attack string
	if scn.Attacker.AtkCommand == "" {
		attack = atksetter.GenerateAttackCommand(scn)
		scn.Attacker.AtkCommand = attack
	}

	ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: ledger.STARTING, Time: time.Now()})

	//######################################################################//
	//							CHECKING POD STATUS							//
	//######################################################################//

	podStates := make(chan bool, 64)
	ready := false
	fmt.Println("attacker: " + attackpod.Name + "		target: " + targetpod.Name)

	go kubeapi.CheckPodsStatus(podStates, attackpod.Uuid, targetpod.Uuid)
	for msg := range podStates {
		if msg {
			ready = ready || msg
		} else {
			ready = ready && msg
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodsStatus(podStates, attackpod.Uuid, targetpod.Uuid)
		}
	}

	if ready {

		go checkHealth("http://"+targetpod.PodIP+":8989/health", "Hello", "Test", attackpod, *scn)

		//######################################################################//
		//				STARTING ATTACK	  + 	CREATING PCAP FILE				//
		//######################################################################//

		go capengi.PcapCreator(scn, scnMap[scn.UUID.String()].captureDir+"/"+scn.UUID.String()+".pcap", attackpod, targetpod)
		fmt.Println("Loading GoPacket...")
		ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: ledger.RUNNING, Time: time.Now()})

		buf := new(bytes.Buffer)
		sweaters := IPAddress{net.ParseIP(targetpod.PodIP), nil}
		tmpl, err := template.New("test").Parse(scn.Attacker.AtkCommand)
		if err != nil {
			fmt.Println("Something went wrong while implementing the attack command: " + err.Error())
		}
		err = tmpl.Execute(buf, sweaters)
		if err != nil {
			fmt.Println("Something went wrong while implementing the attack command (2): " + err.Error())
		}

		fmt.Println("Launching: " + buf.String())

		fmt.Println("The attack will last " + scn.Attacker.AtkTime)
		var command = "timeout " + scn.Attacker.AtkTime + " "
		scn.StartTime = time.Now()
		stdo, stde := kubeapi.ExecShellInContainer("default", attackpod.Uuid, attackpod.ContainerName, command+buf.String())

		if stde != "" {
			fmt.Println(scn.UUID.String() + " : " + scn.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
		}
		fmt.Println(scn.UUID.String() + " : " + scn.Attacker.Name + " : stderr: " + stde)

		//######################################################################//
		//								STOP ATTACK								//
		//######################################################################//

		scn.StopTime = time.Now()

		kubeapi.AddLabelToRunningPod("idle", "true", attackpod.Uuid)
		scenario.WriteScenario(scn, scnMap[scn.UUID.String()].inputDir+"/"+scn.UUID.String()+".yaml")
		targetName := targetpodspec.ObjectMeta.Name
		err = kubeapi.DeletePod(targetName)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("Deleted Pod: " + targetName)
			scenario.MinusTargetPodCount()
		}

		ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: ledger.COMPLETED, Time: time.Now()})
		time.Sleep(5 * time.Second)
	}
}

// startScenario will use our wrappers around k8s to run pods and send commands to these activated pods.
// After waiting til all pods are ready, the attack command is executed (after the necessary values (IPs) are assigned to the command).
// In the background (asynchronous method), packet capturing is enabled.
// This function is run when the support pod pool is in use: a support pod will also be activated and a possible command can be send there.
func startScenarioWithSupport(scn *scenario.Scenario, wg *sync.WaitGroup) {
	defer wg.Done()

	// https://stackoverflow.com/questions/14249467/os-mkdir-and-os-mkdirall-permissions
	os.MkdirAll(scnMap[scn.UUID.String()].transformDir, 0777)
	os.MkdirAll(scnMap[scn.UUID.String()].captureDir, 0777)

	//######################################################################//
	//					CREATING ATTACK, TARGET AND SUPPORT PODS			//
	//######################################################################//

	var attackpod kubeapi.PodSpec
	var targetpod kubeapi.PodSpec
	var targetpodspec apiv1.Pod
	supportpod := make([]kubeapi.PodSpec, len(scn.Support))
	//supportpodspec := scenario.SupportPods(scn, scnMap[scn.UUID.String()].captureDir)
	fmt.Println("There are " + fmt.Sprint(len(scn.Support)) + " support pods found")
	var mu sync.Mutex
	var supportIPs []net.IP

	// Create WaitGroup for all pods
	var Podwg sync.WaitGroup
	Podwg.Add(2)

	// Create Attack Pod
	go func() {
		defer Podwg.Done()
		attackpod = CreateAttackPod(scn, scnMap[scn.UUID.String()].captureDir)

	}()
	// Create Target Pod
	go func() {
		defer Podwg.Done()
		targetpodspec, targetpod = CreateTargetPod(scn, scnMap[scn.UUID.String()].captureDir, &targetpodspec)
		time.Sleep(2 * time.Second)
		if scn.Target.Category == "ssh" {
			stdo, stde := kubeapi.ExecShellInContainer("default", targetpod.Uuid, scn.Target.Name, "sudo service rsyslog restart")

			if stde != "" {
				fmt.Println(scn.UUID.String() + " : " + scn.Target.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
			}
			fmt.Println(scn.UUID.String() + " : " + scn.Target.Name + " : stderr: " + stde)
		}
	}()

	Podwg.Wait()

	supportpodspec := scenario.SupportPods(scn, scnMap[scn.UUID.String()].captureDir, targetpod.PodIP)
	// Create Support Pods
	for index, helperpod := range supportpodspec {
		Podwg.Add(1)
		go func(index int, helperpod *apiv1.Pod) {
			defer Podwg.Done()
			helper, _, _ := kubeapi.CreateRunningPod(helperpod, false)
			fmt.Println(" Created support pod: " + helper.Name + " with IP: " + helper.PodIP + "\n")

			mu.Lock()
			supportpod[index] = helper
			supportIPs = append(supportIPs, net.ParseIP(helper.PodIP))
			mu.Unlock()
			time.Sleep(2 * time.Second)
			if scn.Support[index].Category == "fail2ban" {
				stdo, stde := kubeapi.ExecShellInContainer("default", supportpod[index].Uuid, scn.Support[index].Name, scn.Support[index].SupCommand)

				if stde != "" {
					fmt.Println(scn.UUID.String() + " : " + scn.Support[index].Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
				}
				fmt.Println(scn.UUID.String() + " : " + scn.Support[index].Name + " : stderr: " + stde)
			}

		}(index, helperpod)

	}
	Podwg.Wait()

	var attack string
	if scn.Attacker.AtkCommand == "" {
		attack = atksetter.GenerateAttackCommand(scn)
		scn.Attacker.AtkCommand = attack
	}

	fmt.Println(" KUBEAPI: All pods created")
	ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: ledger.STARTING, Time: time.Now()})

	//######################################################################//
	//							CHECKING POD STATUS							//
	//######################################################################//

	podStates := make(chan bool, 64)
	ready := false
	var pods []string

	pods = append(pods, attackpod.Uuid, targetpod.Uuid)
	for _, pod := range supportpod {
		pods = append(pods, pod.Uuid)
	}

	go kubeapi.CheckPodsStatus(podStates, pods...)

	for msg := range podStates {
		if msg {
			ready = ready || msg
		} else {
			ready = ready && msg
			time.Sleep(10 * time.Second)
			go kubeapi.CheckPodsStatus(podStates, pods...)
		}
	}

	if ready {
		go checkHealth("http://"+targetpod.PodIP+":8989/health", "Hello", "Test", attackpod, *scn)

		//######################################################################//
		//				STARTING ATTACK	  + 	CREATING PCAP FILE				//
		//######################################################################//

		go capengi.PcapCreator2(scn, scnMap[scn.UUID.String()].captureDir+"/"+scn.UUID.String()+".pcap", attackpod, targetpod, supportpod...)
		fmt.Println("Loading GoPacket...\n")

		ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: ledger.RUNNING, Time: time.Now()})

		bufAttack := new(bytes.Buffer)
		attack := IPAddress{net.ParseIP(targetpod.PodIP), supportIPs}
		attacktmpl, err := template.New("test").Parse(scn.Attacker.AtkCommand)
		if err != nil {
			fmt.Println("Something went wrong while implementing the attack command...")
		}
		err = attacktmpl.Execute(bufAttack, attack)
		if err != nil {
			fmt.Println("Something went wrong while implementing the attack command... (2)")
		}

		bufSupport := new(bytes.Buffer)
		support := IPAddress{net.ParseIP(targetpod.PodIP), supportIPs}

		for _, sups := range scn.Support {
			supporttmpl, err := template.New("test").Parse(sups.SupCommand)
			if err != nil {
				fmt.Println("Something went wrong while implementing the support command...")
			}
			err = supporttmpl.Execute(bufSupport, support)
			if err != nil {
				fmt.Println("Something went wrong while implementing the support command... (2)")
			}

		}
		fmt.Println("Launching: " + bufAttack.String() + "\t and " + bufSupport.String() + "\n")

		fmt.Println("The attack will last " + scn.Attacker.AtkTime)
		var command = "timeout " + scn.Attacker.AtkTime + " "

		scn.StartTime = time.Now()
		//stdo, stde := kubeapi.ExecShellInContainer("default", supportpod.Uuid, scn.Support, bufSupport.String())                 //scn.Attacker.AtkCommand) //_ was stdo

		kubeapi.ExecShellInContainer("default", attackpod.Uuid, scn.Attacker.Name, command+bufAttack.String())

		/*if stde != "" {
			fmt.Println("\t" + scn.UUID.String() + " : " + scn.Support[0].Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
		}
		if stdeAttack != "" {
			fmt.Println("\t" + scn.UUID.String() + " : " + scn.Attacker.Name + " : stdout: " + stdoAttack + "\n\t stderr: " + stdeAttack)
		}
		fmt.Println(scn.UUID.String() + " : " + scn.Attacker.Name) //+ " : stderr: " + stde
		*/
		//######################################################################//
		//								STOP ATTACK								//
		//######################################################################//

		scn.StopTime = time.Now()

		for _, podspec := range supportpod {
			kubeapi.AddLabelToRunningPod("idle", "true", podspec.Uuid)
		}
		/*
			for _, podspec:= range supportpodspec{
				err = kubeapi.DeletePodAndPVC(podspec.ObjectMeta.Name)
				if err != nil {
					fmt.Println(err.Error())
				} else {
					scenario.MinusSupportPodCount()
				}
			}
		*/

		kubeapi.AddLabelToRunningPod("idle", "true", attackpod.Uuid)

		ledger.UpdateState(scn.UUID.String(), ledger.LedgerEntry{State: ledger.COMPLETED, Time: time.Now()})
		scenario.WriteScenario(scn, scnMap[scn.UUID.String()].inputDir+"/"+scn.UUID.String()+".yaml")
		targetName := targetpodspec.ObjectMeta.Name
		err = kubeapi.DeletePod(targetName)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("Deleted Pod: " + targetName)
			scenario.MinusTargetPodCount()
		}

	}
}

func CreateAttackPod(scn *scenario.Scenario, captureDir string) kubeapi.PodSpec {

	attackpodspec := scenario.AttackPod(scn, captureDir)

	attackpod, reused, _ := kubeapi.CreateRunningPod(attackpodspec, true)

	if reused {
		fmt.Println(" Attackerpod " + attackpod.Name + " with IP: " + attackpod.PodIP + " will be reused\n")
	} else {
		fmt.Println(" Created attack pod: " + attackpod.Name + " with IP: " + attackpod.PodIP + "\n")
	}
	return attackpod
}

func CreateTargetPod(scn *scenario.Scenario, captureDir string, targetpodspec *apiv1.Pod) (apiv1.Pod, kubeapi.PodSpec) {

	targetpodspec = scenario.TargetPod(scn, captureDir)
	targetpod, _, _ := kubeapi.CreateRunningPod(targetpodspec, false)
	fmt.Println(" Created target pod: " + targetpod.Name + " with IP: " + targetpod.PodIP + "\n")
	return *targetpodspec, targetpod
}

func checkHealth(url string, headerName string, headerValue string, attackerpod kubeapi.PodSpec, scn scenario.Scenario) bool {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set(headerName, headerValue)

	for {
		time.Sleep(4 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK || ctx.Err() == context.DeadlineExceeded {
			kubeapi.ExecShellInContainer("default", attackerpod.Uuid, attackerpod.ContainerName, "pkill -n -e")
			fmt.Println("The Target is down")
			return false
		}
	}
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

func bundling(scene *scenario.Scenario) {
	gotError := false
	errs := [4]error{}
	_, errs[0] = os.Stat(scnMap[scene.UUID.String()].inputDir + "/" + scene.UUID.String() + ".yaml")
	_, errs[1] = os.Stat(scnMap[scene.UUID.String()].captureDir + "/" + scene.UUID.String() + ".pcap")
	_, errs[2] = os.Stat(scnMap[scene.UUID.String()].transformDir + "/" + scene.UUID.String() + ".pcap_Flow.csv")
	_, errs[3] = os.Stat(scnMap[scene.UUID.String()].transformDir + "/" + scene.UUID.String() + ".joy.json")

	for i, err := range errs {
		if err != nil {
			fmt.Println("Error 1 bundling: " + errs[i].Error())
			gotError = true
		}
	}

	if err := os.MkdirAll(scnMap[scene.UUID.String()].outputDir, 0777); err != nil {
		fmt.Println(err.Error())
	} else {
		errs[0] = os.Rename(scnMap[scene.UUID.String()].inputDir+"/"+scene.UUID.String()+".yaml", scnMap[scene.UUID.String()].outputDir+"/"+scene.UUID.String()+".yaml")
		errs[1] = os.Rename(scnMap[scene.UUID.String()].captureDir+"/"+scene.UUID.String()+".pcap", scnMap[scene.UUID.String()].outputDir+"/"+scene.UUID.String()+".pcap")
		errs[2] = os.Rename(scnMap[scene.UUID.String()].transformDir+"/"+scene.UUID.String()+".pcap_Flow.csv", scnMap[scene.UUID.String()].outputDir+"/"+scene.UUID.String()+".pcap_Flow.csv")
		//errs[3] = os.Rename(scnMap[scene.UUID.String()].transformDir+"/"+scene.UUID.String()+".joy.json", scnMap[scene.UUID.String()].outputDir+"/"+scene.UUID.String()+".joy.json")
		jsonFile := scnMap[scene.UUID.String()].transformDir + "/" + scene.UUID.String() + ".joy.json"
		csvFile := scnMap[scene.UUID.String()].outputDir + "/" + scene.UUID.String() + ".joy.csv"
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
	//addMetadataToCSV(scnMap[scene.UUID.String()].outputDir+"/"+scene.UUID.String()+".pcap_Flow", scene, scene.UUID.String()+".pcap_Flow")

	name := time.Now().Format("02-01-2006") + "_" + string(scene.ScenarioType) + "_" + string(scene.Attacker.Category) + "_" + scene.UUID.String()
	if err := zipSource(scnMap[scene.UUID.String()].outputDir, flagstore.MountLoc+"/containercap-completed/"+name+".zip"); err != nil {
		fmt.Println("Error 3 bundling (zipping): " + err.Error())
	} else {
		if err := os.RemoveAll(scnMap[scene.UUID.String()].outputDir); err != nil {
			fmt.Println("Error 3 bundling (removing files): " + err.Error())
		}
	}

	ledger.UpdateState(scene.UUID.String(), ledger.LedgerEntry{State: ledger.BUNDLED, Time: time.Now()})
	if !gotError {
		fmt.Println("Completed bundling for scenario => " + scene.UUID.String())
	} else {
		fmt.Println("Completed bundling (with error) for scenario => " + scene.UUID.String())
	}
}

// Per-Scenario function: all necessary actions are bundled here, from loading the scenario to bundling the results.
// First, a check is run whether the (hard-coded) limit of usable pods for containercap is respected.
// Otherwise waiting 10 seconds til an OK is received.
// Afterwards, processing pods are being defined and the scenario gets instantiated in scnMap (var scnMap = map[string]*scnMeta{}).
// The scenario UUID should be unique, a scenario can only be run once (per execution of the program).
// Then the actual processing of the scenario starts: loading, starting, processing and bundling.
// This function is run asynchronously, to allow for simultaneous execution of multiple scenarios at once.
func bundledFunction(scnUUID string) {

	var parentPath = ""
	var filename = scnUUID

	value, ok := scnMap[filename]
	if ok && value.started { // File already exists in scnMap
		for {
			if value.done { // If scenario was already finished, execute (or wait otherwise)
				//return
				newUUID, err := uuid.NewUUID()
				if err != nil {
					fmt.Println("Something went wrong creating a new UUID for scenario: " + filename)
				}
				yamlFile, err := os.Open(scenarioDir + parentPath + filename + ".yaml")
				if err != nil {
					fmt.Println("Error reading YAML file: " + err.Error())
				}
				yaml := scenario.ReadScenario2(yamlFile)
				yaml.UUID = newUUID
				scenario.WriteScenario(yaml, scenarioDir+filename+".yaml")
				err = os.Rename(scenarioDir+parentPath+filename+".yaml", scenarioDir+parentPath+newUUID.String()+".yaml")
				if err != nil {
					fmt.Println("Error renaming YAML file: " + err.Error())
				}
				filename = newUUID.String()
				fmt.Println("Scenario (uuid) not unique, changing name to " + newUUID.String())
				yamlFile.Close()
				break
			} else {
				time.Sleep(10 * time.Second)
				value = scnMap[filename]
			}
		}
	}

	scn := &scnMeta{
		inputDir:     mountLoc + containercapScenarios + parentPath,
		outputDir:    mountLoc + containercapCompleted + filename,
		captureDir:   mountLoc + containercapCaptures + filename,
		transformDir: mountLoc + containercapTransform + filename,
		done:         false,
		started:      true,
	}

	scnMap[filename] = scn

	//######################################################################//
	//						LOADING SCENARIO								//
	//######################################################################//

	var wgLoad sync.WaitGroup
	wgLoad.Add(1)
	scene := loadScenarios(filename, &wgLoad)
	wgLoad.Wait()

	for {
		//helperpod := kubeapi.AttackerPodExists(scene.Attacker.Name)
		//defer kubeapi.ReuseIdlePod(helperpod)

		ok := scenario.CheckAmountOfPods()

		// PROBLEM: it will just check whether the hard limit is reached.
		// This scenario however might have pods it can reuse... Needs to get checked.

		// Here we need to call a new function that checks for a certain type of pod (possibly attacker/support)

		// 2ND PROBLEM: After reaching 50 pods, no new scenarios will be able to run. Need to remove unused pods.
		// This second problem is handled within CheckAmountOfPods => Will remove idle pods AFTER hardcoded limit has been reached.
		if ok {
			break
		} else {

			fmt.Println("Maximum amount of pods reached, waiting...")
			time.Sleep(10 * time.Second)
		}
	}

	//######################################################################//
	//						CREATING ANALYSIS TOOLS							//
	//######################################################################//

	podNames := []string{"joy", "cicflowmeter"} // We can Add other Processing Pods Like Argus

	var wgCreatePod sync.WaitGroup
	var printOnce sync.Once
	for _, podName := range podNames {
		exists, err := kubeapi.PodExists(podName)
		if err != nil {
			fmt.Printf("Error checking if pod %s exists: %v\n", podName, err)
			continue // skip creating this pod if there's an error
		}
		if !exists {
			wgCreatePod.Add(1)
			go func(name string) {
				defer wgCreatePod.Done()
				podspec := scenario.FlowProcessPod(name)
				_, _, _ = kubeapi.CreateRunningPod(podspec, true)
				printOnce.Do(func() {
					fmt.Printf("Pod %s created\n", name)
				})
			}(podName)
		} else {

			printOnce.Do(func() {
				fmt.Printf("Pod %s already exists\n", podName)
			})

		}
	}

	wgCreatePod.Wait() // wait till all required pods are created

	// get all required pods by name
	for _, podName := range podNames {
		pod, err := kubeapi.GetRunningPodByName(podName)
		if err != nil {
			fmt.Printf("Error getting pod %s: %v\n", podName, err)
			continue // skip getting this pod if there's an error
		}
		switch podName {
		case "joy":
			joyPod = pod
		case "cicflowmeter":
			cicPod = pod
		} // Add case for other processing tools
	}

	//######################################################################//
	//							EXECUTE SCENARIO							//
	//######################################################################//
	var supportname string
	var wgExec sync.WaitGroup
	for _, value := range scene.Support {
		supportname = value.Name
		break
	}
	if supportname != "" {
		wgExec.Add(1)
		startScenarioWithSupport(scene, &wgExec)
	} else {
		wgExec.Add(1)
		startScenario(scene, &wgExec)
	}
	wgExec.Wait()
	fmt.Println("Packet capturing is done\n")

	time.Sleep(5 * time.Second)

	//######################################################################//
	//						ANALYSIS OF PCAP FILE							//
	//######################################################################//

	var wgAnalyse sync.WaitGroup

	wgAnalyse.Add(2)

	go capengi.JoyProcessing(scnMap[filename].captureDir, scnMap[filename].transformDir, &wgAnalyse, joyPod, filename)
	go capengi.CicProcessing(scnMap[filename].captureDir, scnMap[filename].transformDir, &wgAnalyse, cicPod, filename)

	wgAnalyse.Wait()

	fmt.Println("Analysing completed")

	time.Sleep(5 * time.Second)

	bundling(scene)
	scnMap[filename].done = true
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

	mountLoc = GetFlags().MountLoc
	scenarioDir = mountLoc + containercapScenarios

	if _, err := os.Stat(scenarioDir); os.IsNotExist(err) {
		log.Fatal("Scenario directory does not exist")
		return
	}

	err := filepath.WalkDir(scenarioDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			parentPath := strings.Replace(filepath.Dir(path), scenarioDir, "", -1)
			filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))

			if parentPath != "" {
				parentPath = parentPath + "/"
			}

			if strings.Contains(GetFlags().Selection, parentPath) || GetFlags().Selection == "all" {
				scnMap[filename] = &scnMeta{
					inputDir:     mountLoc + containercapScenarios + parentPath,
					outputDir:    mountLoc + containercapCompleted + parentPath + filename,
					captureDir:   mountLoc + containercapCaptures + parentPath + filename,
					transformDir: mountLoc + containercapTransform + parentPath + filename,
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Println(err.Error())
		return
	}

	go scenarioWatcher(scenarioDir)

	fmt.Println("Number of files read: " + fmt.Sprint(len(scnMap)))
	if len(scnMap) == 0 {
		fmt.Println("No scenarios present. Please add or press Q or Escape to exit.")
	} else {

		var wgExec sync.WaitGroup

		for scnUUID := range scnMap {
			wgExec.Add(1)
			go func(scnName string) {
				defer wgExec.Done()
				bundledFunction(scnName)
			}(scnUUID)
		}
		wgExec.Wait()

		fmt.Println("\nAll found scenarios done. \nWaiting for new scenarios, or press Q or Escape to exit.")
	}
}

// scenarioWatcher will watch a folder, checking for newly created/added files.
// New files will be handled as scenarios and the bundledFunction will get triggered.
// The watching process will check every 100 ms.
func scenarioWatcher(folder string) {
	w := watcher.New()
	w.FilterOps(watcher.Create)

	go func() {
		for {
			select {
			case event := <-w.Event:
				newu, err := uuid.Parse(strings.Split(event.FileInfo.Name(), ".")[0])
				if err != nil {
					fmt.Println(err.Error())
				} else {
					fmt.Println(newu.String() + " has been added.")
					go bundledFunction(newu.String())
				}
			case err := <-w.Error:
				fmt.Println("Error scenariowatcher: " + err.Error())
			case <-w.Closed:
				return
			}

		}
	}()

	// Watch this folder for changes.
	if err := w.Add(folder); err != nil {
		log.Fatalln(err)
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}
