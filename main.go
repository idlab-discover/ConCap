package main

import (
	"fmt"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
)

func main() {
	fmt.Println("Containercap")
	//capengi.AfpacketCap()
	//capengi.LibpcapCap("eno1")
	kubeapi.DeployExecutor(kubeapi.DeployTemplateBuilder())
}
