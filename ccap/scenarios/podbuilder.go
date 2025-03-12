package ccap

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildAttackerPod creates a pod definition for an attacker
func BuildAttackerPod(name string, attacker Attacker, scenarioName string, network Network) *apiv1.Pod {
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
			Name:      CleanPodName(scenarioName + "-A"),
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"concap":   "attacker-pod",
				"scenario": scenarioName,
			},
		},
		Spec: apiv1.PodSpec{
			InitContainers: []apiv1.Container{
				{
					Name:    "init-tc",
					Image:   "ghcr.io/idlab-discover/concap/iproute2:1.0.0",
					Command: []string{"sh", "-c", network.GetTCCommand()},
					SecurityContext: &apiv1.SecurityContext{
						Capabilities: &apiv1.Capabilities{
							Add: []apiv1.Capability{"NET_ADMIN"},
						},
					},
				},
			},
			Containers: []apiv1.Container{
				{
					Name:            CleanPodName(attacker.Name),
					Image:           attacker.Image,
					ImagePullPolicy: "Always",
					Command:         []string{"tail", "-f", "/dev/null"}, // Command to keep the container running
					Stdin:           true,
					TTY:             true,
					Resources:       resourceRequirements,
				},
			},
		},
	}
}

// BuildTargetPod creates a pod definition for a target
func BuildTargetPod(name string, target Target, scenarioName string, network Network) *apiv1.Pod {
	resourceRequirements := apiv1.ResourceRequirements{
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse(target.CPURequest),
			apiv1.ResourceMemory: resource.MustParse(target.MemRequest),
		},
	}
	// Add CPU and Memory limits if they are provided
	if target.CPULimit != "" || target.MemLimit != "" {
		resourceRequirements.Limits = apiv1.ResourceList{}
		if target.CPULimit != "" {
			resourceRequirements.Limits[apiv1.ResourceCPU] = resource.MustParse(target.CPULimit)
		}
		if target.MemLimit != "" {
			resourceRequirements.Limits[apiv1.ResourceMemory] = resource.MustParse(target.MemLimit)
		}
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CleanPodName(scenarioName + "-T-" + target.Name),
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"concap":   "target-pod",
				"scenario": scenarioName,
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy: "Never",
			Containers: []apiv1.Container{
				{
					Name:            CleanPodName(target.Name),
					Image:           target.Image,
					ImagePullPolicy: "Always",
					Stdin:           true,
					TTY:             true,
				},
				{
					Name:  "tcpdump",
					Image: "ghcr.io/idlab-discover/concap/tcpdump:1.0.0",
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

// BuildTargetPodFromConfig creates a pod definition for a target from a TargetConfig
func BuildTargetPodFromConfig(targetConfig TargetConfig, scenarioName string, index int) *apiv1.Pod {
	// Convert TargetConfig to Target for reuse
	target := Target{
		Name:       targetConfig.Name,
		Image:      targetConfig.Image,
		Filter:     targetConfig.Filter,
		CPURequest: targetConfig.CPURequest,
		CPULimit:   targetConfig.CPULimit,
		MemRequest: targetConfig.MemRequest,
		MemLimit:   targetConfig.MemLimit,
	}

	// Use the target's network configuration if available, otherwise use an empty one
	network := targetConfig.Network

	// Create a unique suffix for multi-target scenarios
	suffix := fmt.Sprintf("-%d", index)

	pod := BuildTargetPod(target.Name+suffix, target, scenarioName, network)

	// Update the pod name to include the index for multi-target scenarios
	pod.ObjectMeta.Name = CleanPodName(scenarioName + "-T" + suffix)

	return pod
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
