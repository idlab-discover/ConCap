package kubeapi

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

func PodTemplateBuilder(scn *scenario.Scenario) *apiv1.Pod {
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-pod",
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "storage",
			},
		},
		Spec: apiv1.PodSpec{
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{Name: "idlab-gitlab"},
			},
			Containers: []apiv1.Container{
				{
					Name:  "web1",
					Image: "nginx:1.12",
					Ports: []apiv1.ContainerPort{
						{
							Name:          "http1",
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: 80,
						},
					},
				},
				{
					Name:  "nmap",
					Image: "gitlab.ilabt.imec.be:4567/lpdhooge/containercap-imagery/nmap-scanner:v1.0.1",
					Stdin: true,
					TTY:   true,
				},
				{
					Name:  "tcpdump",
					Image: "gitlab.ilabt.imec.be:4567/lpdhooge/containercap-imagery/tcpdump:v1.0.1",
					Stdin: true,
					TTY:   true,
					Lifecycle: &apiv1.Lifecycle{
						PreStop: &apiv1.Handler{
							Exec: &apiv1.ExecAction{
								Command: []string{"/bin/bash", "-c", "kill -15 $(ps e | grep \"[t]cpdump\" | cut -d' ' -f 1)"},
							},
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						// {
						// 	Name:      "pv-cap-store",
						// 	MountPath: "/var/pv-captures",
						// },
						{
							Name:      "hostpath-store",
							MountPath: "/var/h-captures",
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				// {
				// 	Name: "pv-cap-store",
				// 	VolumeSource: apiv1.VolumeSource{
				// 		PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
				// 			ClaimName: "containercap-pvc",
				// 			ReadOnly:  false,
				// 		},
				// 	},
				// },
				{
					Name: "hostpath-store",
					VolumeSource: apiv1.VolumeSource{
						HostPath: &apiv1.HostPathVolumeSource{
							Path: "/hosthome/dhoogla/Documents/PhD/pv-captures",
						},
					},
				},
			},
		},
	}

	return pod
}

func CreatePod(pod *apiv1.Pod) {
	// Create Deployment
	fmt.Println("Creating pod...")
	result, err := podsClient.Create(pod)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
	prompt()
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

	prompt()
}

func ListPod() {
	// List Deployments
	fmt.Printf("Listing pods in namespace %q:\n", apiv1.NamespaceDefault)
	list, err := podsClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, d := range list.Items {
		fmt.Printf(" * %s \n", d.Name)
	}
	prompt()
}

func DeletePod() {
	// Delete Deployment
	prompt()
	fmt.Println("Deleting pod...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := podsClient.Delete("demo-pod", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted pod")

}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}

func int32Ptr(i int32) *int32                                          { return &i }
func modePtr(s apiv1.MountPropagationMode) *apiv1.MountPropagationMode { return &s }
