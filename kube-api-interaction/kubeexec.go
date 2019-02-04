package kubeapi

import (
	"bytes"
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// execute any command inside a pod
func ExecCommand(ns string, podname string, containername string, command ...string) (string, error) {
	fmt.Println("executing command", strings.Join(command, " "))

	var (
		execOut bytes.Buffer
		execErr bytes.Buffer
	)

	// pod, err := kubeClient.CoreV1().Pods(ns).Get(podname, metav1.GetOptions{})
	// if err != nil {
	// 	return "", fmt.Errorf("could not get pod info: %v", err)
	// }

	req := kubeClient.RESTClient().Post().
		Resource("pods").
		Name(podname).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&apiv1.PodExecOptions{
			Container: containername,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(kubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("could not execute: %v", err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &execOut,
		Stderr: &execErr,
		Tty:    false,
	})

	if err != nil {
		return "", fmt.Errorf("could not execute: %v", err)
	}

	if execErr.Len() > 0 {
		return "", fmt.Errorf("stderr: %v", execErr.String())
	}

	return execOut.String(), nil
}
