// Package kubeapi is our main interface on top of the Kubernetes API
// Currently, we directly control Pods. Alternatively we may opt for the Kubernetes Job abstraction if it provides enough low-level access today
package kubeapi

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	apps "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

var kubeConfig *rest.Config
var kubeClient kubernetes.Clientset
var podsClient v1.PodInterface
var deploymentsClient apps.DeploymentInterface
var listMutex sync.Mutex

type PodSpec struct {
	Name          string
	Uuid          string
	Image         string
	Category      string
	ScnType       string
	ContainerCap  string
	ContainerName string
	PodIP         string
}

// The init function initializes the Kubernetes API by retrieving the kubeconfig file and creating a new clientset.
// It uses the clientcmd package to retrieve the kubeconfig file from the default location and creates a clientset from it.
// The clientset is then used to instantiate a Pods client and a Deployments client.
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
	deploymentsClient = kubeClient.AppsV1().Deployments(apiv1.NamespaceDefault)
}

// CreatePod is a function that takes a pointer to a Kubernetes Pod object as input.
// It creates a new context with a timeout of 5 seconds, uses the Kubernetes client to create the new Pod, and returns
// the new Pod object along with any errors encountered during the creation process.
//
// Parameters:
//   - pod: A pointer to the Kubernetes Pod object that needs to be created.
//
// Returns:
//   - A pointer to the new Kubernetes Pod object that was created by the function.
//   - An error if there were any issues encountered during the creation process.
func CreatePod(pod *apiv1.Pod) (*apiv1.Pod, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Use the Kubernetes client to create the new pod.
	result, err := podsClient.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// Return the new pod object and any errors that were encountered.
	return result, nil
}

// CreateRunningPod is a function that creates a Kubernetes Pod and waits for the Pod to enter the Running state using CheckPodStatus.
// If the reusable parameter is set to true, it attempts to find an idle Pod that matches the image specified in the Pod's container,
// and reuse it instead of creating a new Pod.
// The function returns a PodSpec containing the relevant specifications of the created/reused Pod, as well as a boolean value indicating
// whether a Pod was reused or not.
//
// Parameters:
//   - pod: A pointer to the Kubernetes Pod object that needs to be created.
//   - reusable: A boolean value indicating whether an idle Pod that matches the container image of the given Pod should be reused or not.
//
// Returns:
//   - A PodSpec struct containing the specifications of the created/reused Pod.
//   - A boolean value indicating whether a Pod was reused or not.
//   - An error if there were any issues encountered during the Pod creation process.
func CreateRunningPod(pod *apiv1.Pod, reusable bool) (PodSpec, bool, error) {

	//Declaring variables
	var specs PodSpec = PodSpec{} // Stores the PodSpec of created/reused Pod
	reused := false
	var err error

	if reusable {
		specs, err = findIdlePodForAttacker(pod.Spec.Containers[0].Image)
		if err != nil {
			fmt.Println("Error finding pod: " + err.Error())
		}
	}
	if (specs != PodSpec{}) {
		AddLabelToRunningPod("idle", "false", specs.Uuid) //Adds label "idle : false" for this pod which has been reused.
		reused = true                                     //Sets reuse as true.

	} else {
		result, err := CreatePod(pod) //Creating new pod
		if err != nil {
			fmt.Println("Creation of pod failed: " + err.Error())
			return specs, false, err
		}
		podStates := make(chan bool, 64) //Used to get pod status
		log.Println("KubeAPI: Checking Pod Status Name:" + result.Name)

		go CheckPodsStatus(podStates, result.Name) //Concurrently goes on to check the pod status

		for msg := range podStates { //Iterating over channel of podStates
			if msg {
				pod, err := podsClient.Get(context.Background(), result.Name, metav1.GetOptions{})
				if err != nil {
					fmt.Println("Checking pod status failed: " + err.Error())
					return specs, false, err
				}
				result = pod //Set result as got from client.Get()
			} else {
				time.Sleep(10 * time.Second)               //Wait for 10 seconds
				go CheckPodsStatus(podStates, result.Name) //Repeatedly checks for pod state asynchronously
			}
		}

		//When everything works fine, create a pod specification
		specs = SetPodSpec(result)
	}
	return specs, reused, err
}

// SetPodSpec is a helper function that returns a structured object that contains some of the relevant specifications
// of the given Kubernetes Pod. It extracts the Pod's name, UUID, container image, category,
// scenario type, container capability, container name, and Pod IP from the provided Pod
// variable, and organizes them into a struct of type PodSpec.
//
// Parameters:
//   - pod: A pointer to the Kubernetes Pod whose specifications are to be extracted.
//
// Returns:
//   - A PodSpec object containing the specifications of the given Pod.
func SetPodSpec(pod *apiv1.Pod) PodSpec {
	specs := PodSpec{
		Name:          pod.Spec.Containers[0].Name,
		Uuid:          pod.Name,
		Image:         pod.Spec.Containers[0].Image,
		Category:      pod.ObjectMeta.Labels["category"],
		ScnType:       pod.ObjectMeta.Labels["scenarioType"],
		ContainerCap:  pod.ObjectMeta.Labels["containercap"],
		ContainerName: pod.Spec.Containers[0].Name,
		PodIP:         pod.Status.PodIP,
	}
	return specs
}

// UpdatePod can changes pod options / attributes
// Currently, this funciton is not used, because we instantiate the pods fully for single uses
func UpdatePod() {
	fmt.Println("Updating pod...")
	//    You have two options to Update() this Deployment:
	//
	//    1. Modify the "deployment" variable and call: Update(deployment).
	//       This works like the "kubectl replace" command and it overwrites/loses changes
	//       made by other clients between you Create() and Update() the object.
	//    2. Modify the "result" returned by Get() and retry Update(result) until
	//       you no longer get a conflict error. This way, you can preserve changes made
	//       by other clients between Create() and Update(). This is implemented below
	//			 using the retry utility package included with client-go. (RECOMMENDED)
	//
	// More Info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		result, getErr := podsClient.Get(ctx, "demo-pod", metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("failed to get latest version of pod: %v", getErr))
		}

		result.Spec.Containers[0].Image = "nginx:1.13" // change nginx version
		ctxu, cancelu := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelu()
		_, updateErr := podsClient.Update(ctxu, result, metav1.UpdateOptions{})
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("update failed: %v", retryErr))
	}
	fmt.Println("Updated pod...")
}

// The function ListPod prints a list of pods with their statuses in the default namespace
// Not Used => can be used to improve scenario tracking
func ListPod() {
	fmt.Println("Listing pods in namespace: " + apiv1.NamespaceDefault)

	// Create a context object with a timeout of 5 seconds and defer its cancellation until the function returns
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Use the podsClient to retrieve a list of pods with default options using the created context object
	list, err := podsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err) // Panic if an error occurs while getting the list of pods
	}

	// Iterate through the list of pods and print each pod's container statuses
	for k, d := range list.Items {
		fmt.Printf(" %d %+v \n", k, d.Status.ContainerStatuses)
	}
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

// WatchPod gets the current event chain and prints the info
// Not used
func WatchPod() {
	fmt.Println("Watching pods...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	watch, err := podsClient.Watch(ctx, metav1.ListOptions{Watch: true})
	if err != nil {
		panic(err)
	}
	eventChan := watch.ResultChan()
	for event := range eventChan {
		fmt.Println(event)
	}
}

// CheckPodsStatus is a function that takes a channel of boolean values and a slice of Pod names as input.
// It retrieves the status of each Pod with the provided names using the Kubernetes client.
// If a Pod is found to be in the Running state, the function sends a message to the channel indicating that the Pod is ready.
// If a Pod is not found to be in the Running state, the function sends a message to the channel indicating that the Pod is not yet ready.
// Once all Pods have been checked and found to be in the Running state, the function closes the channel.
//
// Parameters:
//   - results: A channel of boolean values used to indicate the status of each Pod being checked.
//   - names: A slice of strings containing the names of the Pods whose status needs to be checked.
func CheckPodsStatus(results chan<- bool, names ...string) {
	count := 0
	for _, name := range names {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		pod, err := podsClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Checking status of pod failed: " + err.Error())
			results <- false
			return
		}
		switch pod.Status.Phase {
		case apiv1.PodPending:
			results <- false
		case apiv1.PodRunning:
			results <- true
			count++
		default:
			fmt.Println("Found an unrecognised phase state " + pod.Status.Phase)
			results <- false
		}
	}
	if count == len(names) {
		close(results)
	}

}

// AddLabelToRunningPod is a function that takes key, value, and uuid strings as input parameters and returns a boolean value.
// The function is used to add a label to a running Pod, mainly to indicate whether the Pod is idle or not.
// The function prepares the patch body in JSON format and patches the Pod with the specified UUID using the Kubernetes client.
//
// Parameters:
//   - key: A string containing the label key to be added to the Pod.
//   - value: A string containing the label value to be added to the Pod.
//   - uuid: A string containing the UUID of the Pod to be labeled.
//
// Returns:
//   - A boolean value indicating whether the patch was successfully applied to the Pod.
func AddLabelToRunningPod(key string, value string, uuid string) bool {

	// preparing the patch body in json format.
	labelPatch := fmt.Sprintf(`[{"op":"add","path":"/metadata/labels/%s","value":"%s" }]`, key, value)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// patching the pod with above created patch.
	_, err := podsClient.Patch(ctx, uuid, types.JSONPatchType, []byte(labelPatch), metav1.PatchOptions{})
	if err != nil {
		fmt.Println("Checking containercap pods error: ", err)
		return false
	}

	return true // returning true after successful patching the pod with label.
}

// CHANGED
// CheckForIdlePod searches for running pods with the same containercap and idle labels as provided 'pod'
// Not used
func CheckForIdlePod(pod *apiv1.Pod) (bool, string) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// List all pods that match the selector created from label key-value pairs where pod's containercap and idle values must be true.
	list, err := podsClient.List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("containercap=%s, %s=true", pod.ObjectMeta.Labels["containercap"], pod.ObjectMeta.Labels["idle"]),
	})
	if err != nil {
		fmt.Println("Checking idle pods error : ", err)
		return false, ""
	}

	// Iterate over the matched Pods list to find an existing Pod which has the same container Name
	for _, d := range list.Items {
		if d.Spec.Containers[0].Name == pod.Spec.Containers[0].Name && d.Name != pod.Name {
			return true, d.Name
		}
	}

	return false, ""
}

// CheckIdleContainerCapPods is a function that checks for all idle Pods that are labeled with "containercap".
// It returns a slice containing the names of all Pods that contain the "containercap" label and are idle (labeled "idle=true").
// The function uses the Kubernetes client to list all Pods with the specified label combination.
//
// Returns:
//   - A slice of strings containing the names of all idle Pods that are labeled with "containercap".
func CheckIdleContainerCapPods() []string {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	options := metav1.ListOptions{
		LabelSelector: "idle=true, containercap", // only select pods with this label combination
	}
	list, err := podsClient.List(ctx, options) // list all pods with above options
	if err != nil {
		fmt.Println("Checking idle containercap pods error:", err)
		return nil
	}

	// Initializing the pods slice with pre-allocated capacity for better performance
	pods := make([]string, 0, len(list.Items))

	for _, d := range list.Items {
		//only append pod name to the pod array if it has a non-empty "containercap" label
		if labelValue, exists := d.ObjectMeta.Labels["containercap"]; exists && labelValue != "" {
			pods = append(pods, d.ObjectMeta.Name)
		}
	}
	return pods
}

// Not used
func ReuseIdlePod(pod *apiv1.Pod) error {
	// Set the "idle" label to false so that the pod won't be reused by other scenarios.
	pod.ObjectMeta.Labels["idle"] = "false"
	_, err := podsClient.Update(context.Background(), pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update pod %s: %v", pod.Name, err)
	}
	// Wait for the pod to be ready before reusing it.
	err = wait.PollImmediate(5*time.Second, 60*time.Second, func() (bool, error) {
		updatedPod, err := podsClient.Get(context.Background(), pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("Failed to get updated pod %s: %v", pod.Name, err)
		}
		for _, c := range updatedPod.Status.ContainerStatuses {
			if !c.Ready {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("Pod %s is not ready after update: %v", pod.Name, err)
	}
	return nil
}

// CreateDeployment is a function that creates a new Kubernetes Deployment using the supplied Deployment object.
// Not used
func CreateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	// Create a new context with a timeout of 5 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel() // Clean up the context once the function returns.

	// Use the Kubernetes client to create the new deployment.
	result, err := deploymentsClient.Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// Return the new deployment object and any errors that were encountered.
	return result, nil
}

// PodExists is a function that takes a string containing the name of a Pod as input.
// It checks if a Pod with the specified name exists in the Kubernetes cluster using the Kubernetes client.
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
		return false, nil
	}
	return true, nil
}

// GetRunningPodByName is a helper function that takes a string containing the name of a running Pod as input.
// It retrieves the specifications of the specified Pod using the Kubernetes client and returns a PodSpec containing the relevant specifications.
//
// Parameters:
//   - name: A string containing the name of the running Pod whose specifications need to be retrieved.
//
// Returns:
//   - A PodSpec struct containing the specifications of the specified running Pod.
//   - An error if there were any issues encountered during the retrieval process.
func GetRunningPodByName(name string) (PodSpec, error) {

	var specs PodSpec

	// lists all current running Pods with the specified name
	pods, err := podsClient.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return specs, err
	}
	specs = SetPodSpec(pods)
	return specs, nil
}

// findIdlePodForAttacker is a function that takes a string containing the image name of an attacker container as input.
// It searches for an idle Pod with the same image in the same namespace as the attacker Pod by listing all Pods with the "idle=true" label.
// The function sorts the list of idle Pods by start time so that the oldest Pod is first.
// It then checks each Pod to see if it is in the Running phase and has not been scheduled by a ReplicaSet, and if it uses the same image as the attacker.
// If a matching Pod is found, the function returns a PodSpec containing the relevant specifications of the Pod.
//
// Parameters:
//   - image: A string containing the image name of the attacker container.
//
// Returns:
//   - A PodSpec struct containing the specifications of the matching idle Pod, if one is found.
//   - An error if there were any issues encountered during the search process.
func findIdlePodForAttacker(image string) (PodSpec, error) {
	var specs PodSpec
	// List all pods in the same namespace as the attacker pod
	listMutex.Lock() // Lock the mutex before accessing the list
	defer listMutex.Unlock()
	podList, err := podsClient.List(context.Background(), metav1.ListOptions{LabelSelector: "idle=true"})
	if err != nil {
		return specs, err
	}

	// Sort the pods by start time so that the oldest pod is first
	sort.SliceStable(podList.Items, func(i, j int) bool {
		return podList.Items[i].Status.StartTime.Before(podList.Items[j].Status.StartTime)
	})

	// Check each pod
	for _, pod := range podList.Items {
		// Check if the pod is in the "Running" phase and has not been scheduled by a ReplicaSet
		if pod.Status.Phase == apiv1.PodRunning && metav1.GetControllerOf(&pod) == nil {
			// Check if the pod uses the same image as the attacker
			for _, container := range pod.Spec.Containers {
				if container.Image == image {
					specs = SetPodSpec(&pod)
					return specs, nil
				}
			}
		}
	}
	return specs, nil
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
	// Create a new context with a timeout of 30 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel() // Clean up the context once the function returns.

	// Copy the file from the specified Pod and container to the local filesystem.

	// Construct the kubectl cp command
	cmd := exec.CommandContext(ctx, "kubectl", "cp", fmt.Sprintf("%s/%s:%s", apiv1.NamespaceDefault, podName, sourcePath), destPath, "-c", containerName)

	// Set the environment variables if needed (e.g., KUBECONFIG)
	// cmd.Env = append(os.Environ(), "KUBECONFIG=/path/to/kubeconfig")

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: %v\n", err)
		log.Printf("Output: %s\n", output)
		return err
	}

	// Delete the file from the Pod if keepFile is set to false
	if !keepFile {
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
	// Create a new context with a timeout of 30 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel() // Clean up the context once the function returns.

	// Copy the file from the specified Pod and container to the local filesystem.

	// Construct the kubectl cp command
	cmd := exec.CommandContext(ctx, "kubectl", "cp", sourcePath, fmt.Sprintf("%s/%s:%s", apiv1.NamespaceDefault, podName, destPath), "-c", containerName)

	// Set the environment variables if needed (e.g., KUBECONFIG)
	// cmd.Env = append(os.Environ(), "KUBECONFIG=/path/to/kubeconfig")

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: %v\n", err)
		log.Printf("Output: %s\n", output)
		return err
	}

	log.Println("File uploaded successfully")

	return nil
}
