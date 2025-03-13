package scenarios

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
	kubeapi "github.com/idlab-discover/concap/internal/kubernetes"
	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
)

// SingleTargetScenario represents a scenario with one attacker and one target
type SingleTargetScenario struct {
	BaseScenario `yaml:",inline"`
	Attacker     Attacker               `yaml:"attacker"`
	Target       TargetConfig           `yaml:"target"`
	Network      Network                `yaml:"network"`
	Labels       map[string]string      `yaml:"labels"`
	Deployment   SingleTargetDeployment `yaml:"deployment"`
}

type SingleTargetDeployment struct {
	AttackPodSpec kubeapi.RunningPodSpec
	TargetPodSpec kubeapi.RunningPodSpec
}

func (s SingleTargetDeployment) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"attacker": s.AttackPodSpec.PodIP,
		"target":   s.TargetPodSpec.PodIP,
	}, nil
}

// FromYAML parses a YAML file into a SingleTargetScenario
func (s *SingleTargetScenario) FromYAML(filePath string) error {
	fileHandler, err := os.Open(filePath)
	if err != nil {
		log.Println("Failed to open file " + filePath + " : " + err.Error())
		return err
	}
	defer fileHandler.Close()

	b, err := io.ReadAll(fileHandler)
	if err != nil {
		return fmt.Errorf("error reading YAML: %w", err)
	}

	err = yaml.UnmarshalStrict(b, s)
	if err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	if s.Attacker.Image == "" {
		return fmt.Errorf("no attack-image provided for attack: '%s'", s.Attacker.Name)
	}

	// Process attack time
	atkTime, err := ParseToSeconds(s.Attacker.AtkTime)
	if err != nil {
		s.Attacker.AtkTime = EmptyAttackDuration
	} else {
		s.Attacker.AtkTime = atkTime
	}

	// Modify the attack command to include a timeout if a duration is provided
	if s.Attacker.AtkTime != EmptyAttackDuration {
		s.Attacker.AtkCommand = "timeout " + s.Attacker.AtkTime + " " + s.Attacker.AtkCommand
	}
	// Append the command to write output to default container logs
	s.Attacker.AtkCommand += " 2>&1 | tee -a /proc/1/fd/1"

	s.UUID = uuid.New()
	s.Name = CleanPodName(strings.TrimSuffix(filepath.Base(fileHandler.Name()), filepath.Ext(fileHandler.Name())))

	if s.Target.Filter == "" {
		s.Target.Filter = DefaultTcpdumpFilter
	}

	// Default resource requests to help K8s with scheduling
	if s.Attacker.CPURequest == "" {
		s.Attacker.CPURequest = "100m"
	}
	if s.Attacker.MemRequest == "" {
		s.Attacker.MemRequest = "250Mi"
	}
	if s.Target.CPURequest == "" {
		s.Target.CPURequest = "100m"
	}
	if s.Target.MemRequest == "" {
		s.Target.MemRequest = "250Mi"
	}

	// Initialize an empty Network struct for attacker if not provided
	// This ensures MergeNetworks will work correctly
	if s.Attacker.Network == (Network{}) {
		s.Attacker.Network = Network{}
	}

	// Merge the global network configuration with the attacker-specific one
	// Attacker-specific configuration takes precedence over global configuration
	s.Attacker.Network = MergeNetworks(s.Network, s.Attacker.Network)

	// Initialize an empty Network struct for target if not provided
	// This ensures MergeNetworks will work correctly
	if s.Target.Network == (Network{}) {
		s.Target.Network = Network{}
	}

	// Merge the global network configuration with the target-specific one
	// Target-specific configuration takes precedence over global configuration
	s.Target.Network = MergeNetworks(s.Network, s.Target.Network)

	// Initialize empty Labels map for target if not provided
	if s.Target.Labels == nil {
		s.Target.Labels = make(map[string]string)
	}

	// Merge global labels with target-specific labels
	// Target-specific labels take precedence over global labels
	s.Target.Labels = MergeLabels(s.Labels, s.Target.Labels)

	return nil
}

// AttackPod returns the pod definition for the attacker
func (s *SingleTargetScenario) AttackPod() *apiv1.Pod {
	return BuildAttackerPod(s.Attacker.Name, s.Attacker, s.Name)
}

// TargetPod returns the pod definition for the target
func (s *SingleTargetScenario) TargetPod() *apiv1.Pod {
	return BuildTargetPod(s.Target, s.Name, 0)
}

// Execute executes the scenario
func (s *SingleTargetScenario) Execute(outputDir string) error {
	return ExecuteScenario(s, outputDir)
}

// WriteScenario marshals the scenario to YAML and writes it to disk
func (s *SingleTargetScenario) WriteScenario(outputDir string) error {
	return WriteScenario(s, outputDir)
}

// DeployAllPods deploys the attacker and target pods for the scenario in a concurrent manner.
// It waits for all pods to be running before returning.
func (s *SingleTargetScenario) DeployAllPods() error {
	log.Println("Deploying pods for scenario: ", s.Name)
	var wg sync.WaitGroup
	wg.Add(2) // 1 for attacker + 1 for target

	// Channel to capture errors
	errChan := make(chan error, 2)

	// 1. Deploy the attacker pod
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

	// 4. Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// StartTrafficCapture starts traffic capture on the target pod
func (s *SingleTargetScenario) StartTrafficCapture() error {
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
	return nil
}

// ExecuteAttack executes the attack
func (s *SingleTargetScenario) ExecuteAttack() error {
	envVar := s.GetShellEnvVars()
	log.Printf("Executing attack '%v' in scenario %v", s.Attacker.AtkCommand, s.Name)
	s.StartTime = time.Now()
	stdo, stde, err := kubeapi.ExecShellInContainerWithEnvVars(apiv1.NamespaceDefault, s.Deployment.AttackPodSpec.PodName, s.Deployment.AttackPodSpec.ContainerName, s.Attacker.AtkCommand, envVar)
	if err != nil {
		return fmt.Errorf("error executing command in scenario %v, error: %v", s.Name, err)
	}
	if stde != "" {
		log.Printf("%s : %s : stdout: %s\n\t stderr: %s", s.Name, s.Attacker.Name, stdo, stde)
	}
	s.StopTime = time.Now()
	log.Printf("Attack finished in scenario %v", s.Name)
	return nil
}

// DownloadResults downloads the pcap capture and tcpdump log file from the target pod
func (s *SingleTargetScenario) DownloadResults(outputDir string) error {
	pcapPath := filepath.Join(outputDir, "dump.pcap")
	tcpdumpLogPath := filepath.Join(outputDir, "tcpdump.log")
	targetPodName := s.Deployment.TargetPodSpec.PodName

	// Stop tcpdump. Workaround for tcpdump becoming a zombie process because spawned by other shell
	_, _, err := kubeapi.ExecShellInContainer(
		apiv1.NamespaceDefault,
		s.Deployment.TargetPodSpec.PodName,
		"tcpdump",
		`kill -SIGINT $(cat /data/tcpdump.pid) && 
		 while ! ps | grep "\[tcpdump\]" 2>/dev/null; do 
			 sleep 1; 
		 done`,
	)
	if err != nil {
		return fmt.Errorf("failed to stop tcpdump in target pod: %v", err)
	}

	// Download the pcap file and tcpdump log file from the target pod
	log.Printf("Stopped traffic capture on target pod %v for scenario %v", targetPodName, s.Name)
	err = kubeapi.CopyFileFromPod(targetPodName, "tcpdump", "/data/dump.pcap", pcapPath, true)
	if err != nil {
		return fmt.Errorf("failed to download pcap file from target pod: %v", err)
	}
	err = kubeapi.CopyFileFromPod(targetPodName, "tcpdump", "/data/tcpdump.log", tcpdumpLogPath, true)
	if err != nil {
		return fmt.Errorf("failed to download tcpdump log file from target pod: %v", err)
	}

	// Write the finished scenario to output directory
	err = s.WriteScenario(outputDir)
	if err != nil {
		return fmt.Errorf("error writing scenario file: %v", err)
	}

	return nil
}

// ProcessResults processes the results of the attack
func (s *SingleTargetScenario) ProcessResults(outputDir string, processingPods []*ProcessingPod) error {
	log.Printf("Analyzing traffic for scenario %v...", s.Name)
	var wg sync.WaitGroup
	for _, pod := range processingPods {
		wg.Add(1)
		go func(pod *ProcessingPod, scenarioName string, outputDir string, labels map[string]string) {
			defer wg.Done()

			// Add target name as a label
			labels["target"] = s.Target.Name

			err := pod.ProcessPcap(filepath.Join(outputDir, "dump.pcap"), scenarioName, outputDir, labels)
			if err != nil {
				log.Printf("error analysing the pcap at processing pod %v: %v", pod.Name, err)
			}
		}(pod, s.Name, outputDir, s.Target.Labels)
	}
	wg.Wait()
	log.Println("Traffic analysis completed for scenario: ", s.Name)
	return nil
}

// DeleteAllPods deletes all pods for the scenario
func (s *SingleTargetScenario) DeleteAllPods() error {
	// Create a slice of pod names to delete
	podsToDelete := []string{
		s.Deployment.AttackPodSpec.PodName,
		s.Deployment.TargetPodSpec.PodName,
	}

	// Delete each pod in the slice
	for _, podName := range podsToDelete {
		if err := kubeapi.DeletePod(podName); err != nil {
			return fmt.Errorf("failed to delete pod %s: %w", podName, err)
		}
	}
	return nil
}

// GetTrafficFilter returns the tcpdump filter for the scenario with the placeholders replaced by the actual pod IPs
func (s *SingleTargetScenario) GetTrafficFilter() string {
	if s.Deployment != (SingleTargetDeployment{}) {
		replacer := strings.NewReplacer(
			"$ATTACKER_IP", s.Deployment.AttackPodSpec.PodIP,
			"$TARGET_IP", s.Deployment.TargetPodSpec.PodIP)
		return replacer.Replace(s.Target.Filter)
	}
	return s.Target.Filter
}

// GetShellEnvVars returns the environment variables for the shell command to be executed in the pods
func (s *SingleTargetScenario) GetShellEnvVars() map[string]string {
	envVars := make(map[string]string)
	envVars["ATTACKER_IP"] = s.Deployment.AttackPodSpec.PodIP
	envVars["TARGET_IP"] = s.Deployment.TargetPodSpec.PodIP
	return envVars
}
