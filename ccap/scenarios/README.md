# Concap Scenarios

This package implements a flexible scenario system for the Concap (Container Capture) tool. It allows defining different types of attack scenarios with varying configurations.

## Scenario Types

The system currently supports two types of scenarios:

1. **Single Target Scenario**: One attacker pod and one target pod.
2. **Multi-Target Scenario**: One attacker pod and multiple target pods.

## Architecture

The scenario system uses the Strategy design pattern to provide a common interface for all scenario types while allowing each type to have its own specific implementation.

### Key Components

- **ScenarioInterface**: Defines the common interface for all scenario types.
- **BaseScenario**: Contains common fields and methods shared by all scenario types.
- **SingleTargetScenario**: Implementation for scenarios with one attacker and one target.
- **MultiTargetScenario**: Implementation for scenarios with one attacker and multiple targets.
- **Factory**: Creates the appropriate scenario type based on the YAML configuration.
- **PodBuilder**: Provides reusable functions for building Kubernetes pod definitions.

### File Structure

- **base_scenario.go**: Contains the ScenarioInterface and BaseScenario definitions.
- **single_target.go**: Implementation of the SingleTargetScenario.
- **multi_target.go**: Implementation of the MultiTargetScenario.
- **factory.go**: Factory function to create the appropriate scenario type.
- **types.go**: Common type definitions used across different scenario types.
- **utils.go**: Utility functions used across different scenario types.
- **podbuilder.go**: Functions for building Kubernetes pod definitions.
- **network.go**: Network-related functionality for traffic shaping.

## Usage

### Creating a Scenario

Scenarios are defined in YAML files. The `type` field determines which scenario type will be used.

```go
// Create a scenario from a YAML file
scenario, err := scenarios.CreateScenario("path/to/scenario.yaml")
if err != nil {
    log.Fatalf("Failed to create scenario: %v", err)
}

// Execute the scenario
err = scenario.Execute()
if err != nil {
    log.Fatalf("Failed to execute scenario: %v", err)
}
```

### Example YAML Files

#### Single Target Scenario

```yaml
type: single-target
name: http-flood-attack
attacker:
  name: http-flooder
  image: attacker/http-flooder:latest
  atkCommand: ./http-flood.sh $TARGET_IP 80 100
  atkTime: 30s
target:
  name: web-server
  image: nginx:latest
network:
  bandwidth: 10Mbit
  delay: 20ms
```

#### Multi-Target Scenario

```yaml
type: multi-target
name: distributed-scan-attack
attacker:
  name: port-scanner
  image: attacker/port-scanner:latest
  atkCommand: ./scan.sh $TARGET_IPS
  atkTime: 60s
targets:
  - name: web-server
    image: nginx:latest
  - name: database
    image: postgres:latest
  - name: cache
    image: redis:latest
network:
  bandwidth: 100Mbit
  delay: 5ms
```

## Extending the System

To add a new scenario type:

1. Create a new struct that embeds `BaseScenario`
2. Implement all methods required by the `ScenarioInterface`
3. Update the factory to recognize and create the new scenario type

## Pod Building

The pod building functionality is implemented in a reusable way:

1. **BuildAttackerPod**: Creates a pod definition for an attacker.
2. **BuildTargetPod**: Creates a pod definition for a target.
3. **BuildTargetPodFromConfig**: Creates a pod definition for a target from a TargetConfig (used in multi-target scenarios).

These functions are used by both SingleTargetScenario and MultiTargetScenario to create the necessary Kubernetes pod definitions.

## Environment Variables

The following environment variables are available in attack commands:

- **Single Target Scenario**:
  - `$ATTACKER_IP`: IP address of the attacker pod
  - `$TARGET_IP`: IP address of the target pod

- **Multi-Target Scenario**:
  - `$ATTACKER_IP`: IP address of the attacker pod
  - `$TARGET1_IP`, `$TARGET2_IP`, etc.: IP addresses of individual target pods
  - `$TARGET_IPS`: Comma-separated list of all target IPs 