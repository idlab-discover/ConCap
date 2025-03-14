# ConCap

![Demo Video](concap-demo.gif)

`concap` is a framework designed to capture realistic cyberattacks in controlled, containerized environments for the purpose of dataset creation. By creating a scenario file containing an attacker and target(s), `concap` will parse the scenario and execute it. All traffic towards the target(s) will be captured and automatically extracted for flow features. The scenario is executed on a Kubernetes cluster, requiring only a `kubeconfig` in the default location, and results will be downloaded to the machine running the `concap` framework.

## Features

- Execute cyberattack scenarios in a controlled Kubernetes environment.
- Support for both single-target and multi-target scenarios.
- Capture network traffic and extract flow features.
- Fine-grained network flow labeling.
- Automate the creation and management of attack and target pods.
- Download results to the local machine for further (ML) analysis.

## Requirements

- Kubernetes cluster with access configured via `kubeconfig`.
- Go environment for running the framework.
- Docker images for the attack and target pods.

## Installation

Build the repository using the provided build script:

```sh
./build.sh
```

Or use the Makefile:

```sh
make build
```

## Usage

### Flags

- `-d, --dir` (required): The mount path on the host.
- `-w, --workers` (optional): The number of concurrent workers that will execute scenarios, default is `1`.
- `-s, --scenario` (optional): The scenario to run, default is `all`.

### Example Command

```sh
./concap --dir ./example
```

Or use the Makefile:

```sh
make run
```

### Running Scenarios

1. Ensure your Kubernetes cluster is up and running.
2. Place your scenario and processing YAML files in the specified directories.
3. Execute the framework using the command above.
4. The framework will:

   1. Parse the processing and scenario files.
   2. Create the necessary pods.
   3. Asynchronously execute the attacks.
   4. Capture all traffic received by the target(s) to pcap file(s).
   5. Perform flow reconstruction and feature extraction to csv file(s).
   6. When labels are provided in the scenario definition, the csv file(s) are labeled.
   7. Download output files to your machine.

## Scenario Types

Concap supports two types of scenarios:

1. **Single-Target Scenario**: One attacker pod and one target pod.
2. **Multi-Target Scenario**: One attacker pod and multiple target pods.

A scenario file is a YAML file defining the attacker and target(s). The filename must be unique and no more than 58 characters.

### Single-Target Scenario

A single-target scenario consists of one attacker pod and one target pod. This is the default scenario type if no type is specified. The attacker executes commands against the target, and the traffic is captured for analysis.

Example single-target scenario file:

```yaml
type: single-target  # Optional, defaults to single-target if not specified
name: http-flood-attack
attacker:
  name: http-flooder
  image: attacker/http-flooder:latest
  atkCommand: ./http-flood.sh $TARGET_IP 80 100
  atkTime: 30s
  cpuRequest: 100m
  memRequest: 250Mi
target:
  name: web-server
  image: nginx:latest
  filter: "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp"
  cpuRequest: 100m
  memRequest: 250Mi
network:
  bandwidth: 10Mbit
  delay: 20ms
labels:
  label: 1
  category: "dos"
  subcategory: "http-flood"
```

In a single-target scenario, the following environment variables are available in the attack command:
- `$ATTACKER_IP`: IP address of the attacker pod
- `$TARGET_IP`: IP address of the target pod

### Multi-Target Scenario

A multi-target scenario consists of one attacker pod and multiple target pods. This allows for more complex attack scenarios, such as distributed attacks or attacks that target multiple services.

Example multi-target scenario file:

```yaml
type: multi-target  # Required for multi-target scenarios
name: distributed-scan-attack
attacker:
  name: port-scanner
  image: attacker/port-scanner:latest
  atkCommand: ./scan.sh $TARGET_IPS
  atkTime: 60s
  cpuRequest: 100m
  memRequest: 250Mi
targets:
  - name: web-server-1
    image: httpd:2.4.38
    cpuRequest: 100m
    memRequest: 250Mi
    labels:
      service: "web"
      port: "80"
  - name: web-server-2
    image: httpd:2.4.38
    cpuRequest: 100m
    memRequest: 250Mi
    labels:
      service: "web"
      port: "80"
  - name: web-server-3
    image: httpd:2.4.38
    cpuRequest: 100m
    memRequest: 250Mi
    labels:
      service: "web"
      port: "80"
network:  # Global network settings, used as defaults for all targets and attacker
  bandwidth: 100Mbit
  delay: 5ms
labels:  # Global labels, merged with target-specific labels
  label: 1
  category: "scanning"
  subcategory: "port-scan"
```

Note: The global `network` and `labels` fields are used during scenario parsing to set defaults and merge with target-specific configurations.

In a multi-target scenario, the following environment variables are available in the attack command:
- `$ATTACKER_IP`: IP address of the attacker pod
- `$TARGET_IPS`: Comma-separated list of all target IP addresses
- `$TARGET_IP_0`, `$TARGET_IP_1`, etc.: IP addresses of individual target pods (zero-based indexing, where `$TARGET_IP_0` is the first target)

### Deployment Information

When a scenario is executed, the deployment information is captured and included in the output YAML file. For multi-target scenarios, this includes the IP addresses of the attacker and all target pods:

```yaml
deployment:
  attacker: "10.244.0.15"
  target_0: "10.244.0.16"
  target_1: "10.244.0.17"
  target_2: "10.244.0.18"
```

This information is useful for post-processing and analysis of the captured traffic.

### Target-Specific Network Configuration

You can specify target-specific network configurations. The global network configuration serves as a default, and target-specific configurations override these defaults.

```yaml
type: multi-target
name: mixed-network-attack
attacker:
  name: mixed-attacker
  image: attacker/mixed:latest
  atkCommand: ./attack.sh $TARGET_IPS
  atkTime: 60s
targets:
  - name: fast-target
    image: nginx:latest
    network:
      bandwidth: 1Gbit
      delay: 1ms
  - name: slow-target
    image: nginx:latest
    network:
      bandwidth: 10Mbit
      delay: 100ms
network:  # Default network settings for targets without specific settings
  bandwidth: 100Mbit
  delay: 10ms
```

### Target-Specific Labels

Similarly, you can specify target-specific labels that will be merged with the scenario-level labels. Target-specific labels take precedence over global labels.

```yaml
type: multi-target
name: labeled-targets
attacker:
  name: labeled-attacker
  image: attacker/labeled:latest
  atkCommand: ./attack.sh $TARGET_IPS
targets:
  - name: web-target
    image: nginx:latest
    labels:
      service: "web"
      port: "80"
  - name: db-target
    image: postgres:latest
    labels:
      service: "database"
      port: "5432"
labels:  # These labels will be applied to all targets
  attack: "true"
  category: "mixed"
```

### Target-Specific Filters

Each target can have its own custom traffic capture filter. The filter is used by `tcpdump` to determine which packets to capture. You can use special variables in your filter strings that will be automatically replaced with the actual IP addresses during execution:

```yaml
type: multi-target
name: custom-filter-targets
attacker:
  name: port-scanner
  image: attacker/port-scanner:latest
  atkCommand: ./scan.sh $TARGET_IPS
targets:
  - name: web-target
    image: nginx:latest
    filter: "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp"
  - name: db-target
    image: postgres:latest
    filter: "host $TARGET_IP and (host $ATTACKER_IP or host $TARGET_IP_0)"
```

If no filter is specified for a target, the following default filter will be used:

```
((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp
```

This default filter captures all traffic between the attacker and the target, excluding ARP packets.

The following variables are available in filter strings:

- `$ATTACKER_IP`: IP address of the attacker pod
- `$TARGET_IP`: IP address of the current target pod (the one where tcpdump is running)
- `$TARGET_IP_0`, `$TARGET_IP_1`, etc.: IP addresses of specific target pods in the scenario (zero-based indexing)

This allows you to create sophisticated capture filters that can include or exclude traffic between specific pods in your multi-target scenario.

## Processing Pods

Processing pods analyze the traffic received by the target(s) during scenario execution. This traffic is captured by `tcpdump`. Each processing pod requires the following specifications:

- **Name**: A unique identifier for the processing pod. Will be used as filename for output files.
- **Container Image**: The Docker image to be used for the processing pod.
- **Command**: The command that starts the processing of the pcap file.
- **CPU/Memory Request**: Helps K8s with scheduling the pods.

### Command Details

- **Environment Variables**:
  - `$INPUT_FILE`: The file path to the pcap file to be processed.
  - `$OUTPUT_FILE`: The file path where the processing results should be written. This file will be downloaded by `concap`.
  - `$INPUT_FILE_NAME`: A unique value for each scenario, equal to the filename of `$INPUT_FILE` without the '.pcap' extension.

### Important Considerations

- Ensure output files are unique to avoid concurrency issues when multiple scenarios run concurrently. Use `$INPUT_FILE_NAME` to generate unique output file names.

### Example

```yaml
name: cicflowmeter
containerImage: ghcr.io/idlab-discover/concap/cicflowmeter:tools-1.0.0
command: >
  mkdir -p /data/output/$INPUT_FILE_NAME/ &&
  pcapfix $INPUT_FILE -o $INPUT_FILE &&
  reordercap $INPUT_FILE /data/output/$INPUT_FILE_NAME/$INPUT_FILE_NAME_fix.pcap &&
  mv /data/output/$INPUT_FILE_NAME/$INPUT_FILE_NAME_fix.pcap $INPUT_FILE &&
  /CICFlowMeter/bin/cfm $INPUT_FILE /data/output/$INPUT_FILE_NAME/ &&
  mv /data/output/$INPUT_FILE_NAME/$INPUT_FILE_NAME.pcap_Flow.csv $OUTPUT_FILE
```

See `example/processingpods` for more configurations of popular flow exporters such as `argus`, `nfstream`, and `rustiflow`.

## Project Structure

The project is organized as follows:

```
concap/
├── cmd/                      # Command-line applications
│   └── concap/               # Main application
│       └── main.go           # Entry point
├── internal/                 # Private application code
│   ├── controller/           # Controller logic
│   │   └── controller.go     # Scenario scheduling and execution
│   ├── kubernetes/           # Kubernetes interaction
│   │   ├── exec.go           # Pod execution
│   │   ├── api.go            # Kubernetes API interactions
│   │   └── watcher.go        # Pod watching
│   └── scenarios/            # Scenario implementations
│       ├── scenario.go       # Base scenario and interface
│       ├── factory.go        # Scenario factory
│       ├── multi_target.go   # Multi-target scenario
│       ├── network.go        # Network configuration
│       ├── podbuilder.go     # Pod building utilities
│       ├── processingpod.go  # Processing pod logic
│       ├── single_target.go  # Single-target scenario
│       ├── types.go          # Common type definitions
│       └── utils.go          # Utility functions
├── examples/                 # Example scenarios and configurations
├── go.mod                    # Go module file
└── README.md                 # Project README
```
