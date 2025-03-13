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
  - name: web-server
    image: nginx:latest
    filter: "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp"
    cpuRequest: 100m
    memRequest: 250Mi
    labels:
      service: "web"
  - name: database
    image: postgres:latest
    filter: "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp"
    cpuRequest: 100m
    memRequest: 250Mi
    labels:
      service: "database"
  - name: cache
    image: redis:latest
    filter: "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp"
    cpuRequest: 100m
    memRequest: 250Mi
    labels:
      service: "cache"
network:
  bandwidth: 100Mbit
  delay: 5ms
labels:
  label: 1
  category: "scanning"
  subcategory: "port-scan"
```

In a multi-target scenario, the following environment variables are available in the attack command:
- `$ATTACKER_IP`: IP address of the attacker pod
- `$TARGET_IPS`: Comma-separated list of all target IP addresses
- `$TARGET_IP_1`, `$TARGET_IP_2`, etc.: IP addresses of individual target pods

### Target-Specific Network Configuration

In multi-target scenarios, you can also specify target-specific network configurations:

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

In multi-target scenarios, you can also specify target-specific labels that will be merged with the scenario-level labels:

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

## Scenario File Format

A scenario file is a YAML file defining the attacker and target(s). The filename must be unique and no more than 61 characters. Below is an example of a single-target scenario:

```yaml
# nmap-tcp-syn-version.yaml
type: single-target  # Optional, defaults to single-target if not specified
attacker:
  name: nmap
  image: instrumentisto/nmap:latest
  atkCommand: nmap $TARGET_IP -p 0-80,443,8080 -sV --version-light -T3
  atkTime: 10s # Optional: Leave empty to execute atkCommand until it finishes.
  cpuRequest: 100m # Default value: helps K8s with scheduling
  memRequest: 100Mi # Default value: helps K8s with scheduling
  cpuLimit: 500m # Optional: empty for no limits
  memLimit: 500Mi # Optional: empty for no limits
target:
  name: httpd
  image: httpd:2.4.38
  filter: "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp" # Optional, default
  cpuRequest: 100m # Default value: helps K8s with scheduling
  memRequest: 100Mi # Default value: helps K8s with scheduling
  cpuLimit: 500m # Optional: empty for no limits
  memLimit: 500Mi # Optional: empty for no limits
network: # Optional, uses tc to emulate a realistic network and requires kernel module sch_netem on nodes in the K8s cluster, install with modprobe sch_netem
  bandwidth: 1gbit # kbit, mbit, gbit
  queueSize: 100ms # us, ms, s
  limit: 10000
  delay: 0ms # latency is sum of delay and jitter
  jitter: 0ms # jitter may cause reordering of packets
  distribution: normal # uniform, normal, pareto or paretonormal
  loss: 0%
  corrupt: 0%
  duplicate: 0%
  seed: 0 # Seed used to reproduce the randomly generated loss or corruption events
labels: # Optional, if present it will be included as extra columns in the flows CSV. Any key, value combination is allowed here.
  label: 1
  category: "scanning"
  subcategory: "nmap"
```

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
├── docs/                     # Documentation
├── go.mod                    # Go module file
└── README.md                 # Project README
```
