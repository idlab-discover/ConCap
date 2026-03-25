package scenarios

import (
	"testing"

	apiv1 "k8s.io/api/core/v1"
)

func TestBuildAttackerPodUsesScenarioCleanupPolicy(t *testing.T) {
	pod := BuildAttackerPod("hydra", Attacker{
		Name:       "hydra",
		Image:      "example/hydra:latest",
		CPURequest: "100m",
		MemRequest: "128Mi",
	}, "scenario-a")

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
