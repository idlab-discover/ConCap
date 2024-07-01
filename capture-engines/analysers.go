package capengi

import (
	"log"
	"sync"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
)

// JoyProcessing is a helper function that extracts features from a captured .pcap file using the Joy tool
// and saves them as a .joy.json file. It takes in the capture directory, the transform directory, a wait group,
// a PodSpec struct, and a scenario UUID as parameters. It executes a shell command in the Joy container with
// the specified arguments and options. It then adds a label to the pod indicating that it is idle.
//
// Parameters:
//   - captureDir: A string containing the path to the directory where the .pcap file is stored.
//   - transformDir: A string containing the path to the directory where the .joy.json file will be saved.
//   - wg: A pointer to a sync.WaitGroup that synchronizes multiple goroutines.
//   - pod: A PodSpec struct that contains information about the Joy pod.
//   - scenarioUUID: A string containing the unique identifier of the scenario.
func JoyProcessing(captureDir string, transformDir string, wg *sync.WaitGroup, pod kubeapi.PodSpec, scenarioUUID string) {
	defer wg.Done()
	log.Println("JOY: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", pod.Uuid, pod.ContainerName, "./joy username=kali promisc=1 retain=1 count=20 bidir=1 num_pkts=200 dist=1 cdist=none entropy=1 wht=0 example=0 dns=1 ssh=1 tls=1 dhcp=1 dhcpv6=1 http=1 ike=1 payload=1 salt=0 ppi=0 fpx=0 verbosity=4 "+"/ContainerCap/containercap-captures/"+scenarioUUID+"/"+scenarioUUID+".pcap"+" | gunzip > "+"/ContainerCap/containercap-transformed/"+scenarioUUID+"/"+scenarioUUID+".joy.json")
	kubeapi.AddLabelToRunningPod("idle", "true", pod.Uuid)
	log.Println("JOY: finished processing for ", scenarioUUID)
}

// CicProcessing is a helper function that extracts features from a captured .pcap file using the CICFlowMeter tool
// and saves them as .csv files. It takes in the capture directory, the transform directory, a wait group,
// a PodSpec struct, and a scenario UUID as parameters. It executes a shell command in the CICFlowMeter container with
// the specified arguments. It then adds a label to the pod indicating that it is idle.
//
// Parameters:
//   - captureDir: A string containing the path to the directory where the .pcap file is stored.
//   - transformDir: A string containing the path to the directory where the .csv files will be saved.
//   - wg: A pointer to a sync.WaitGroup that synchronizes multiple goroutines.
//   - pod: A PodSpec struct that contains information about the CICFlowMeter pod.
//   - scenarioUUID: A string containing the unique identifier of the scenario
func CicProcessing(captureDir string, transformDir string, wg *sync.WaitGroup, pod kubeapi.PodSpec, scenarioUUID string) {
	defer wg.Done()
	log.Println("CICFlowMeter: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", pod.Uuid, pod.ContainerName, "./cfm "+"/ContainerCap/containercap-captures/"+scenarioUUID+"/"+scenarioUUID+".pcap "+"/ContainerCap/containercap-transformed/"+scenarioUUID+"/")
	kubeapi.AddLabelToRunningPod("idle", "true", pod.Uuid)
	log.Println("CICFlowMeter: finished processing for ", scenarioUUID)
}
