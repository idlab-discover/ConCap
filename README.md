
# ContainerCap

`containercap` is a framework designed to capture realistic cyberattacks in controlled, containerized environments for the purpose of dataset creation. By creating a scenario file containing an attacker and a target, `containercap` will parse the scenario and execute it. All traffic towards the target will be captured and automatically extracted for flow features. The scenario is executed on a Kubernetes cluster, requiring only a `kubeconfig` in the default location, and results will be downloaded to the machine running the `containercap` framework.

## Features

- Execute cyberattack scenarios in a controlled Kubernetes environment.
- Capture network traffic and extract flow features.
- Automate the creation and management of attack and target pods.
- Download results to the local machine for further analysis.

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
./containercap --dir ./example
```

### Running Scenarios

1. Ensure your Kubernetes cluster is up and running.
2. Place your scenario YAML files in the specified directory.
3. Execute the framework using the command above.
4. The framework will:

    1. Parse the scenario files.
    2. Create the necessary pods.
    3. Asynchronously execute the attacks.
    4. Capture all traffic received by the target to pcap file.
    5. Preform flow reconstruction and feature extraction to csv file.
    6. Download output files to your machine.

## Scenario File

A scenario file is a YAML file defining the attacker and target pods. Below is an example:

```yaml
attacker:
  name: nmap
  image: instrumentisto/nmap:latest
  atkCommand: nmap {{.TargetAddress}} -p 0-80,443,8080 -sV --version-light -T3
  atkTime: 10s # Optional: Leave empty to execute atkCommand until it finishes.
  category: scan # Optional
target:
  name: httpd
  image: httpd:2.4.38
  filter: "((dst host {{.AttackAddress}} and src host {{.TargetAddress}}) or (dst host {{.TargetAddress}} and src host {{.AttackAddress}})) and not arp" # Optional, default
  ports:
  - 80
  category: webserver # Optional
tag: "" # Optional
scenarioType: "" # Optional
labels: # Optional, if present it will be included as extra columns in the flows CSV. Any key, value combination is allowed here.
  label: 1
  category: "scanning"
  subcategory: "nmap"
```

