package scenarios

import apiv1 "k8s.io/api/core/v1"

// Constants used across different scenario types
const (
	EmptyAttackDuration  = ""
	SingleTargetType     = "single-target"
	MultiTargetType      = "multi-target"
	DefaultTcpdumpFilter = "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp"
)

// Common type definitions used across different scenario types
type Attacker struct {
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	AtkCommand string `yaml:"atkCommand"`
	AtkTime    string `yaml:"atkTime"`
	CPURequest string `yaml:"cpuRequest"`
	CPULimit   string `yaml:"cpuLimit"`
	MemRequest string `yaml:"memRequest"`
	MemLimit   string `yaml:"memLimit"`
	// Network configuration for this attacker, initially contains attacker-specific settings
	// After YAML parsing, contains the merged configuration (global + attacker-specific)
	Network Network `yaml:"network,omitempty"`
	// Privileged mode for attacker pod
	Privileged bool `yaml:"privileged,omitempty"`
}

type TargetConfig struct {
	Name        string `yaml:"name"`
	Image       string `yaml:"image"`
	CommandArgs string `yaml:"commandArgs"`
	Filter      string `yaml:"filter"`
	CPURequest  string `yaml:"cpuRequest"`
	CPULimit    string `yaml:"cpuLimit"`
	MemRequest  string `yaml:"memRequest"`
	MemLimit    string `yaml:"memLimit"`
	// Network configuration for this target, initially contains target-specific settings
	// After YAML parsing, contains the merged configuration (global + target-specific)
	Network Network `yaml:"network,omitempty"`
	// Labels specific to this target, initially contains target-specific labels
	// After YAML parsing, contains the merged labels (global + target-specific)
	Labels map[string]string `yaml:"labels,omitempty"`
	// RawStartupProbe configuration for the target pod
	RawStartupProbe interface{} `yaml:"startupProbe,omitempty"`
	// Parsed startup probe, not exposed in YAML
	StartupProbe *apiv1.Probe `yaml:"-"`
	// Privileged mode for target pod
	Privileged bool `yaml:"privileged,omitempty"`
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

// ScenarioType is used to determine the type of scenario from YAML
type ScenarioType struct {
	Type string `yaml:"type"`
}
