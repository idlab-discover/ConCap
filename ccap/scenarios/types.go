package ccap

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
}

type Target struct {
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	Filter     string `yaml:"filter"`
	CPURequest string `yaml:"cpuRequest"`
	CPULimit   string `yaml:"cpuLimit"`
	MemRequest string `yaml:"memRequest"`
	MemLimit   string `yaml:"memLimit"`
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
