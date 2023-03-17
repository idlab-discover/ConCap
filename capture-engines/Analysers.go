package capengi

import (
	"fmt"
	"sync"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
)

// joyProcessing is called after an experiment pod with scenarioUUID is done to extract features from the captured .pcap file
// it sends the command to the long-living joy container
// the shell argument is very specific and should not be modified
func JoyProcessing(captureDir string, transformDir string, wg *sync.WaitGroup, pod kubeapi.PodSpec, scenarioUUID string) {
	defer wg.Done()
	fmt.Println("JOY: received order for ", scenarioUUID)
	//fmt.Println(captureDir + "/" + scenarioUUID + ".pcap")
	kubeapi.ExecShellInContainer("default", pod.Uuid, pod.ContainerName, "./joy username=kali promisc=1 retain=1 count=20 bidir=1 num_pkts=200 dist=1 cdist=none entropy=1 wht=0 example=0 dns=1 ssh=1 tls=1 dhcp=1 dhcpv6=1 http=1 ike=1 payload=1 salt=0 ppi=0 fpx=0 verbosity=4 "+captureDir+"/"+scenarioUUID+".pcap"+" | gunzip > "+transformDir+"/"+scenarioUUID+".joy.json") // We can use a extra filter here like this: bpf='host 10.32.0.8 and host 10.32.0.9 and not arp' => We should make a seperate file for getting the filter
	kubeapi.AddLabelToRunningPod("idle", "true", pod.Uuid)
}

// cicProcessing is called after an experiment pod with scenarioUUID is done to extract features from the captured .pcap file
// it sends the command to the long-living joy container
// the shell argument should not be modified
func CicProcessing(captureDir string, transformDir string, wg *sync.WaitGroup, pod kubeapi.PodSpec, scenarioUUID string) {
	defer wg.Done()
	fmt.Println("CIC: received order for ", scenarioUUID)
	kubeapi.ExecShellInContainer("default", pod.Uuid, pod.ContainerName, "./cfm "+captureDir+"/"+scenarioUUID+".pcap "+transformDir)
	kubeapi.AddLabelToRunningPod("idle", "true", pod.Uuid)
}
