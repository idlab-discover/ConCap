package scenario

import (
	"fmt"
	"math/rand"
	"time"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	attackerPodCount   int = 0
	targetPodCount     int = 0
	supportPodCount    int = 0
	processingPodCount int = 0
	DeploymentCount    int = 0
	totalPods          int = 0
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Helperfunction to decrease the total Pod count by one
func MinusTotalPodsCount() {
	totalPods--
}

func MinusTargetPodCount() {
	targetPodCount--
	totalPods--
}
func MinusSupportPodCount() {
	supportPodCount--
	totalPods--
}
func MinusAttackerPodCount() {
	attackerPodCount--
	totalPods--
}

// Checks the amount of active ContainerCap pods.
// Hardcoded maximum of X active pods. In order to accomodate for scenarios, can be extended, but if the amount of pods
// for a new scenario would be more than the hardcoded value, the scenario cannot be added.
func CheckAmountOfPods() bool {
	if totalPods <= 50 {
		return true
	} else {
		for _, element := range kubeapi.CheckIdleContainerCapPods() {
			kubeapi.DeletePod(element)
			MinusTotalPodsCount()
		}
		if totalPods <= 50 {
			return true
		}
	}
	return false
}

// ScenarioPod takes a Scenario specification and turns it into a pod
// In the current implementation one pod = one experiment and one experiment is one invocation of an attack tool against one target
// All required metadata and containers are part of this one pod
// This trick also allows running tools on 127.0.0.1 because the containers in the pod share the same network
// The redesign will decouple this for flexiblity and resource efficiency reasons
func ScenarioPod(scn *Scenario, captureDir string) *apiv1.Pod {
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
				{Name: "containercap"},
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
					Command: []string{"tcpdump", "-i", scn.CaptureEngine.Interface, "-n", "-w", captureDir + "/" + scn.UUID.String() + ".pcap"},
					Lifecycle: &apiv1.Lifecycle{
						PreStop: &apiv1.LifecycleHandler{
							Exec: &apiv1.ExecAction{
								// This trick allows a clean exit of tcpdump, without the pcaps are often corrupted or not saved at all
								Command: []string{"/bin/sh", "-c", "kill -15 $(ps aux | grep \"[t]cpdump\" | tr -s \" \" | cut -d \" \" -f 2)"},
							},
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "nfs-volume",
							MountPath: "/storage/nfs/L/kube",
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

// AttackPod takes a Scenario specification and makes a pod with a sole attack-container.
func AttackPod(scn *Scenario, captureDir string) *apiv1.Pod {

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scn.UUID.String() + "-attacker-" + fmt.Sprint(attackerPodCount),
			Namespace: apiv1.NamespaceAll,
			Labels: map[string]string{
				"containercap": "attacker-pod",
				"category":     string(scn.Attacker.Category),
				"scenarioType": string(scn.ScenarioType),
				"idle":         "false",
			},
		},
		Spec: apiv1.PodSpec{
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{Name: "idlab-gitlab"},
			},
			Containers: []apiv1.Container{
				{
					Name:  scn.Attacker.Name,
					Image: scn.Attacker.Image,

					Stdin: true,
					TTY:   true,
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "nfs-volume",
							MountPath: "/Containercap",
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
	attackerPodCount++
	totalPods++

	return pod
}

// TargetPod takes a Scenario specification and makes a pod with a sole target-container.
func TargetPod(scn *Scenario, captureDir string) *apiv1.Pod {

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scn.UUID.String() + "-target-" + fmt.Sprint(targetPodCount),
			Namespace: apiv1.NamespaceAll,
			Labels: map[string]string{
				"containercap": "target-pod",
				"category":     string(scn.Target.Category),
				"scenarioType": string(scn.ScenarioType),
				"idle":         "false",
			},
		},

		Spec: apiv1.PodSpec{
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{Name: "idlab-gitlab"},
			},
			RestartPolicy: "Never",
			Containers: []apiv1.Container{
				{
					Name:  scn.Target.Name,
					Image: scn.Target.Image,
					Stdin: true,
					TTY:   true,
					SecurityContext: &apiv1.SecurityContext{
						Privileged: func() *bool { b := true; return &b }(),
					},

					Ports: []apiv1.ContainerPort{
						{
							Name:          RandStringRunes(8),
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: scn.Target.Ports[0],
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "nfs-volume",
							MountPath: "/Containercap",
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
	targetPodCount++
	totalPods++
	return pod
}

// SupportPods takes a Scenario specification and makes pods with a sole support-container.
func SupportPods(scn *Scenario, captureDir string, podIP string) []*apiv1.Pod {
	pods := make([]*apiv1.Pod, len(scn.Support))
	for index, support := range scn.Support {
		pod := &apiv1.Pod{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      scn.UUID.String() + "-support-" + fmt.Sprint(supportPodCount),
				Namespace: apiv1.NamespaceDefault,
				Labels: map[string]string{
					"containercap": "support-pod",
					"category":     string(support.Category),
					"scenarioType": string(scn.ScenarioType),
					"idle":         "false",
				},
			},
			Spec: apiv1.PodSpec{
				ImagePullSecrets: []apiv1.LocalObjectReference{
					{Name: "idlab-gitlab"}},
				Containers: []apiv1.Container{
					{
						Name:  support.Name,
						Image: support.Image,
						Ports: []apiv1.ContainerPort{
							{
								Name:          RandStringRunes(8),
								Protocol:      apiv1.ProtocolTCP,
								ContainerPort: support.Ports[0],
							},
						},
						Stdin: true,
						TTY:   true,
						SecurityContext: &apiv1.SecurityContext{
							Privileged: func() *bool { b := true; return &b }(),
						},
						Env: []apiv1.EnvVar{
							{
								Name:  "TARGETPOD_IP",
								Value: podIP,
							},
						},

						VolumeMounts: []apiv1.VolumeMount{
							{
								Name:      "nfs-volume",
								MountPath: "/Containercap",
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
		supportPodCount++
		totalPods++
		pods[index] = pod
	}
	return pods
}

// This function takes in a Scenario and a string representing
// the directory where captured data will be stored. It returns a pointer to an
// appsv1.Deployment object.
func AttackDeployment(scn *Scenario, captureDir string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: scn.UUID.String() + "-deployment-" + fmt.Sprint(DeploymentCount),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2), // Set the number of replicas to 2. This will need to be changed in the future.
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"containercap": "attack-deployment",
					"category":     string(scn.Attacker.Category), // Assign the attacker category as a label.
					"scenarioType": string(scn.ScenarioType),
					"idle":         "false", // Set the idle flag to false.
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: apiv1.NamespaceAll,
					Labels: map[string]string{
						"containercap": "attacker-pod",
						"category":     string(scn.Attacker.Category),
						"scenarioType": string(scn.ScenarioType),
						"idle":         "false",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  scn.Attacker.Name,
							Image: scn.Attacker.Image,

							Stdin: true,
							TTY:   true,
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
			},
		},
	}
	DeploymentCount++
	for i := 0; i < 2; i++ { // Increase the count of attacker pods and total pods.
		attackerPodCount++
		totalPods++
	}
	return deployment // Return the created deployment object.
}

// FlowProcessPod is the api pod constructor for feature processing pods (currently iscxflowmeter & cisco joy). We will include more tools such as argus and other netflow tools
func FlowProcessPod(name string) *apiv1.Pod {
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "processing-pod",
				"idle":         "false",
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
							MountPath: "/mnt/ContainerCap",
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
	processingPodCount++
	totalPods++
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

func int32Ptr(i int32) *int32 { return &i }
