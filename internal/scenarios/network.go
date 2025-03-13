package scenarios

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// GetTCCommand builds the tc command to be executed in the pod to shape the network traffic
// The command is built based on the network configuration in the scenario
// Configuration options that are empty or zero are not added to the tc command
func (n *Network) GetTCCommand() string {
	tcCommand := "tc qdisc show dev eth0" // Show the current qdisc configuration in the Kubernetes logs
	if n.Bandwidth != "" {
		// Calculate the burst buffer size based on the bandwidth and a burst duration of 5ms
		bandwidthBitsPerSecond, err := ParseSize(n.Bandwidth)
		if err != nil {
			log.Println("Error parsing bandwidth: ", err)
			return ""
		}
		burst := bandwidthBitsPerSecond * 0.005 / 8
		tcCommand += fmt.Sprintf(" && tc qdisc replace dev eth0 root handle 1: tbf rate %s burst %.f", n.Bandwidth, burst)
		if n.QueueSize != "" {
			tcCommand += fmt.Sprintf(" latency %s", n.QueueSize)
		}
	}

	if n.needsNetem() {
		if n.Bandwidth != "" {
			tcCommand += " && tc qdisc replace dev eth0 parent 1:1 netem"
		} else {
			tcCommand += " && tc qdisc replace dev eth0 root netem"
		}
		tcCommand += n.buildNetemCommand()
	}
	return tcCommand
}

// needsNetem checks if netem configuration is needed
func (n *Network) needsNetem() bool {
	return n.Limit != "" || (n.Delay != "" && n.Delay != "0ms") || (n.Jitter != "" && n.Jitter != "0ms") || n.Distribution != "" || (n.Loss != "" && n.Loss != "0%") || (n.Corrupt != "" && n.Corrupt != "0%") || (n.Duplicate != "" && n.Duplicate != "0%")
}

// buildNetemCommand builds the netem part of the tc command
func (n *Network) buildNetemCommand() string {
	netemCommand := ""
	if n.Limit != "" {
		netemCommand += fmt.Sprintf(" limit %s", n.Limit)
	}
	if (n.Delay != "" && n.Delay != "0ms") || (n.Jitter != "" && n.Jitter != "0ms") {
		netemCommand += fmt.Sprintf(" delay %s", n.Delay)
		if n.Jitter != "" && n.Jitter != "0ms" {
			netemCommand += " " + n.Jitter
			if n.Distribution != "" {
				netemCommand += fmt.Sprintf(" distribution %s", n.Distribution)
			}
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

// ParseSize parses a size string (e.g., "10Mbit") to a float64 value in bits per second
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

// MergeNetworks merges two Network configurations, with the second one taking precedence
// If a field in the second network is empty, the value from the first network is used
func MergeNetworks(base, override Network) Network {
	result := base

	// Override fields that are set in the override network
	if override.Bandwidth != "" {
		result.Bandwidth = override.Bandwidth
	}
	if override.QueueSize != "" {
		result.QueueSize = override.QueueSize
	}
	if override.Limit != "" {
		result.Limit = override.Limit
	}
	if override.Delay != "" {
		result.Delay = override.Delay
	}
	if override.Jitter != "" {
		result.Jitter = override.Jitter
	}
	if override.Distribution != "" {
		result.Distribution = override.Distribution
	}
	if override.Loss != "" {
		result.Loss = override.Loss
	}
	if override.Corrupt != "" {
		result.Corrupt = override.Corrupt
	}
	if override.Duplicate != "" {
		result.Duplicate = override.Duplicate
	}
	if override.Seed != "" {
		result.Seed = override.Seed
	}

	return result
}
