package scenario

import (
	"math/rand"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Scenario) AttackPod() *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cleanPodName(s.Name + "-attacker-" + s.Attacker.Name),
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "attacker-pod",
				"category":     string(s.Attacker.Category),
				"scenarioType": string(s.ScenarioType),
			},
		},
		Spec: apiv1.PodSpec{
			// ImagePullSecrets: []apiv1.LocalObjectReference{
			// 	{Name: "idlab-gitlab"},
			// },
			Containers: []apiv1.Container{
				{
					Name:    cleanPodName(s.Attacker.Name),
					Image:   s.Attacker.Image,
					Command: []string{"tail", "-f", "/dev/null"}, // Command to keep the container running
					Stdin:   true,
					TTY:     true,
				},
			},
		},
	}
}

func (s *Scenario) TargetPod() *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cleanPodName(s.Name + "-target-" + s.Target.Name),
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "target-pod",
				"category":     string(s.Target.Category),
				"scenarioType": string(s.ScenarioType),
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
					Ports: []apiv1.ContainerPort{
						{
							Name:          RandStringRunes(8),
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: s.Target.Ports[0],
						},
					},
				},
				{
					Name:  "tcpdump",
					Image: "corfr/tcpdump",
					// When pods are deployed the actual tcpdump command will be started with correct filter including the IP addresses.
					Command: []string{"tail", "-f", "/dev/null"}, // Command to keep the container running
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "node-storage",
							MountPath: "/data",
						},
					},
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

func ProcessingPodSpec(name string, image string) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "processing-pod",
			},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:    name,
					Image:   image,
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
