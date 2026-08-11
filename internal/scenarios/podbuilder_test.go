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
