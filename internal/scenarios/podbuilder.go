package scenarios

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Image names used in pod definitions
const (
	// ImageIproute2 is the container image used for network traffic shaping in init containers
	ImageIproute2 = "ghcr.io/idlab-discover/concap/iproute2:1.0.0"
	// ImageTcpdump is the container image used for capturing network traffic in target pods
	ImageTcpdump = "ghcr.io/idlab-discover/concap/tcpdump:1.0.0"
)

// Pod configuration constants
const (
	// InitContainerName is the name of the init container used for network configuration
	InitContainerName = "init-tc"
	// TcpdumpContainerName is the name of the container used for traffic capture
	TcpdumpContainerName = "tcpdump"
	// DataMountPath is the path where captured data is stored in the tcpdump container
	DataMountPath = "/data"
	// DataVolumeName is the name of the volume used for storing captured data
	DataVolumeName = "node-storage"
)

// Pod label constants
const (
	// LabelConcap is the label key used to identify the pod type
	LabelConcap = "concap"
	// LabelScenario is the label key used to identify the scenario
	LabelScenario = "scenario"
	// LabelAttackerPod is the label value for attacker pods
	LabelAttackerPod = "attacker-pod"
	// LabelTargetPod is the label value for target pods
	LabelTargetPod = "target-pod"
	// LabelProcessingPod is the label value for processing pods
	LabelProcessingPod = "processing-pod"

	// PodNamespaceName is the namespace where all pods are created
	PodNamespaceName = "default"
	// AttackerPodSuffix is the suffix used for attacker pod names
	AttackerPodSuffix = "-A"
	// TargetPodSuffix is the suffix used for target pod names
	TargetPodSuffix = "-T"
	// DefaultImagePullPolicy is the default image pull policy for all containers
	DefaultImagePullPolicy = "Always"
	// DefaultIdleCommand is the command used to keep containers running
	DefaultIdleCommand = "tail -f /dev/null"
	// RestartPolicyNever is the restart policy for target pods
	RestartPolicyNever = "Never"
	// CapabilityNetAdmin is the capability required for network configuration
	CapabilityNetAdmin = "NET_ADMIN"
)

// BuildAttackerPod creates a pod definition for an attacker
func BuildAttackerPod(name string, attacker Attacker, scenarioName string) *apiv1.Pod {
	resourceRequirements := apiv1.ResourceRequirements{
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse(attacker.CPURequest),
			apiv1.ResourceMemory: resource.MustParse(attacker.MemRequest),
		},
	}
	// Add CPU and Memory limits if they are provided
	if attacker.CPULimit != "" || attacker.MemLimit != "" {
		resourceRequirements.Limits = apiv1.ResourceList{}
		if attacker.CPULimit != "" {
			resourceRequirements.Limits[apiv1.ResourceCPU] = resource.MustParse(attacker.CPULimit)
		}
		if attacker.MemLimit != "" {
			resourceRequirements.Limits[apiv1.ResourceMemory] = resource.MustParse(attacker.MemLimit)
		}
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CleanPodName(scenarioName + AttackerPodSuffix),
			Namespace: PodNamespaceName,
			Labels: map[string]string{
				LabelConcap:   LabelAttackerPod,
				LabelScenario: scenarioName,
			},
		},
		Spec: apiv1.PodSpec{
			InitContainers: []apiv1.Container{
				{
					Name:  InitContainerName,
					Image: ImageIproute2,
					// Use the attacker's network configuration which already contains the merged settings
					// (global network settings + attacker-specific overrides)
					Command: []string{"sh", "-c", attacker.Network.GetTCCommand()},
					SecurityContext: &apiv1.SecurityContext{
						Capabilities: &apiv1.Capabilities{
							// NET_ADMIN capability is required to configure network settings
							Add: []apiv1.Capability{CapabilityNetAdmin},
						},
					},
				},
			},
			Containers: []apiv1.Container{
				{
					Name:            CleanPodName(attacker.Name),
					Image:           attacker.Image,
					ImagePullPolicy: DefaultImagePullPolicy,
					Command:         []string{"sh", "-c", DefaultIdleCommand}, // Command to keep the container running
					Stdin:           true,
					TTY:             true,
					Resources:       resourceRequirements,
				},
			},
		},
	}
}

// BuildTargetPod creates a pod definition for a target from a TargetConfig
func BuildTargetPod(targetConfig TargetConfig, scenarioName string, index int) *apiv1.Pod {
	resourceRequirements := apiv1.ResourceRequirements{
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse(targetConfig.CPURequest),
			apiv1.ResourceMemory: resource.MustParse(targetConfig.MemRequest),
		},
	}
	// Add CPU and Memory limits if they are provided
	if targetConfig.CPULimit != "" || targetConfig.MemLimit != "" {
		resourceRequirements.Limits = apiv1.ResourceList{}
		if targetConfig.CPULimit != "" {
			resourceRequirements.Limits[apiv1.ResourceCPU] = resource.MustParse(targetConfig.CPULimit)
		}
		if targetConfig.MemLimit != "" {
			resourceRequirements.Limits[apiv1.ResourceMemory] = resource.MustParse(targetConfig.MemLimit)
		}
	}

	// Create a unique suffix for multi-target scenarios
	suffix := fmt.Sprintf("-%d", index)
	podName := CleanPodName(scenarioName + TargetPodSuffix + suffix)
	containerName := CleanPodName(targetConfig.Name + suffix)

	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: PodNamespaceName,
			Labels: map[string]string{
				LabelConcap:   LabelTargetPod,
				LabelScenario: scenarioName,
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy: RestartPolicyNever,
			InitContainers: []apiv1.Container{
				{
					Name:  InitContainerName,
					Image: ImageIproute2,
					// Use the target's network configuration which already contains the merged settings
					// (global network settings + target-specific overrides)
					Command: []string{"sh", "-c", targetConfig.Network.GetTCCommand()},
					SecurityContext: &apiv1.SecurityContext{
						Capabilities: &apiv1.Capabilities{
							// NET_ADMIN capability is required to configure network settings
							Add: []apiv1.Capability{CapabilityNetAdmin},
						},
					},
				},
			},
			Containers: []apiv1.Container{
				{
					Name:            containerName,
					Image:           targetConfig.Image,
					ImagePullPolicy: DefaultImagePullPolicy,
					Stdin:           true,
					TTY:             true,
				},
				{
					Name:  "tcpdump",
					Image: ImageTcpdump,
					// When pods are deployed the actual tcpdump command will be started with correct filter including the IP addresses.
					Command: []string{"tail", "-f", "/dev/null"}, // Command to keep the container running
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "node-storage",
							MountPath: "/data",
						},
					},
					Resources: resourceRequirements,
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "node-storage",
					VolumeSource: apiv1.VolumeSource{
						EmptyDir: &apiv1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func ProcessingPodSpec(processingPod *ProcessingPod) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      processingPod.Name,
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"concap": "processing-pod",
			},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:            processingPod.Name,
					Image:           processingPod.ContainerImage,
					ImagePullPolicy: "Always",
					Command:         []string{"tail", "-f", "/dev/null"},
					Stdin:           true,
					TTY:             true,
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "node-storage-input",
							MountPath: "/data/input",
						},
						{
							Name:      "node-storage-output",
							MountPath: "/data/output",
						},
					},
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							apiv1.ResourceCPU:    resource.MustParse(processingPod.CPURequest),
							apiv1.ResourceMemory: resource.MustParse(processingPod.MemRequest),
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "node-storage-input",
					VolumeSource: apiv1.VolumeSource{
						EmptyDir: &apiv1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "node-storage-output",
					VolumeSource: apiv1.VolumeSource{
						EmptyDir: &apiv1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}
