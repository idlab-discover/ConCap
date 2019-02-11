package main

import (
	"fmt"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
)

func main() {
	fmt.Println("Containercap")
	podspec := kubeapi.PodTemplateBuilder()
	kubeapi.CreatePod(podspec)
	go kubeapi.ExecCommandInContainer("default", "demo-pod", "tcpdump", "tcpdump", "-i", "lo", "-n", "-w", "/var/h-captures/exp.pcap")
	go kubeapi.ExecShellInContainer("default", "demo-pod", "nmap", "nmap -sS -A -T5 localhost")
	kubeapi.ListPod()
	//kubeapi.UpdatePod()
	kubeapi.DeletePod()
}
