package ccap

import (
	"math/rand"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Scenario) AttackPod() *apiv1.Pod {
	resourceRequirements := apiv1.ResourceRequirements{
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse(s.Attacker.CPURequest),
			apiv1.ResourceMemory: resource.MustParse(s.Attacker.MemRequest),
		},
	}
	// Add CPU and Memory limits if they are provided
	if s.Attacker.CPULimit != "" || s.Attacker.MemLimit != "" {
		resourceRequirements.Limits = apiv1.ResourceList{}
		if s.Attacker.CPULimit != "" {
			resourceRequirements.Limits[apiv1.ResourceCPU] = resource.MustParse(s.Attacker.CPULimit)
		}
		if s.Attacker.MemLimit != "" {
			resourceRequirements.Limits[apiv1.ResourceMemory] = resource.MustParse(s.Attacker.MemLimit)
		}
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cleanPodName(s.Name + "-A"),
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"concap":   "attacker-pod",
				"scenario": s.Name,
			},
		},
		Spec: apiv1.PodSpec{
			// ImagePullSecrets: []apiv1.LocalObjectReference{
			// 	{Name: "idlab-gitlab"},
			// },
			InitContainers: []apiv1.Container{
				{
					Name:    "init-tc",
					Image:   "ghcr.io/idlab-discover/concap/iproute2:1.0.0",
					Command: []string{"sh", "-c", s.Network.GetTCCommand()},
					SecurityContext: &apiv1.SecurityContext{
						Capabilities: &apiv1.Capabilities{
							Add: []apiv1.Capability{"NET_ADMIN"},
						},
					},
				},
			},
			Containers: []apiv1.Container{
				{
					Name:      cleanPodName(s.Attacker.Name),
					Image:     s.Attacker.Image,
					Command:   []string{"tail", "-f", "/dev/null"}, // Command to keep the container running
					Stdin:     true,
					TTY:       true,
					Resources: resourceRequirements,
				},
			},
		},
	}
}

func (s *Scenario) TargetPod() *apiv1.Pod {
	resourceRequirements := apiv1.ResourceRequirements{
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse(s.Target.CPURequest),
			apiv1.ResourceMemory: resource.MustParse(s.Target.MemRequest),
		},
	}
	// Add CPU and Memory limits if they are provided
	if s.Target.CPULimit != "" || s.Target.MemLimit != "" {
		resourceRequirements.Limits = apiv1.ResourceList{}
		if s.Target.CPULimit != "" {
			resourceRequirements.Limits[apiv1.ResourceCPU] = resource.MustParse(s.Target.CPULimit)
		}
		if s.Target.MemLimit != "" {
			resourceRequirements.Limits[apiv1.ResourceMemory] = resource.MustParse(s.Target.MemLimit)
		}
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cleanPodName(s.Name + "-T"),
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"concap":   "target-pod",
				"scenario": s.Name,
			},
		},

		Spec: apiv1.PodSpec{
			RestartPolicy: "Never",
			Containers: []apiv1.Container{
				{
					Name:  cleanPodName(s.Target.Name),
					Image: s.Target.Image,
					Stdin: true,
					TTY:   true,
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
					Name:    processingPod.Name,
					Image:   processingPod.ContainerImage,
					Command: []string{"tail", "-f", "/dev/null"},
					Stdin:   true,
					TTY:     true,
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

// RandStringRunes is a small helper function to create random n-length strings from the smallcap letterRunes
func RandStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
