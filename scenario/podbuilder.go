package scenario

import (
	"math/rand"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ScenarioPod takes a Scenario specification and turns it into a pod
// In the current implementation one pod = one experiment and one experiment is one invocation of an attack tool against one target
// All required metadata and containers are part of this one pod
// This trick also allows running tools on 127.0.0.1 because the containers in the pod share the same network
// The redesign will decouple this for flexiblity and resource efficiency reasons
func ScenarioPod(scn *Scenario) *apiv1.Pod {
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
					// Stdin: true,
					// TTY:   true,
				},
				{
					Name:  scn.Attacker.Name,
					Image: scn.Attacker.Image,
					Stdin: true,
					TTY:   true,
				},
				{
					Name:  scn.CaptureEngine.Name,
					Image: scn.CaptureEngine.Image,
					// The Stdin and TTY fields are important for debugging purposes, without them, you can't access the containers through kubectl
					Stdin:   true,
					TTY:     true,
					Command: []string{"tcpdump", "-i", scn.CaptureEngine.Interface, "-n", "-w", "/mnt/containercap-captures/" + scn.UUID.String() + ".pcap"},
					Lifecycle: &apiv1.Lifecycle{
						PreStop: &apiv1.Handler{
							Exec: &apiv1.ExecAction{
								// This trick allows a clean exit of tcpdump, without the pcaps are often corrupted or not saved at all
								Command: []string{"/bin/sh", "-c", "kill -15 $(ps aux | grep \"[t]cpdump\" | tr -s \" \" | cut -d \" \" -f 2)"},
							},
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "nfs-volume",
							MountPath: "/mnt",
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "nfs-volume",
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

// FlowProcessPod is the api pod constructor for feature processing pods (currently iscxflowmeter & cisco joy). We will include more tools such as argus and other netflow tools
func FlowProcessPod(name string) *apiv1.Pod {
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
							Name:      "nfs-volume",
							MountPath: "/mnt",
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "nfs-volume",
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

// RandStringRunes is a small helper function to create random n-length strings from the smallcap letterRunes
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
