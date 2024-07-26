// Package kubeapi is our main interface on top of the Kubernetes API
// Currently, we directly control Pods. Alternatively we may opt for the Kubernetes Job abstraction if it provides enough low-level access today
package kubeapi

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type RunningPodSpec struct {
	PodName        string
	ContainerImage string
	ContainerName  string
	PodIP          string
}

var kubeConfig *rest.Config
var kubeClient kubernetes.Clientset
var podsClient v1.PodInterface
var podWatcher PodWatcher

// The init function initializes the Kubernetes API by retrieving the kubeconfig file and creating a new clientset.
// It uses the clientcmd package to retrieve the kubeconfig file from the default location and creates a clientset from it.
// The clientset is then used to instantiate a Pods client and a Deployments client. It also starts the PodWatcher, receiving updates on all pod events.
//
// The function does not take any parameters and does not return any values. It is automatically called when the program starts.
func init() {

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	kubeConf, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(kubeConf)
	if err != nil {
		panic(err)
	}
	kubeConfig = kubeConf
	kubeClient = *clientset
	podsClient = kubeClient.CoreV1().Pods(apiv1.NamespaceDefault)
	podWatcher = NewPodWatcher(podsClient)
	podWatcher.Start(context.Background()) // Start the pod watcher
}

// CreatePod is a function that takes a pointer to a Kubernetes Pod object as input.
// It uses the Kubernetes client to create the new Pod, and returns
// the new Pod object along with any errors encountered during the creation process.
//
// Parameters:
//   - pod: A pointer to the Kubernetes Pod object that needs to be created.
//
// Returns:
//   - A pointer to the new Kubernetes Pod object that was created by the function.
//   - An error if there were any issues encountered during the creation process.
func CreatePod(ctx context.Context, pod *apiv1.Pod) (*apiv1.Pod, error) {
	result, err := podsClient.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CreateRunningPod is a function that creates a Kubernetes Pod and waits for the Pod to enter the Running state using the PodWatcher.
// The function returns a PodSpec containing the relevant specifications of the created Pod.
//
// Parameters:
//   - pod: A pointer to the Kubernetes Pod object that needs to be created.
//
// Returns:
//   - A PodSpec struct containing the specifications of the created Pod.
//   - An error if there were any issues encountered during the Pod creation process.
func CreateRunningPod(pod *apiv1.Pod) (RunningPodSpec, error) {
	result, err := CreatePod(context.Background(), pod)
	if err != nil {
		fmt.Println("Creation of pod failed: " + err.Error())
		return RunningPodSpec{}, err
	}

	log.Printf("Waiting for pod %s to be running...", pod.Name)
	result, err = podWatcher.WaitForPodRunning(context.Background(), result.Name)
	if err != nil {
		fmt.Println("Waiting for pod to be running failed: " + err.Error())
		return RunningPodSpec{}, err
	}

	return SetPodSpec(result), err
}

// SetPodSpec is a helper function that returns a structured object that contains some of the relevant specifications
// of the given Kubernetes Pod. It extracts the Pod's name, container imag, container name, and Pod IP from the provided Pod
// variable, and organizes them into a flat struct of type RunningPodSpec.
//
// Parameters:
//   - pod: A pointer to the Kubernetes Pod whose specifications are to be extracted.
//
// Returns:
//   - A PodSpec object containing the specifications of the given Pod.
func SetPodSpec(pod *apiv1.Pod) RunningPodSpec {
	specs := RunningPodSpec{
		PodName:        pod.Name,
		ContainerImage: pod.Spec.Containers[0].Image,
		ContainerName:  pod.Spec.Containers[0].Name,
		PodIP:          pod.Status.PodIP,
	}
	return specs
}

// DeletePod is a function that takes a string containing the name of a Pod as input.
// It deletes the specified Pod from the Kubernetes cluster using the Kubernetes client.
//
// Parameters:
//   - podName: A string containing the name of the Pod that needs to be deleted.
//
// Returns:
//   - An error if there were any issues encountered during the Pod deletion process.
func DeletePod(podName string) error {
	ctx := context.Background()

	// Delete the pod
	err := podsClient.Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}
	return nil
}

// PodExists checks if a Pod with the specified name exists in the cluster using the Kubernetes client.
// If the Pod exists, the function returns true. If the Pod does not exist, the function returns false.
//
// Parameters:
//   - podName: A string containing the name of the Pod whose existence needs to be checked.
//
// Returns:
//   - A boolean value indicating whether the specified Pod exists or not.
//   - An error if there were any issues encountered during the existence check process.
func PodExists(podName string) (bool, error) {
	_, err := podsClient.Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Pod does not exist
			return false, nil
		}
		// Other error occurred
		return false, err
	}
	// Pod exists
	return true, nil
}

// CopyFileFromPod is a function that copies a file from a specified Pod and container to the local filesystem.
// Parameters:
//   - podName: A string containing the name of the Pod from which the file should be copied.
//   - containerName: A string containing the name of the container from which the file should be copied.
//   - sourcePath: A string containing the path to the file in the Pod that should be copied.
//   - destPath: A string containing the path to the destination file on the local filesystem.
//   - keepFile: A boolean value indicating whether the file should be kept in the Pod after copying.
//
// Returns:
//   - An error if there were any issues encountered during the file copy process.
func CopyFileFromPod(podName string, containerName string, sourcePath string, destPath string, keepFile bool) error {
	// Construct the kubectl cp command
	cmd := exec.CommandContext(context.Background(), "kubectl", "cp", fmt.Sprintf("%s/%s:%s", apiv1.NamespaceDefault, podName, sourcePath), destPath, "-c", containerName)

	// Set the environment variables if needed (e.g., KUBECONFIG)
	// cmd.Env = append(os.Environ(), "KUBECONFIG=/path/to/kubeconfig")

	// Execute the command on the machine running concap
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: %v\n", err)
		log.Printf("Output: %s\n", output)
		return err
	}

	// Delete the file from the Pod if keepFile is set to false
	if !keepFile {
		// Execute the rm command in the Pod to delete the file
		_, stde, err := ExecCommandInContainer(apiv1.NamespaceDefault, podName, containerName, "rm", sourcePath)
		if err != nil {
			return err
		}
		if stde != "" {
			log.Println("Error deleting file from Pod: ", stde)
		}
	}

	log.Println("File downloaded successfully")

	return nil
}

// CopyFileToPod is a function that copies a file to a specified Pod and container from the local filesystem.
// Parameters:
//   - podName: A string containing the name of the Pod to which the file should be copied.
//   - containerName: A string containing the name of the container to which the file should be copied.
//   - sourcePath: A string containing the path to the source file on the local filesystem.
//   - destPath: A string containing the path to the destination file in the Pod.
//
// Returns:
//   - An error if there were any issues encountered during the file copy process.
func CopyFileToPod(podName string, containerName string, sourcePath string, destPath string) error {
	// Construct the kubectl cp command
	cmd := exec.CommandContext(context.Background(), "kubectl", "cp", sourcePath, fmt.Sprintf("%s/%s:%s", apiv1.NamespaceDefault, podName, destPath), "-c", containerName)

	// Set the environment variables if needed (e.g., KUBECONFIG)
	// cmd.Env = append(os.Environ(), "KUBECONFIG=/path/to/kubeconfig")

	// Execute the command on the machine running concap
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: %v\n", err)
		log.Printf("Output: %s\n", output)
		return err
	}

	log.Println("File uploaded successfully")

	return nil
}
