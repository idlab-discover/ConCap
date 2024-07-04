// Package scenario handles our scenario specification through a number of structs.
// It also contains the podbuilder which fills in the Kubernetes templates for pods with the appropriate information
// These scenario files are exported to yaml and serve as the starting point of every experiment
package scenario

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
)

const EmptyAttackDuration = ""

type Scenario struct {
	UUID         uuid.UUID          `yaml:"uuid"`
	Name         string             `yaml:"name"`
	ScenarioType string             `yaml:"scenarioType"`
	StartTime    time.Time          `yaml:"startTime"`
	StopTime     time.Time          `yaml:"stopTime"`
	Attacker     Attacker           `yaml:"attacker"`
	Target       Target             `yaml:"target"`
	Support      []Support          `yaml:"support"`
	Tag          string             `yaml:"tag"`
	OutputDir    string             `yaml:"-"`
	Deployment   ScenarioDeployment `yaml:"-"`
}

type Attacker struct {
	Category   string `yaml:"category"`
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	AtkCommand string `yaml:"atkCommand"`
	AtkTime    string `yaml:"atkTime"`
}

type Target struct {
	Category string  `yaml:"category"`
	Name     string  `yaml:"name"`
	Image    string  `yaml:"image"`
	Ports    []int32 `yaml:"ports"`
	Filter   string  `yaml:"filter"`
}

type Support struct {
	Category   string  `yaml:"category"`
	Name       string  `yaml:"name"`
	Image      string  `yaml:"image"`
	SupCommand string  `yaml:"supCommand"`
	Ports      []int32 `yaml:"ports"`
}

type ProcessingPod struct {
	Name  string
	Image string
}

type ScenarioDeployment struct {
	AttackPodSpec kubeapi.RunningPodSpec
	TargetPodSpec kubeapi.RunningPodSpec
}

var ProcessingPods = []ProcessingPod{
	{"cicflowmeter", "mielverkerken/cicflowmeter:latest"},
	{"rustiflow", "ghcr.io/matissecallewaert/rustiflow:slim"},
}

// ReadScenario will unmarshall the yaml back into the in-memory Scenario representation
func ReadScenario(filePath string) (*Scenario, error) {
	s := Scenario{}

	fileHandler, err := os.Open(filePath)
	if err != nil {
		log.Println("Failed to open file " + filePath + " : " + err.Error())
		return nil, err
	}
	defer fileHandler.Close()

	b, err := io.ReadAll(fileHandler)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML: %w", err)
	}

	err = yaml.UnmarshalStrict(b, &s)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	if s.Attacker.Image == "" {
		return nil, fmt.Errorf("no attack-image provided for attack: '%s'", s.Attacker.Name)
	}

	atkTime, err := parseToSeconds(s.Attacker.AtkTime)
	if err != nil {
		s.Attacker.AtkTime = EmptyAttackDuration
	} else {
		s.Attacker.AtkTime = atkTime
	}
	// Modify the attack command to include a timeout if a duration is provided
	if s.Attacker.AtkTime != EmptyAttackDuration {
		s.Attacker.AtkCommand = "timeout " + s.Attacker.AtkTime + " " + s.Attacker.AtkCommand
	}

	s.UUID = uuid.New()
	s.Name = cleanPodName(strings.TrimSuffix(filepath.Base(fileHandler.Name()), filepath.Ext(fileHandler.Name())))

	if s.Target.Filter == "" {
		s.Target.Filter = "((dst host {{.AttackAddress}} and src host {{.TargetAddress}}) or (dst host {{.TargetAddress}} and src host {{.AttackAddress}})) and not arp"
	}
	return &s, nil
}

// WriteScenario will marshall the in-memory Scenario to valid yaml and write it to disk
func (s *Scenario) WriteScenario(outputDir string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return os.WriteFile(filepath.Join(outputDir, "scenario.yaml"), b, 0644)
}

// Executes the scenario from start to finish.
// 1. Deploys the pods
// 2. Start traffic capture on the target pod
// 3. Executes the attack
// 4. Downloads the pcap capture and updated scenario file
// 5. Cleans up the pods
func (s *Scenario) Execute() error {
	// 1. Deploy the pods for this scenario
	s, err := s.DeployPods()
	if err != nil {
		return fmt.Errorf("failed to deploy pods for scenario: %v", err)
	}

	// 2. Start traffic capture on the target pod
	s.replacePlaceholdersInCommands(s.Deployment)
	log.Printf("Starting traffic capture on target pod %v for scenario %v", s.Deployment.TargetPodSpec.PodName, s.Name)
	// Start tcpdump in the target pod, redirect stdo and stde to a log file, and write the pid to a file for later cleanup
	stdo, stde, err := kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, s.Deployment.TargetPodSpec.PodName, "tcpdump", "nohup tcpdump --no-promiscuous-mode --immediate-mode --buffer-size=32768 --packet-buffered -n --interface=eth0 -w /data/dump.pcap '"+s.Target.Filter+"' > /data/tcpdump.log 2>&1 & echo $! > /data/tcpdump.pid")
	if err != nil {
		return fmt.Errorf("error starting tcpdump in scenario %v, error: %v", s.Name, err)
	}
	if stde != "" {
		log.Printf("%s : %s : stdout: %s\n\t stderr: %s", s.Name, s.Attacker.Name, stdo, stde)
	}

	// 3. Execute the attack
	log.Printf("Executing attack '%v' in scenario %v", s.Attacker.AtkCommand, s.Name)
	s.StartTime = time.Now()
	stdo, stde, err = kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, s.Deployment.AttackPodSpec.PodName, s.Deployment.AttackPodSpec.ContainerName, s.Attacker.AtkCommand)
	if err != nil {
		return fmt.Errorf("error executing command in scenario %v, error: %v", s.Name, err)
	}
	if stde != "" {
		log.Printf("%s : %s : stdout: %s\n\t stderr: %s", s.Name, s.Attacker.Name, stdo, stde)
	}
	s.StopTime = time.Now()
	log.Printf("Attack finished in scenario %v", s.Name)

	// 4. Download the pcap capture from the target pod and export the updated scenario file
	_, _, err = kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, s.Deployment.TargetPodSpec.PodName, "tcpdump", `kill -SIGINT $(cat /data/tcpdump.pid) && while ! ps | grep "\[tcpdump\]" 2>/dev/null; do sleep 1; done`) // Workaround for tcpdump becoming a zombie process because spawned by other shell
	if err != nil {
		return fmt.Errorf("failed to stop tcpdump in target pod: %v", err)
	}
	log.Printf("Stopped traffic capture on target pod %v for scenario %v", s.Deployment.TargetPodSpec.PodName, s.Name)
	err = kubeapi.CopyFileFromPod(s.Deployment.TargetPodSpec.PodName, "tcpdump", "/data/dump.pcap", filepath.Join(s.OutputDir, "/dump.pcap"), true)
	if err != nil {
		return fmt.Errorf("failed to download pcap file from target pod: %v", err)
	}
	err = kubeapi.CopyFileFromPod(s.Deployment.TargetPodSpec.PodName, "tcpdump", "/data/tcpdump.log", filepath.Join(s.OutputDir, "/tcpdump.log"), true)
	if err != nil {
		return fmt.Errorf("failed to download tcpdump log file from target pod: %v", err)
	}
	err = s.WriteScenario(s.OutputDir)
	if err != nil {
		return fmt.Errorf("error writing scenario file: %v", err)
	}

	// 5. Cleanup the pods
	_ = kubeapi.DeletePod(s.Deployment.TargetPodSpec.PodName)
	_ = kubeapi.DeletePod(s.Deployment.AttackPodSpec.PodName)
	return nil
}

// AnalysePcap performs flow reconstruction and feature extraction on the pcap file
// by executing a command in the processing pods concurrently. When for commands to complete in all processing pods.
func (s *Scenario) AnalysePcap() error {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		stdo, stde, err := kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, "cicflowmeter", "cicflowmeter", "/CICFlowMeter/bin/cfm /data/pcap/"+s.Name+".pcap /data/flow/")
		if err != nil {
			log.Println(s.Name + " : " + s.Attacker.Name + " : error: " + err.Error())
		}
		if stde != "" {
			log.Println(s.Name + " : " + s.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
		}
	}()

	go func() {
		defer wg.Done()
		_, _, err := kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, "rustiflow", "rustiflow", "rustiflow pcap cic-flow 120 /data/pcap/"+s.Name+".pcap csv /data/flow/"+s.Name+".csv")
		if err != nil {
			log.Println(s.Name + " : " + s.Attacker.Name + " : error: " + err.Error())
		}
		// TODO fix rustiflow to not output logging to stderr
		// if stde != "" {
		// 	log.Println(scenario.Name + " : " + scenario.Attacker.Name + " : stdout: " + stdo + "\n\t stderr: " + stde)
		// }
	}()

	wg.Wait()
	log.Println("Flow reconstruction & feature extraction completed for scenario: ", s.Name)
	return nil
}

// DeployPods deploys the attacker, target and support pods for the scenario in a concurrent manner.
// It waits for all pods to be running before returning.
func (s *Scenario) DeployPods() (*Scenario, error) {
	log.Println("Deploying pods for scenario: ", s.Name)
	var wg sync.WaitGroup
	wg.Add(2)

	// Channels to capture PodSpecs and errors
	errChan := make(chan error, 2)

	// 1. Deploy the attacker pod(s)
	go func() {
		defer wg.Done()
		attackPod := s.AttackPod()
		podspec, err := kubeapi.CreateRunningPod(attackPod)
		if err != nil {
			errChan <- fmt.Errorf("failed to deploy attacker pod: %w", err)
			return
		}
		s.Deployment.AttackPodSpec = podspec
	}()

	// 2. Deploy the target pod
	go func() {
		defer wg.Done()
		targetPod := s.TargetPod()
		podspec, err := kubeapi.CreateRunningPod(targetPod)
		if err != nil {
			errChan <- fmt.Errorf("failed to deploy target pod: %w", err)
			return
		}
		s.Deployment.TargetPodSpec = podspec
	}()
	// 3. Deploy the support pod(s)

	// 4. Wait for all pods to be running
	log.Println("Waiting for pods to be running for scenario: ", s.Name)
	wg.Wait()
	log.Println("All pods are running for scenario: ", s.Name)
	close(errChan)

	// 5. Check for errors and return values
	for err := range errChan {
		if err != nil {
			return s, err
		}
	}

	return s, nil
}

func (s *Scenario) DeletePods() error {
	if s.Deployment != (ScenarioDeployment{}) {
		err := kubeapi.DeletePod(s.Deployment.AttackPodSpec.PodName)
		if err != nil {
			return fmt.Errorf("failed to delete attacker pod: %w", err)
		}

		err = kubeapi.DeletePod(s.Deployment.TargetPodSpec.PodName)
		if err != nil {
			return fmt.Errorf("failed to delete target pod: %w", err)
		}
	}
	return nil
}

// GetTimeout converts a time string (e.g., "10s", "2m", "1h") to a standardized
// string representation of seconds (e.g., "600s" for "10m").
// It returns an error if the input is in an unsupported format.
func parseToSeconds(s string) (string, error) {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return EmptyAttackDuration, err
	}

	seconds := int(duration.Seconds())
	return fmt.Sprint(seconds) + "s", nil
}

// Returns a cleaned up version of the pod name with spaces and underscores replaced by dashes
func cleanPodName(name string) string {
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", ":", "-")
	return replacer.Replace(name)
}

// Replaces the placeholders in the commands with the actual pod IPs
// Available placeholders: '{{.AttackAddress}}', '{{.TargetAddress}}'
func (s *Scenario) replacePlaceholdersInCommands(deploymentSpec ScenarioDeployment) {
	s.Target.Filter = strings.ReplaceAll(s.Target.Filter, "{{.AttackAddress}}", deploymentSpec.AttackPodSpec.PodIP)
	s.Target.Filter = strings.ReplaceAll(s.Target.Filter, "{{.TargetAddress}}", deploymentSpec.TargetPodSpec.PodIP)
	s.Attacker.AtkCommand = strings.ReplaceAll(s.Attacker.AtkCommand, "{{.TargetAddress}}", deploymentSpec.TargetPodSpec.PodIP)
}
