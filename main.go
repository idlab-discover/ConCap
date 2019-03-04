package main

import (
	"fmt"
	"log"
	"os"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

func main() {
	fmt.Println("Containercap")
	file, err := os.Open("scenario-test.yaml")
	if err != nil {
		log.Fatal(err)
	}
	scenario := scenario.BuildScenario(file)
	ledger.Register(scenario)
	// podspec := kubeapi.PodTemplateBuilder()
	// kubeapi.CreatePod(podspec)
	// go kubeapi.ExecCommandInContainer("default", "demo-pod", "tcpdump", "tcpdump", "-i", "lo", "-n", "-w", "/var/h-captures/exp.pcap")
	// go kubeapi.ExecShellInContainer("default", "demo-pod", "nmap", "nmap -sS -A -T5 localhost")
	// kubeapi.ListPod()
	// //kubeapi.UpdatePod()
	// kubeapi.DeletePod()
}
