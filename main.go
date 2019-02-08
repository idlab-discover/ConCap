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
	kubeapi.DeletePod()
}
