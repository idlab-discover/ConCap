package scenarios

import (
	"testing"

	kubeapi "github.com/idlab-discover/concap/internal/kubernetes"
	apiv1 "k8s.io/api/core/v1"
)

func assertPodRuntimeContract(t *testing.T, pod *apiv1.Pod, nodeRole string) {
	t.Helper()
	if got, want := pod.Namespace, kubeapi.WorkloadNamespace; got != want {
		t.Fatalf("Namespace = %q, want %q", got, want)
	}
	if got, want := pod.Spec.NodeSelector[NodeRoleLabel], nodeRole; got != want {
		t.Fatalf("NodeSelector[%q] = %q, want %q", NodeRoleLabel, got, want)
	}
	if got := pod.Spec.ImagePullSecrets; len(got) != 1 || got[0].Name != kubeapi.ImagePullSecretName {
		t.Fatalf("ImagePullSecrets = %#v, want %q", got, kubeapi.ImagePullSecretName)
	}
}

func TestBuildAttackerPodUsesScenarioCleanupPolicy(t *testing.T) {
	pod := BuildAttackerPod("hydra", Attacker{
		Name:       "hydra",
		Image:      "example/hydra:latest",
		CPURequest: "100m",
		MemRequest: "128Mi",
	}, "scenario-a")
	assertPodRuntimeContract(t, pod, NodeRoleAttacker)

	if got, want := pod.Spec.RestartPolicy, apiv1.RestartPolicy(RestartPolicyNever); got != want {
		t.Fatalf("RestartPolicy = %q, want %q", got, want)
	}
	if pod.Spec.TerminationGracePeriodSeconds == nil {
		t.Fatal("TerminationGracePeriodSeconds is nil")
	}
	if got, want := *pod.Spec.TerminationGracePeriodSeconds, PodTerminationGracePeriodSeconds; got != want {
		t.Fatalf("TerminationGracePeriodSeconds = %d, want %d", got, want)
	}
}

func TestBuildTargetPodUsesScenarioCleanupPolicy(t *testing.T) {
	pod := BuildTargetPod(TargetConfig{
		Name:       "target",
		Image:      "example/target:latest",
		CPURequest: "100m",
		MemRequest: "128Mi",
	}, "scenario-a", 0)
	assertPodRuntimeContract(t, pod, NodeRoleTarget)

	if got, want := pod.Spec.RestartPolicy, apiv1.RestartPolicy(RestartPolicyNever); got != want {
		t.Fatalf("RestartPolicy = %q, want %q", got, want)
	}
	if pod.Spec.TerminationGracePeriodSeconds == nil {
		t.Fatal("TerminationGracePeriodSeconds is nil")
	}
	if got, want := *pod.Spec.TerminationGracePeriodSeconds, PodTerminationGracePeriodSeconds; got != want {
		t.Fatalf("TerminationGracePeriodSeconds = %d, want %d", got, want)
	}
}

func TestBuildTargetPodIncludesCaptureNormalizationSidecars(t *testing.T) {
	pod := BuildTargetPod(TargetConfig{
		Name:       "target",
		Image:      "example/target:latest",
		CPURequest: "100m",
		MemRequest: "128Mi",
	}, "scenario-a", 0)

	containers := map[string]apiv1.Container{}
	for _, container := range pod.Spec.Containers {
		containers[container.Name] = container
	}

	tcpdump, ok := containers[TcpdumpContainerName]
	if !ok {
		t.Fatalf("target pod missing %q container", TcpdumpContainerName)
	}
	if got, want := tcpdump.Image, ImageTcpdump; got != want {
		t.Fatalf("tcpdump image = %q, want %q", got, want)
	}

	reordercap, ok := containers[ReordercapContainerName]
	if !ok {
		t.Fatalf("target pod missing %q container", ReordercapContainerName)
	}
	if got, want := reordercap.Image, ImageReordercap; got != want {
		t.Fatalf("reordercap image = %q, want %q", got, want)
	}

	for _, container := range []apiv1.Container{tcpdump, reordercap} {
		if len(container.VolumeMounts) != 1 {
			t.Fatalf("%s volume mounts = %#v, want one shared data mount", container.Name, container.VolumeMounts)
		}
		mount := container.VolumeMounts[0]
		if mount.Name != DataVolumeName || mount.MountPath != DataMountPath {
			t.Fatalf("%s data mount = %#v, want %s mounted at %s", container.Name, mount, DataVolumeName, DataMountPath)
		}
	}
}

func TestProcessingPodUsesConcapRuntimeContract(t *testing.T) {
	pod := ProcessingPodSpec(&ProcessingPod{
		Name:           "processor",
		ContainerImage: "example/processor:latest",
		CPURequest:     "100m",
		MemRequest:     "128Mi",
	})

	if got, want := pod.Namespace, kubeapi.WorkloadNamespace; got != want {
		t.Fatalf("Namespace = %q, want %q", got, want)
	}
	if got := pod.Spec.ImagePullSecrets; len(got) != 1 || got[0].Name != kubeapi.ImagePullSecretName {
		t.Fatalf("ImagePullSecrets = %#v, want %q", got, kubeapi.ImagePullSecretName)
	}
	if len(pod.Spec.NodeSelector) != 0 {
		t.Fatalf("processing pod NodeSelector = %#v, want scheduler-managed placement", pod.Spec.NodeSelector)
	}
}
