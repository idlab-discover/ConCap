package main

import (
	"fmt"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
)

func main() {
	fmt.Println("Containercap")

	podspec := kubeapi.PodTemplateBuilder()
	kubeapi.CreatePod(podspec)
	kubeapi.UpdatePod()
	kubeapi.ListPod()
	// kubeapi.ExecCommandInContainer("default", "demo-pod", "tcpdump", "tcpdump -i lo")
	// fmt.Println(kubeapi.ExecShellInContainer("default", "demo-pod", "nmap", "nmap -sS -A -T5 localhost"))
	fmt.Println(kubeapi.ExecShellInContainer("default", "demo-pod", "tcpdump", "echo hi > /var/captures/ax"))
	fmt.Println(kubeapi.ExecShellInContainer("default", "demo-pod", "tcpdump", "echo hi > /var/pv-captures/ax"))
	kubeapi.DeletePod()
}
