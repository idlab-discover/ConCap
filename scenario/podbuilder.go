package scenario

import (
	"log"
	"math/rand"
	"os"
	"strings"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gopkg.in/yaml.v2"
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

const MaxRunningPods = 50

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
// This needs to be dynamically chosen in the feature.
// This was not tested yet
func ExceedingMaxRunningPods() bool {
	if totalPods <= MaxRunningPods {
		return false
	}
	return true
}

func DeleteIdlePods() {
	for _, element := range kubeapi.CheckIdleContainerCapPods() {
		err := kubeapi.DeletePod(element)
		if err != nil {
			MinusTotalPodsCount()
		} else {
			log.Println("Error deleting idle pod: " + err.Error())
		}
	}
}

// AttackPod takes a Scenario specification and makes a pod with a sole attack-container.
func AttackPod(scn *Scenario) *apiv1.Pod {

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ReplaceAll(strings.ReplaceAll(scn.Attacker.Image, "/", "-"), ":", "-") + "-attacker",
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "attacker-pod",
				"category":     string(scn.Attacker.Category),
				"scenarioType": string(scn.ScenarioType),
				"idle":         "false",
			},
		},
		Spec: apiv1.PodSpec{
			// ImagePullSecrets: []apiv1.LocalObjectReference{
			// 	{Name: "idlab-gitlab"},
			// },
			Containers: []apiv1.Container{
				{
					Name:    scn.Attacker.Name,
					Image:   scn.Attacker.Image,
					Command: []string{"tail", "-f", "/dev/null"}, // Command to keep the container running
					Stdin:   true,
					TTY:     true,
				},
			},
		},
	}
	attackerPodCount++
	totalPods++

	return pod
}

// TargetPod takes a Scenario specification and makes a pod with a sole target-container.
func TargetPod(scn *Scenario) *apiv1.Pod {

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scn.Name + "-target-" + scn.Target.Name,
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "target-pod",
				"category":     string(scn.Target.Category),
				"scenarioType": string(scn.ScenarioType),
				"idle":         "false",
			},
		},

		Spec: apiv1.PodSpec{
			SecurityContext: &apiv1.PodSecurityContext{
				FSGroup: int64Ptr(1000),
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
					Env: []apiv1.EnvVar{
						{
							Name:  "TARGETPOD_IP",
							Value: "0.0.0.0",
						},
					},
				},
				{
					Name:  "tcpdump",
					Image: "corfr/tcpdump",
					// TODO change command to only capture traffic on the target container
					Command: []string{"tcpdump", "-i", "any", "-w", "/data/dump.pcap", "!arp"},
					SecurityContext: &apiv1.SecurityContext{
						Privileged: func() *bool { b := true; return &b }(),
					},
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
	targetPodCount++
	totalPods++
	return pod
}

// LoadPodSpecFromYaml takes a path to a yaml file and returns a pointer to an apiv1.Pod object.
// Watchout podspec is has difference from kubectl yaml files..
func LoadPodSpecFromYaml(path string) (*apiv1.Pod, error) {
	// Read the yaml file
	podYAML, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the yaml file to a Pod object
	var podSpec apiv1.Pod
	if err := yaml.Unmarshal(podYAML, &podSpec); err != nil {
		return nil, err
	}
	return &podSpec, nil
}

func ProcessingPodSpec(name string, image string) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"containercap": "processing-pod",
				"idle":         "false",
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
							Name:      "node-storage-pcap",
							MountPath: "/data/pcap",
						},
						{
							Name:      "node-storage-flow",
							MountPath: "/data/flow",
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "node-storage-pcap",
					VolumeSource: apiv1.VolumeSource{
						EmptyDir: &apiv1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "node-storage-flow",
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

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
