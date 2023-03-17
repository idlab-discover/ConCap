// Package kubeapi is our main interface on top of the Kubernetes API
// Currently, we directly control Pods. Alternatively we may opt for the Kubernetes Job abstraction if it provides enough low-level access today
package kubeapi

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
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

// init for the kubeapi will get the kubeconfig and create a new clientset from which the a pod api is instantiated
func init() {
	// kubadm install
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	// local kind cluster
	// kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "kind-config-kind")
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

// CreatePod is a function that creates a new Kubernetes pod using the supplied Pod object.
func CreatePod(pod *apiv1.Pod) (*apiv1.Pod, error) {
	// Create a new context with a timeout of 5 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel() // Clean up the context once the function returns.

	// Use the Kubernetes client to create the new pod.
	result, err := podsClient.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// Return the new pod object and any errors that were encountered.
	return result, nil
}

// CreateRunningPod invokes the kubernetes API to create a pod.
// But, compared to CreatePod, will also wait until the pod is found in the Running state using CheckPodStatus.
// This way, certain information such as PodIP can also be retrieved.
// It also receives a boolean which, if set, can let the pod try obtain an idle running pod if available.
// It will return a PodSpec as well as another bool to notify if a pod has been reused or not.
func CreateRunningPod(pod *apiv1.Pod, reusable bool) (PodSpec, bool) {

	//Declaring variables
	var specs PodSpec = PodSpec{} // Stores the PodSpec of created/reused Pod
	reused := false
	var err error
	//ok, podUuid := CheckForIdlePod(pod) //Checking if idle pod is available

	if reusable {
		specs, err = findIdlePodForAttacker(pod.Spec.Containers[0].Image)
		if err != nil {
			fmt.Println("Error finding pod: " + err.Error())
		}
		fmt.Println(specs.Name + "	" + specs.PodIP)
	}

	if (specs != PodSpec{}) {

		AddLabelToRunningPod("idle", "false", specs.Uuid) //Adds label "idle : false" for this pod which has been reused.
		reused = true                                     //Sets reuse as true.

	} else {
		result, err := CreatePod(pod) //Creating new pod

		if err != nil {
			fmt.Errorf("Creation of pod failed: " + err.Error())
			return specs, false
		}

		podStates := make(chan bool, 64) //Used to get pod status
		fmt.Println("\n KubeAPI: Checking Pod Status Name:" + result.Name)

		go CheckPodsStatus(podStates, result.Name) //Concurrently goes on to check the pod status

		for msg := range podStates { //Iterating over channel of podStates
			if msg { //If there is a message...
				pod, err := podsClient.Get(context.Background(), result.Name, metav1.GetOptions{})
				if err != nil {
					fmt.Errorf("Checking pod status failed: " + err.Error())
					return specs, false
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
	return specs, reused
}

/*
	//If idle pod is present, and can be reused (attackerpod)
	if ok && reusable {
		list := CheckContainerCapPods()

		for _, result := range list.Items { //Check each pod in the list
			if result.Name == podUuid { //Compare pod Uuid with Uuid of idle pod

				//If pod matches, stores the pod informations in PodSpec variable
				specs = SetPodSpec(&result)

				AddLabelToRunningPod("idle", "false", result.Name) //Adds label "idle : false" for this pod which has been reused.
				reused = true                                      //Sets reuse as true.
			}
		}
	} else { //Else create a new pod

		result, err := CreatePod(pod) //Creating new pod

		if err != nil {
			fmt.Errorf("Creation of pod failed: " + err.Error())
			return specs, false
		}

		podStates := make(chan bool, 64) //Used to get pod status
		fmt.Println("\n KubeAPI: Checking Pod Status Name:" + result.Name)

		go CheckPodsStatus(podStates, result.Name) //Concurrently goes on to check the pod status

		for msg := range podStates { //Iterating over channel of podStates
			if msg { //If there is a message...
				pod, err := podsClient.Get(context.Background(), result.Name, metav1.GetOptions{})
				if err != nil {
					fmt.Errorf("Checking pod status failed: " + err.Error())
					return specs, false
				}
				result = pod //Set result as got from client.Get()
			} else {
				time.Sleep(10 * time.Second)               //Wait for 10 seconds
				go CheckPodsStatus(podStates, result.Name) //Repeatedly checks for pod state asynchronously
			}
		}

		//When everything works fine, create a pod specification
		specs = SetPodSpec(result)
	}*/

//Returns PodSpecification and whether pod was reused
//return specs, reused
//}

// SetPodSpec is a helper function that returns a structured object that contains some of the relevant specifications
// of the given Kubernetes Pod. It extracts the Pod's name, UUID, container image, category,
// scenario type, container capability, container name, and Pod IP from the provided Pod
// variable, and organizes them into a struct of type PodSpec.
// Parameters:
// 		- pod: A pointer to the Kubernetes Pod whose specifications are to be extracted.
// Returns:
//		- A PodSpec object containing the specifications of the given Pod.

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

func UpdatePodSpec(pod *apiv1.Pod, specs PodSpec) PodSpec {

	return PodSpec{}
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
	fmt.Println("Listing pods in namespace %q:\n", apiv1.NamespaceDefault)

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

// DeletePod function to delete a pod using its name
// TO DO: Will be used
func DeletePod(name string) {

	fmt.Println("Deleting pod...")

	// Attempt to delete the specified pod using the Kubernetes podsClient, passing in the name and delete options

	if err := podsClient.Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		panic(err)
	}

	fmt.Println("Deleted pod " + name)
}

// TO DO
// WatchPod gets the current event chain and prints the info
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

// CHANGED
func CheckPodsStatus(results chan<- bool, names ...string) {
	count := 0
	for _, name := range names {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		//	pod, err := podsClient.Get(ctx, names[name], metav1.GetOptions{})
		pod, err := podsClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			//panic(err)
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

// CheckPodStatus is a wrapper around get which uses the Phase part of the Status to signal to the lifecycle part in main if the pod is running and ready to accept an order
/*func CheckPodsStatus(results chan<- bool, names ...string) {

	// Create a waitgroup variable
	var wg sync.WaitGroup

	// Iterate through all pod names
	for _, name := range names {

		// Add 1 to the wait group count
		wg.Add(1)

		// Start a goroutine which takes in the name of the current pod instance
		go func(podName string) {

			// Release the wait group after the current goroutine completes its task
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			// Get the status of the pod for the given context using the podClient
			pod, err := podsClient.Get(ctx, podName, metav1.GetOptions{})

			// If an error occurs, set results <- false to into the channel that the pod has failed case
			if err != nil {
				fmt.Println("Checking status of pod failed: " + err.Error())
				results <- false
				return
			}

			// Based on the pod status's phase, write the result to a results channel
			switch pod.Status.Phase {
			case apiv1.PodPending:
				results <- false
			case apiv1.PodRunning:
				results <- true
			default:
				fmt.Println("Found an unrecognised phase state " + pod.Status.Phase)
				results <- false
			}
		}(name)
	}

	// This anonymous function launches another goroutine that waits for completion of all goroutines before closing the results channel.
	go func() {
		wg.Wait()
		close(results)
	}()
}
*/
// CheckPodStatus is a wrapper around get which uses the Phase part of the Status to signal to the lifecycle part in main if the pod is running and ready to accept an order
func CheckPodStatus(name string, results chan<- bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	pod, err := podsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	switch pod.Status.Phase {
	case "Pending":
		results <- false
	case "Running":
		results <- true
		close(results)
	default:
		fmt.Println("Found an unrecognised phase state", pod.Status.Phase)
		panic("help")
	}
}

// AddLabelToRunningPod function takes key, value and uuid as string input parameters and returns a boolean.
// This function is used to add a label to a running pod.
// Mainly used to indicate a pod is idle or not.
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

// func int32Ptr(i int32) *int32 { return &i }
// func modePtr(s apiv1.MountPropagationMode) *apiv1.MountPropagationMode { return &s }

// CHANGED
// Returns a list with the names of all pods containing the containercap label.
func CheckContainerCapPods() *apiv1.PodList {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Get a list of pods matching the "containercap" label.
	list, err := podsClient.List(ctx, metav1.ListOptions{
		LabelSelector: "containercap",
	})
	if err != nil {
		fmt.Println("Checking containercap pods error:", err)
		return nil
	}
	/*
		pods := make([]string, 0, len(list.Items))
		for _, d := range list.Items {
			// Check that the pod has a non-empty value for the "containercap" label before adding it.
			if labelValue, exists := d.ObjectMeta.Labels["containercap"]; exists && labelValue != "" {
				pods = append(pods, d.Name)
			}
		}*/
	//return pods
	return list
}

// CHANGED
// CheckIdleContainerCapPods checks for all idle pods that are labeled with "containercap".
// Returns a list with the names of all pods containing the containercap label (thus used for this program).
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
			pods = append(pods, d.Name)
		}
	}
	return pods
}

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

// PodExists checks if a pod exists with a given podName
func PodExists(podName string) (bool, error) {

	_, err := podsClient.Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetRunningPodByName Is a helper function called to get the specs of a runningPod
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

// This will be the bottleneck of the program because of the sorting
func findIdlePodForAttacker(image string) (PodSpec, error) {
	var specs PodSpec
	// List all pods in the same namespace as the attacker pod
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
