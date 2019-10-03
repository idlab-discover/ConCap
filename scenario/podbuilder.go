package scenario

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var Local bool

func init() {
	rand.Seed(time.Now().UnixNano())
}

func ScenarioPod(scn *Scenario) *apiv1.Pod {
	var path string
	if Local {
		path = "/hosthome/dhoogla"
		fmt.Println("Storing results in ", path+"/PhD/containercap-captures")
	} else {
		path = os.Getenv("HOME")
		fmt.Println("Storing results in ", path+"/PhD/containercap-captures")
	}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scn.UUID.String(),
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "default-pod",
			},
		},
		Spec: apiv1.PodSpec{
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{Name: "idlab-gitlab"},
			},
			Containers: []apiv1.Container{
				{
					Name:  scn.Target.Name,
					Image: scn.Target.Image,
					Ports: []apiv1.ContainerPort{
						{
							Name:          RandStringRunes(8),
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: scn.Target.Ports[0],
						},
					},
				},
				{
					Name:  scn.Attacker.Name,
					Image: scn.Attacker.Image,
					Stdin: true,
					TTY:   true,
				},
				{
					Name:    scn.CaptureEngine.Name,
					Image:   scn.CaptureEngine.Image,
					Stdin:   true,
					TTY:     true,
					Command: []string{"tcpdump", "-i", scn.CaptureEngine.Interface, "-n", "-w", "/tmp/containercap-captures/" + scn.UUID.String() + ".pcap"},
					Lifecycle: &apiv1.Lifecycle{
						PreStop: &apiv1.Handler{
							Exec: &apiv1.ExecAction{
								Command: []string{"/bin/bash", "-c", "kill -15 $(ps aux | grep \"[t]cpdump\" | tr -s \" \" | cut -d \" \" -f 1)"},
							},
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "nfs-captures",
							MountPath: "/tmp/containercap-captures",
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "nfs-captures",
					VolumeSource: apiv1.VolumeSource{
						PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "pvc-nfs",
							ReadOnly:  false,
						},
					},
				},
			},
		},
	}

	return pod
}

func FlowProcessPod(name string) *apiv1.Pod {
	var path string
	if Local {
		path = "/hosthome/dhoogla"
	} else {
		path = os.Getenv("HOME")
	}
	fmt.Println(path)
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "processing-pod",
			},
		},
		Spec: apiv1.PodSpec{
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{Name: "idlab-gitlab"},
			},
			Containers: []apiv1.Container{
				{
					Name:  name,
					Image: "gitlab.ilabt.imec.be:4567/lpdhooge/containercap-imagery/" + name + ":latest",
					Stdin: true,
					TTY:   true,
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "nfs-captures",
							MountPath: "/tmp/containercap-captures",
						},
						{
							Name:      "nfs-transformed",
							MountPath: "/tmp/containercap-transformed",
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "nfs-captures",
					VolumeSource: apiv1.VolumeSource{
						PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "pvc-nfs",
							ReadOnly:  false,
						},
					},
				},
				{
					Name: "nfs-transformed",
					VolumeSource: apiv1.VolumeSource{
						PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "pvc-nfs",
							ReadOnly:  false,
						},
					},
				},
			},
		},
	}
	return pod
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
