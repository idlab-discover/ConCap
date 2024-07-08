package kubeapi

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type ExecOptions struct {
	Command            []string
	Namespace          string
	PodName            string
	ContainerName      string
	Stdin              io.Reader
	CaptureStdout      bool
	CaptureStderr      bool
	PreserveWhitespace bool // If false, whitespace in std{err,out} will be removed.
}

// ExecWithOptions is a function that executes a command in a specified container using the Kubernetes API.
// The function takes an ExecOptions struct as input, which includes the name of the Pod and Container to execute the command in,
// the command to execute, and additional options such as whether to capture stdout and stderr, whether to preserve whitespace in output,
// and whether to use a TTY.
//
// Parameters:
//   - options: an ExecOptions struct containing the necessary information to execute the command.
//
// Returns:
//   - stdout: a string containing the standard output of the command execution.
//   - stderr: a string containing the standard error of the command execution.
//   - err: an error object that indicates whether an error occurred while executing the command.
func ExecWithOptions(options ExecOptions) (string, string, error) {
	const tty = false
	req := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	req.VersionedParams(&apiv1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       tty,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err := execute("POST", req.URL(), kubeConfig, options.Stdin, &stdout, &stderr, tty)

	if options.PreserveWhitespace {
		return stdout.String(), stderr.String(), err
	}

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// ExecCommandInContainerWithFullOutput is a function that executes a command in a specified container and returns the stdout,
// stderr, and error using the Kubernetes client. It uses the ExecWithOptions function to execute the command at the API level.
// The function takes the namespace, pod name, container name, and command as input parameters.
//
// Parameters:
//   - nameSpace: A string representing the namespace of the pod.
//   - podName: A string representing the name of the pod.
//   - containerName: A string representing the name of the container.
//   - cmd: A variadic parameter of strings representing the command to be executed in the container.
//
// Returns:
//   - stdout: A string containing the standard output of the executed command.
//   - stderr: A string containing the standard error output of the executed command.
//   - error: An error object indicating any errors encountered while executing the command.
func ExecCommandInContainer(nameSpace, podName, containerName string, cmd ...string) (string, string, error) {
	return ExecWithOptions(ExecOptions{
		Command:            cmd,
		Namespace:          nameSpace,
		PodName:            podName,
		ContainerName:      containerName,
		Stdin:              nil,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	})
}

// ExecShellInContainer is a function that launches a command in the specified container using sh.
// It takes the namespace, pod name, container name, and command as input parameters and returns the stdout and stderr outputs of the command.
// This function is used in the main function.
//
// Parameters:
//   - nameSpace: A string containing the name of the namespace in which the pod is running.
//   - podName: A string containing the name of the pod in which the container is running.
//   - containerName: A string containing the name of the container in which the command is to be executed.
//   - cmd: A string containing the command to be executed using sh.
//
// Returns:
//   - stdout: A string containing the standard output of the executed command.
//   - stderr: A string containing the standard error output of the executed command.
func ExecBashInContainer(nameSpace, podName, containerName, cmd string) (string, string, error) {
	return ExecCommandInContainer(nameSpace, podName, containerName, "/bin/bash", "-c", cmd)
}

// ExecShellInContainer is a function that launches a command in the specified container using sh.
// It takes the namespace, pod name, container name, and command as input parameters and returns the stdout and stderr outputs of the command.
// This function is used in the main function.
//
// Parameters:
//   - nameSpace: A string containing the name of the namespace in which the pod is running.
//   - podName: A string containing the name of the pod in which the container is running.
//   - containerName: A string containing the name of the container in which the command is to be executed.
//   - cmd: A string containing the command to be executed using sh.
//
// Returns:
//   - stdout: A string containing the standard output of the executed command.
//   - stderr: A string containing the standard error output of the executed command.
func ExecShellInContainer(nameSpace, podName, containerName, cmd string) (string, string, error) {
	return ExecCommandInContainer(nameSpace, podName, containerName, "/bin/sh", "-c", cmd)
}

func ExecShellInContainerWithEnvVars(namespace string, podName string, containerName string, cmd string, envVars map[string]string) (string, string, error) {
	commandWithVars := []string{"env"}
	for key, value := range envVars {
		commandWithVars = append(commandWithVars, key+"="+value)
	}
	commandWithVars = append(commandWithVars, "/bin/sh", "-c", cmd)
	return ExecCommandInContainer(namespace, podName, containerName, commandWithVars...)
}

// execute is a helper function used internally to execute a command in a container.
// It uses SPDY protocol to wrap the command.
// SPDY is deprecated as a protocol, so this may have to change if kubenetes drops it
//
// Parameters:
//   - method: The HTTP method to use for the request (e.g. POST, GET, PUT).
//   - url: The URL to execute the command against.
//   - config: The Kubernetes REST config.
//   - stdin: An io.Reader to read input from for the command.
//   - stdout: An io.Writer to write standard output to.
//   - stderr: An io.Writer to write standard error to.
//   - tty: A boolean indicating whether to allocate a TTY for the command.
//
// Returns:
//   - An error if one occurs during execution.
func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})

}
