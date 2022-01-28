// Package kubeapi is our main interface on top of the Kubernetes API
// Currently, we directly control Pods. Alternatively we may opt for the Kubernetes Job abstraction if it provides enough low-level access today
package kubeapi

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

var kubeConfig *rest.Config
var kubeClient kubernetes.Clientset
var podsClient v1.PodInterface

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
}

// CreatePod invokes the kubernetes API to create a pod
func CreatePod(pod *apiv1.Pod) {
	// Create Deployment
	fmt.Println("Creating pod...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	result, err := podsClient.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
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

// ListPod wraps around the kube api pod listing function
// The listoptions are part of kubernetes meta
func ListPod() {
	// List Deployments
	fmt.Printf("Listing pods in namespace %q:\n", apiv1.NamespaceDefault)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	list, err := podsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for k, d := range list.Items {
		fmt.Printf(" %d %+v \n", k, d.Status.ContainerStatuses)
	}
}

// DeletePod wraps around the kube api pod deletion call
// the deleteoptions are part of kubernetes meta
func DeletePod(name string) {
	// Delete Deployment
	fmt.Println("Deleting pod...")
	deletePolicy := metav1.DeletePropagationForeground
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := podsClient.Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted pod")

}

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

// func int32Ptr(i int32) *int32 { return &i }
// func modePtr(s apiv1.MountPropagationMode) *apiv1.MountPropagationMode { return &s }
