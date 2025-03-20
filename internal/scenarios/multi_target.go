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

// MultiTargetScenario represents a scenario with one attacker and multiple targets
type MultiTargetScenario struct {
	BaseScenario `yaml:",inline"`
	Attacker     Attacker       `yaml:"attacker"`
	Targets      []TargetConfig `yaml:"targets"`
	// Global network configuration, used as default for all targets
	Network Network `yaml:"network,omitempty"`
	// Global labels, applied to all targets
	Labels     map[string]string     `yaml:"labels,omitempty"`
	Deployment MultiTargetDeployment `yaml:"deployment"`
}

type TargetConfig struct {
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	Filter     string `yaml:"filter"`
	CPURequest string `yaml:"cpuRequest"`
	CPULimit   string `yaml:"cpuLimit"`
	MemRequest string `yaml:"memRequest"`
	MemLimit   string `yaml:"memLimit"`
	// Network configuration for this target, initially contains target-specific settings
	// After YAML parsing, contains the merged configuration (global + target-specific)
	Network Network `yaml:"network"`
	// Labels specific to this target, initially contains target-specific labels
	// After YAML parsing, contains the merged labels (global + target-specific)
	Labels map[string]string `yaml:"labels"`
	// StartupProbe configuration for the target pod
	StartupProbe *apiv1.Probe `yaml:"startupProbe,omitempty"`
}

type MultiTargetDeployment struct {
	AttackPodSpec  kubeapi.RunningPodSpec
	TargetPodSpecs []kubeapi.RunningPodSpec
}

func (s MultiTargetDeployment) MarshalYAML() (interface{}, error) {
	result := map[string]interface{}{
		"attacker": s.AttackPodSpec.PodIP,
		"targets":  map[string]string{},
	}

	// Add each target with its name as key and IP as value
	targetMap := result["targets"].(map[string]string)
	for _, target := range s.TargetPodSpecs {
		targetMap[target.ContainerName] = target.PodIP
	}

	return result, nil
}

// FromYAML parses a YAML file into a MultiTargetScenario
func (s *MultiTargetScenario) FromYAML(filePath string) error {
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

	if s.Attacker.Name == "" {
		s.Attacker.Name = "Attacker"
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

	// Set default filter and resource requests for each target
	for i := range s.Targets {
		if s.Targets[i].Name == "" {
			s.Targets[i].Name = fmt.Sprintf("Target-%d", i)
		}

		if s.Targets[i].Filter == "" {
			s.Targets[i].Filter = DefaultTcpdumpFilter
		}
		if s.Targets[i].CPURequest == "" {
			s.Targets[i].CPURequest = "100m"
		}
		if s.Targets[i].MemRequest == "" {
			s.Targets[i].MemRequest = "250Mi"
		}

		// Initialize an empty Network struct if not provided
		// This ensures MergeNetworks will work correctly
		if s.Targets[i].Network == (Network{}) {
			s.Targets[i].Network = Network{}
		}

		// Merge the global network configuration with the target-specific one
		// Target-specific configuration takes precedence over global configuration
		s.Targets[i].Network = MergeNetworks(s.Network, s.Targets[i].Network)

		// Initialize empty Labels map if not provided
		if s.Targets[i].Labels == nil {
			s.Targets[i].Labels = make(map[string]string)
		}

		// Merge global labels with target-specific labels
		// Target-specific labels take precedence over global labels
		s.Targets[i].Labels = MergeLabels(s.Labels, s.Targets[i].Labels)
		s.Labels = nil // Clear so it is not written to the output YAML file
	}

	// Default resource requests for attacker
	if s.Attacker.CPURequest == "" {
		s.Attacker.CPURequest = "100m"
	}
	if s.Attacker.MemRequest == "" {
		s.Attacker.MemRequest = "250Mi"
	}

	// Initialize an empty Network struct for attacker if not provided
	// This ensures MergeNetworks will work correctly
	if s.Attacker.Network == (Network{}) {
		s.Attacker.Network = Network{}
	}

	// Merge the global network configuration with the attacker-specific one
	// Attacker-specific configuration takes precedence over global configuration
	s.Attacker.Network = MergeNetworks(s.Network, s.Attacker.Network)
	s.Network = Network{} // Clear so it is not written to the output YAML file

	return nil
}

// AttackPod returns the pod definition for the attacker
func (s *MultiTargetScenario) AttackPod() *apiv1.Pod {
	return BuildAttackerPod(s.Attacker.Name, s.Attacker, s.Name)
}

// TargetPod returns the pod definition for a specific target
func (s *MultiTargetScenario) TargetPod(index int) *apiv1.Pod {
	if index < 0 || index >= len(s.Targets) {
		log.Printf("Target index %d out of range (0-%d)", index, len(s.Targets)-1)
		return nil
	}

	return BuildTargetPod(s.Targets[index], s.Name, index)
}

// Execute executes the scenario
func (s *MultiTargetScenario) Execute(outputDir string) error {
	return ExecuteScenario(s, outputDir)
}

// WriteScenario marshals the scenario to YAML and writes it to disk
func (s *MultiTargetScenario) WriteScenario(outputDir string) error {
	return WriteScenario(s, outputDir)
}

// DeployAllPods deploys the attacker and all target pods for the scenario in a concurrent manner.
// It waits for all pods to be running before returning.
func (s *MultiTargetScenario) DeployAllPods() error {
	log.Println("Deploying pods for scenario: ", s.Name)
	var wg sync.WaitGroup
	wg.Add(1 + len(s.Targets)) // 1 for attacker + number of targets

	// Channel to capture errors
	errChan := make(chan error, 1+len(s.Targets))

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

	// Initialize the TargetPodSpecs slice with the correct length
	s.Deployment.TargetPodSpecs = make([]kubeapi.RunningPodSpec, len(s.Targets))

	// 2. Deploy all target pods
	for i := range s.Targets {
		go func(index int) {
			defer wg.Done()
			targetPod := s.TargetPod(index)
			podspec, err := kubeapi.CreateRunningPod(targetPod)
			if err != nil {
				errChan <- fmt.Errorf("failed to deploy target pod %s: %w", s.Targets[index].Name, err)
				return
			}
			s.Deployment.TargetPodSpecs[index] = podspec
		}(i)
	}

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

// StartTrafficCapture starts traffic capture on all target pods
func (s *MultiTargetScenario) StartTrafficCapture() error {
	var wg sync.WaitGroup
	wg.Add(len(s.Deployment.TargetPodSpecs))

	// Channel to capture errors
	errChan := make(chan error, len(s.Deployment.TargetPodSpecs))

	// Start traffic capture on each target pod concurrently
	for i, targetPodSpec := range s.Deployment.TargetPodSpecs {
		go func(index int, podSpec kubeapi.RunningPodSpec) {
			defer wg.Done()

			log.Printf("Starting traffic capture on target pod %v for scenario %v", podSpec.PodName, s.Name)
			// Get the filter for this target
			filter := s.GetTrafficFilterForTarget(index)

			// Start tcpdump in the target pod
			stdo, stde, err := kubeapi.ExecShellInContainer(
				apiv1.NamespaceDefault,
				podSpec.PodName,
				"tcpdump",
				`nohup tcpdump --no-promiscuous-mode --immediate-mode --buffer-size=32768 --packet-buffered -n --interface=eth0 -w /data/dump.pcap "`+filter+`" > /data/tcpdump.log 2>&1 & echo $! > /data/tcpdump.pid`)
			if err != nil {
				errChan <- fmt.Errorf("error starting tcpdump in target %s for scenario %v, error: %v", s.Targets[index].Name, s.Name, err)
				return
			}
			if stde != "" {
				log.Printf("%s : %s : stdout: %s\n\t stderr: %s", s.Name, s.Targets[index].Name, stdo, stde)
			}
		}(i, targetPodSpec)
	}

	// Wait for all traffic captures to start
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAttack executes the attack
func (s *MultiTargetScenario) ExecuteAttack() error {
	// Create environment variables with all target IPs
	envVars := s.GetShellEnvVars()

	log.Printf("Executing attack '%v' in scenario %v", s.Attacker.AtkCommand, s.Name)
	s.StartTime = time.Now()
	stdo, stde, err := kubeapi.ExecShellInContainerWithEnvVars(
		apiv1.NamespaceDefault,
		s.Deployment.AttackPodSpec.PodName,
		s.Deployment.AttackPodSpec.ContainerName,
		s.Attacker.AtkCommand,
		envVars)
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
func (s *MultiTargetScenario) DownloadResults(outputDir string) error {
	var wg sync.WaitGroup
	wg.Add(len(s.Deployment.TargetPodSpecs))

	// Channel to capture errors
	errChan := make(chan error, len(s.Deployment.TargetPodSpecs))

	// Download the pcap capture and tcpdump log file from each target pod concurrently
	for i, targetPodSpec := range s.Deployment.TargetPodSpecs {
		go func(index int, podSpec kubeapi.RunningPodSpec) {
			defer wg.Done()

			// Stop tcpdump. Workaround for tcpdump becoming a zombie process because spawned by other shell
			_, _, err := kubeapi.ExecShellInContainer(
				apiv1.NamespaceDefault,
				podSpec.PodName,
				"tcpdump",
				`kill -SIGINT $(cat /data/tcpdump.pid) && 
				 while ! ps | grep "\[tcpdump\]" 2>/dev/null; do 
					 sleep 1; 
				 done`,
			)
			if err != nil {
				errChan <- fmt.Errorf("failed to stop tcpdump in target pod %s: %v", s.Targets[index].Name, err)
				return
			}

			// Create target-specific output directory
			targetDir := filepath.Join(outputDir, s.Targets[index].Name)
			err = os.MkdirAll(targetDir, 0755)
			if err != nil {
				errChan <- fmt.Errorf("failed to create output directory for target %s: %v", s.Targets[index].Name, err)
				return
			}

			// Download pcap file
			err = kubeapi.CopyFileFromPod(podSpec.PodName, "tcpdump", "/data/dump.pcap", filepath.Join(targetDir, "dump.pcap"), true)
			if err != nil {
				errChan <- fmt.Errorf("failed to download pcap file from target pod %s: %v", s.Targets[index].Name, err)
				return
			}

			// Download tcpdump log
			err = kubeapi.CopyFileFromPod(podSpec.PodName, "tcpdump", "/data/tcpdump.log", filepath.Join(targetDir, "tcpdump.log"), true)
			if err != nil {
				errChan <- fmt.Errorf("failed to download tcpdump log file from target pod %s: %v", s.Targets[index].Name, err)
				return
			}

			log.Printf("Processed results for target %s in scenario %v", s.Targets[index].Name, s.Name)
		}(i, targetPodSpec)
	}

	// Wait for all result processing to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Write the scenario file
	err := s.WriteScenario(outputDir)
	if err != nil {
		return fmt.Errorf("error writing scenario file: %v", err)
	}

	return nil
}

// ProcessResults processes the results of the attack
func (s *MultiTargetScenario) ProcessResults(outputDir string, processingPods []*ProcessingPod) error {
	var wg sync.WaitGroup

	// Process each target's results
	for _, target := range s.Targets {
		targetDir := filepath.Join(outputDir, target.Name)

		// For each target, process with all processing pods
		for _, pod := range processingPods {
			wg.Add(1)
			go func(pod *ProcessingPod, scenarioName string, targetName string, targetDir string, labels map[string]string) {
				defer wg.Done()

				// Add target name as a label
				labels["target"] = targetName

				err := pod.ProcessPcap(filepath.Join(targetDir, "dump.pcap"), scenarioName, targetName, targetDir, labels)
				if err != nil {
					log.Printf("error analysing the pcap for target %s at processing pod %v: %v", targetName, pod.Name, err)
				}
			}(pod, s.Name, target.Name, targetDir, target.Labels)
		}
	}

	wg.Wait()
	return nil
}

// DeleteAllPods deletes all pods for the scenario
func (s *MultiTargetScenario) DeleteAllPods() error {
	// Create a slice to hold all pod names to delete
	podsToDelete := []string{
		s.Deployment.AttackPodSpec.PodName,
	}

	// Add all target pods
	for _, targetPodSpec := range s.Deployment.TargetPodSpecs {
		podsToDelete = append(podsToDelete, targetPodSpec.PodName)
	}

	// Delete each pod in the slice
	for _, podName := range podsToDelete {
		if err := kubeapi.DeletePod(podName); err != nil {
			return fmt.Errorf("failed to delete pod %s: %w", podName, err)
		}
	}

	return nil
}

// GetTrafficFilterForTarget returns the tcpdump filter for a specific target with placeholders replaced
func (s *MultiTargetScenario) GetTrafficFilterForTarget(targetIndex int) string {
	if targetIndex >= len(s.Targets) || targetIndex >= len(s.Deployment.TargetPodSpecs) {
		return ""
	}

	if s.Deployment.AttackPodSpec.PodIP != "" && s.Deployment.TargetPodSpecs[targetIndex].PodIP != "" {
		// Create a replacer with the basic replacements
		replacements := []string{
			"$ATTACKER_IP", s.Deployment.AttackPodSpec.PodIP,
			"$TARGET_IP", s.Deployment.TargetPodSpecs[targetIndex].PodIP,
		}

		// Add replacements for $TARGET_IP_{index} for all available target pods
		for i, targetPodSpec := range s.Deployment.TargetPodSpecs {
			if targetPodSpec.PodIP != "" {
				placeholder := fmt.Sprintf("$TARGET_IP_%d", i)
				replacements = append(replacements, placeholder, targetPodSpec.PodIP)
			}
		}

		replacer := strings.NewReplacer(replacements...)
		return replacer.Replace(s.Targets[targetIndex].Filter)
	}

	return s.Targets[targetIndex].Filter
}

// GetShellEnvVars returns the environment variables for the shell command to be executed in the pods
func (s *MultiTargetScenario) GetShellEnvVars() map[string]string {
	envVars := make(map[string]string)
	envVars["ATTACKER_IP"] = s.Deployment.AttackPodSpec.PodIP

	// Add each target IP as a separate environment variable
	for i, targetPodSpec := range s.Deployment.TargetPodSpecs {
		envVars[fmt.Sprintf("TARGET_IP_%d", i)] = targetPodSpec.PodIP
	}

	// Also add a comma-separated list of all target IPs
	var targetIPs []string
	for _, targetPodSpec := range s.Deployment.TargetPodSpecs {
		targetIPs = append(targetIPs, targetPodSpec.PodIP)
	}
	envVars["TARGET_IPS"] = strings.Join(targetIPs, ",")

	return envVars
}

// MergeLabels merges two label maps, with the second one taking precedence
func MergeLabels(base, override map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy base labels
	for k, v := range base {
		result[k] = v
	}

	// Override with labels from override
	for k, v := range override {
		result[k] = v
	}

	return result
}
