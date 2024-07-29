// Package scenario handles our scenario specification through a number of structs.
// It also contains the podbuilder which fills in the Kubernetes templates for pods with the appropriate information
// These scenario files are exported to yaml and serve as the starting point of every experiment
package ccap

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	kubeapi "github.com/idlab-discover/concap/kube-api-interaction"
	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
)

const EmptyAttackDuration = ""

type Scenario struct {
	UUID       uuid.UUID          `yaml:"uuid"`
	Name       string             `yaml:"name"`
	StartTime  time.Time          `yaml:"startTime"`
	StopTime   time.Time          `yaml:"stopTime"`
	Attacker   Attacker           `yaml:"attacker"`
	Target     Target             `yaml:"target"`
	Network    Network            `yaml:"network"`
	Labels     map[string]string  `yaml:"labels"`
	OutputDir  string             `yaml:"-"`
	Deployment ScenarioDeployment `yaml:"deployment"`
}

type Attacker struct {
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	AtkCommand string `yaml:"atkCommand"`
	AtkTime    string `yaml:"atkTime"`
}

type Target struct {
	Name   string `yaml:"name"`
	Image  string `yaml:"image"`
	Filter string `yaml:"filter"`
}

type Network struct {
	Bandwidth    string `yaml:"bandwidth"`
	QueueSize    string `yaml:"queueSize"`
	Limit        string `yaml:"limit"`
	Delay        string `yaml:"delay"`
	Jitter       string `yaml:"jitter"`
	Distribution string `yaml:"distribution"`
	Loss         string `yaml:"loss"`
	Corrupt      string `yaml:"corrupt"`
	Duplicate    string `yaml:"duplicate"`
	Seed         string `yaml:"seed"`
}

type ScenarioDeployment struct {
	AttackPodSpec kubeapi.RunningPodSpec
	TargetPodSpec kubeapi.RunningPodSpec
}

const defaultTcpdumpFilter = "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp"

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
		s.Target.Filter = defaultTcpdumpFilter
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
	log.Printf("Starting traffic capture on target pod %v for scenario %v", s.Deployment.TargetPodSpec.PodName, s.Name)
	// Start tcpdump in the target pod, redirect stdo and stde to a log file, and write the pid to a file for later cleanup
	stdo, stde, err := kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, s.Deployment.TargetPodSpec.PodName, "tcpdump",
		`nohup tcpdump --no-promiscuous-mode --immediate-mode --buffer-size=32768 --packet-buffered -n --interface=eth0 -w /data/dump.pcap "`+s.GetTrafficFilter()+`" > /data/tcpdump.log 2>&1 & echo $! > /data/tcpdump.pid`)
	if err != nil {
		return fmt.Errorf("error starting tcpdump in scenario %v, error: %v", s.Name, err)
	}
	if stde != "" {
		log.Printf("%s : %s : stdout: %s\n\t stderr: %s", s.Name, s.Attacker.Name, stdo, stde)
	}

	// 3. Execute the attack
	envVar := s.GetShellEnvVars()
	log.Printf("Executing attack '%v' in scenario %v", s.Attacker.AtkCommand, s.Name)
	s.StartTime = time.Now()
	stdo, stde, err = kubeapi.ExecShellInContainerWithEnvVars(apiv1.NamespaceDefault, s.Deployment.AttackPodSpec.PodName, s.Deployment.AttackPodSpec.ContainerName, s.Attacker.AtkCommand, envVar)
	if err != nil {
		return fmt.Errorf("error executing command in scenario %v, error: %v", s.Name, err)
	}
	if stde != "" {
		log.Printf("%s : %s : stdout: %s\n\t stderr: %s", s.Name, s.Attacker.Name, stdo, stde)
	}
	s.StopTime = time.Now()
	log.Printf("Attack finished in scenario %v", s.Name)

	// 4. Download the pcap capture from the target pod and export the updated scenario file
	_, _, err = kubeapi.ExecShellInContainer(apiv1.NamespaceDefault, s.Deployment.TargetPodSpec.PodName, "tcpdump", `kill -SIGINT $(cat /data/tcpdump.pid) && while ! ps | grep "\[tcpdump\]" 2>/dev/null; do sleep 1; done`) // Stop tcpdump. Workaround for tcpdump becoming a zombie process because spawned by other shell
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

// DeployPods deploys the attacker and target pods for the scenario in a concurrent manner.
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

	// 3. Wait for all pods to be running
	log.Println("Waiting for pods to be running for scenario: ", s.Name)
	wg.Wait()
	log.Println("All pods are running for scenario: ", s.Name)
	close(errChan)

	// 4. Check for errors and return values
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

// Returns the tcpdump filter for the scenario with the placeholders replaced by the actual pod IPs if scenario is already deployed
// Available placeholders: $ATTACKER_IP, $TARGET_IP
func (s *Scenario) GetTrafficFilter() string {
	if s.Deployment != (ScenarioDeployment{}) {
		replacer := strings.NewReplacer(
			"$ATTACKER_IP", s.Deployment.AttackPodSpec.PodIP,
			"$TARGET_IP", s.Deployment.TargetPodSpec.PodIP)
		return replacer.Replace(s.Target.Filter)
	}
	return s.Target.Filter
}

// Returns the environment variables for the shell command to be executed in the pods as a map.
// Available variables: $ATTACKER_IP, $TARGET_IP
func (s *Scenario) GetShellEnvVars() map[string]string {
	envVars := make(map[string]string)
	envVars["ATTACKER_IP"] = s.Deployment.AttackPodSpec.PodIP
	envVars["TARGET_IP"] = s.Deployment.TargetPodSpec.PodIP
	return envVars
}

func (s ScenarioDeployment) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"attacker": s.AttackPodSpec.PodIP,
		"target":   s.TargetPodSpec.PodIP,
	}, nil
}

// Builds the tc command to be executed in the pod to shape the network traffic
// The command is built based on the network configuration in the scenario
// Configuration options that are empty or zero are not added to the tc command
func (n *Network) GetTCCommand() string {
	tcCommand := ""
	if n.Bandwidth != "" {
		// Calculate the burst buffer size based on the bandwidth and a burst duration of 5ms
		bandwidthBitsPerSecond, err := ParseSize(n.Bandwidth)
		if err != nil {
			log.Println("Error parsing bandwidth: ", err)
			return ""
		}
		burst := bandwidthBitsPerSecond * 0.005 / 8
		tcCommand += fmt.Sprintf("tc qdisc add dev eth0 root handle 1: tbf rate %s burst %.f", n.Bandwidth, burst)
		if n.QueueSize != "" {
			tcCommand += fmt.Sprintf(" latency %s", n.QueueSize)
		}
	}

	if n.needsNetem() {
		if tcCommand != "" {
			tcCommand += " && tc qdisc add dev eth0 parent 1:1 netem"
		} else {
			tcCommand += "tc qdisc add dev eth0 root netem"
		}
		tcCommand += n.buildNetemCommand()
	}
	return tcCommand
}

func (n *Network) needsNetem() bool {
	return n.Limit != "" || (n.Delay != "" && n.Delay != "0ms") || (n.Jitter != "" && n.Jitter != "0ms") || n.Distribution != "" || (n.Loss != "" && n.Loss != "0%") || (n.Corrupt != "" && n.Corrupt != "0%") || (n.Duplicate != "" && n.Duplicate != "0%")
}

func (n *Network) buildNetemCommand() string {
	netemCommand := ""
	if n.Limit != "" {
		netemCommand += fmt.Sprintf(" limit %s", n.Limit)
	}
	if (n.Delay != "" && n.Delay != "0ms") || (n.Jitter != "" && n.Jitter != "0ms") {
		netemCommand += fmt.Sprintf(" delay %s", n.Delay)
		if n.Jitter != "" {
			netemCommand += " " + n.Jitter
		}
		if n.Distribution != "" {
			netemCommand += fmt.Sprintf(" distribution %s", n.Distribution)
		}
	}
	if n.Loss != "" && n.Loss != "0%" {
		netemCommand += fmt.Sprintf(" loss random %s", n.Loss)
	}
	if n.Corrupt != "" && n.Corrupt != "0%" {
		netemCommand += fmt.Sprintf(" corrupt %s", n.Corrupt)
	}
	if n.Duplicate != "" && n.Duplicate != "0%" {
		netemCommand += fmt.Sprintf(" duplicate %s", n.Duplicate)
	}
	if n.Seed != "" {
		netemCommand += fmt.Sprintf(" seed %s", n.Seed)
	}
	return netemCommand
}

func ParseSize(size string) (float64, error) {
	// Regular expression to match the numerical part and the unit
	re := regexp.MustCompile(`(?i)^(\d+(\.\d+)?)([kmgte]?bit)?$`)
	matches := re.FindStringSubmatch(size)

	if matches == nil {
		return 0, fmt.Errorf("invalid size: %s", size)
	}

	// Extract the numerical part and the unit
	valueStr := matches[1]
	unit := strings.ToLower(matches[3])

	// Convert the numerical part to a float64
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, err
	}

	// Convert based on the unit
	switch unit {
	case "", "bit":
		return value, nil
	case "kbit":
		return value * 1e3, nil
	case "mbit":
		return value * 1e6, nil
	case "gbit":
		return value * 1e9, nil
	case "tbit":
		return value * 1e12, nil
	default:
		return 0, fmt.Errorf("invalid size unit: %s", unit)
	}
}
