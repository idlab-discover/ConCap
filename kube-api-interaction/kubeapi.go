package kubeapi

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/k0kubun/pp"
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

func init() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	kubeConf, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
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

func CreatePod(pod *apiv1.Pod) {
	// Create Deployment
	fmt.Println("Creating pod...")
	result, err := podsClient.Create(pod)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
}

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
		result, getErr := podsClient.Get("demo-pod", metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get latest version of pod: %v", getErr))
		}

		result.Spec.Containers[0].Image = "nginx:1.13" // change nginx version
		_, updateErr := podsClient.Update(result)
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("Update failed: %v", retryErr))
	}
	fmt.Println("Updated pod...")
}

func ListPod() {
	// List Deployments
	fmt.Printf("Listing pods in namespace %q:\n", apiv1.NamespaceDefault)
	list, err := podsClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for k, d := range list.Items {
		pp.Printf(" %d %+v \n", k, d.Status.ContainerStatuses)
	}
}

func DeletePod(name string) {
	// Delete Deployment
	fmt.Println("Deleting pod...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := podsClient.Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted pod")

}

func WatchPod() {
	fmt.Println("Watching pods...")
	watch, err := podsClient.Watch(metav1.ListOptions{Watch: true})
	if err != nil {
		panic(err)
	}
	eventChan := watch.ResultChan()
	for event := range eventChan {
		fmt.Println(event)
	}
}

func CheckPodStatus(name string, results chan<- bool) {
	pod, err := podsClient.Get(name, metav1.GetOptions{})
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

func int32Ptr(i int32) *int32                                          { return &i }
func modePtr(s apiv1.MountPropagationMode) *apiv1.MountPropagationMode { return &s }
