
# ConCap

![Demo Video](concap-demo.gif)

`concap` is a framework designed to capture realistic cyberattacks in controlled, containerized environments for the purpose of dataset creation. By creating a scenario file containing an attacker and a target, `concap` will parse the scenario and execute it. All traffic towards the target will be captured and automatically extracted for flow features. The scenario is executed on a Kubernetes cluster, requiring only a `kubeconfig` in the default location, and results will be downloaded to the machine running the `concap` framework.


## Features

- Execute cyberattack scenarios in a controlled Kubernetes environment.
- Capture network traffic and extract flow features.
- Fine-grained network flow labeling.
- Automate the creation and management of attack and target pods.
- Download results to the local machine for further (ML) analysis.

## Requirements

- Kubernetes cluster with access configured via `kubeconfig`.
- Go environment for running the framework.
- Docker images for the attack and target pods.

## Installation

Build the repository.

```sh
go build
```

## Usage

### Flags

- `-d, --dir` (required): The mount path on the host.
- `-d, --workers` (optional): The number of concurrent workers that will execute scenarios, default is `1`.
- `-s, --scenario` (optional): The scenario to run, default is `all`.

### Example Command

```sh
go run main.go --dir ./example
# OR (after building the repository)
./concap --dir ./example
```

### Running Scenarios

1. Ensure your Kubernetes cluster is up and running.
2. Place your scenario and processing YAML files in the specified directories.
3. Execute the framework using the command above.
4. The framework will:

    1. Parse the processing and scenario files.
    2. Create the necessary pods.
    3. Asynchronously execute the attacks.
    4. Capture all traffic received by the target to pcap file.
    5. Preform flow reconstruction and feature extraction to csv file.
    6. When labels are provided in the scenario definition, the csv file is labeled.
    6. Download output files to your machine.

## Scenario File

A scenario file is a YAML file defining the attacker and target pods. The filename must be unique and no more than 61 characters. Below is an example:

```yaml
# nmap-tcp-syn-version.yaml
attacker:
  name: nmap
  image: instrumentisto/nmap:latest
  atkCommand: nmap $TARGET_IP -p 0-80,443,8080 -sV --version-light -T3
  atkTime: 10s # Optional: Leave empty to execute atkCommand until it finishes.
target:
  name: httpd
  image: httpd:2.4.38
  filter: "((dst host $ATTACKER_IP and src host $TARGET_IP) or (dst host $TARGET_IP and src host $ATTACKER_IP)) and not arp" # Optional, default
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

Processing pods analyze the traffic received by the target during scenario execution. This traffic is captured by `tcpdump`. Each processing pod requires the following specifications:

- **Name**: A unique identifier for the processing pod. Will be used as filename for output files.
- **Container Image**: The Docker image to be used for the processing pod.
- **Command**: The command that starts the processing of the pcap file.

### Command Details

- **Environment Variables**:
  - `$INPUT_FILE`: The file path to the pcap file to be processed.
  - `$OUTPUT_FILE`: The file path where the processing results should be written. This file will be downloaded by `concap`.
  - `$INPUT_FILE_NAME`: A unique value for each scenario, equal to the filename of `$INPUT_FILE` without the '.pcap' extension.

### Important Considerations

- Ensure output files are unique to avoid concurrency issues when multiple scenarios run concurrently. Use `$INPUT_FILE_NAME` to generate unique output file names.

### Examples

```yaml
name: cicflowmeter
containerImage: mielverkerken/cicflowmeter:latest
command: "mkdir -p /data/output/$INPUT_FILE_NAME/ && /CICFlowMeter/bin/cfm $INPUT_FILE /data/output/$INPUT_FILE_NAME/ && mv /data/output/$INPUT_FILE_NAME/$INPUT_FILE_NAME.pcap_Flow.csv $OUTPUT_FILE"
```

```yaml
name: rustiflow
containerImage: ghcr.io/matissecallewaert/rustiflow:slim
command: "rustiflow pcap cic-flow 120 $INPUT_FILE csv $OUTPUT_FILE"
```
