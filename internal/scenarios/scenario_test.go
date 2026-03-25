package scenarios

import (
	"context"
	"errors"
	"testing"

	kubeexec "k8s.io/client-go/util/exec"
)

func TestExecuteScenarioUsesCleanupContextForDeletion(t *testing.T) {
	sentinel := errors.New("attack failed")
	scenario := &fakeScenario{
		executeAttackErr: sentinel,
	}

	err := ExecuteScenario(context.Background(), scenario, t.TempDir())
	if !errors.Is(err, sentinel) {
		t.Fatalf("ExecuteScenario error = %v, want %v", err, sentinel)
	}
	if !scenario.deleteCalled {
		t.Fatal("DeleteAllPods was not called")
	}
	if scenario.deleteCtxErr != nil {
		t.Fatalf("DeleteAllPods received canceled context: %v", scenario.deleteCtxErr)
	}
}

func TestExecuteScenarioPreservesPartialResultsOnTimeout(t *testing.T) {
	sentinel := kubeexec.CodeExitError{Err: errors.New("timeout"), Code: 124}
	scenario := &fakeScenario{
		executeAttackErr: sentinel,
	}

	err := ExecuteScenario(context.Background(), scenario, t.TempDir())
	if !errors.Is(err, sentinel) {
		t.Fatalf("ExecuteScenario error = %v, want timeout error %v", err, sentinel)
	}
	if !scenario.partialDownloadCalled {
		t.Fatal("DownloadPartialResults was not called for timeout")
	}
}

func TestExecuteScenarioDoesNotPreservePartialResultsForOtherFailures(t *testing.T) {
	sentinel := errors.New("attack failed")
	scenario := &fakeScenario{
		executeAttackErr: sentinel,
	}

	err := ExecuteScenario(context.Background(), scenario, t.TempDir())
	if !errors.Is(err, sentinel) {
		t.Fatalf("ExecuteScenario error = %v, want %v", err, sentinel)
	}
	if scenario.partialDownloadCalled {
		t.Fatal("DownloadPartialResults was called for non-timeout failure")
	}
}

type fakeScenario struct {
	deleteCalled          bool
	deleteCtxErr          error
	executeAttackErr      error
	partialDownloadCalled bool
}

func (s *fakeScenario) FromYAML(string) error {
	return nil
}

func (s *fakeScenario) DeployAllPods(context.Context) error {
	return nil
}

func (s *fakeScenario) StartTrafficCapture(context.Context) error {
	return nil
}

func (s *fakeScenario) ExecuteAttack(context.Context) error {
	return s.executeAttackErr
}

func (s *fakeScenario) DownloadResults(context.Context, string) error {
	return nil
}

func (s *fakeScenario) DownloadPartialResults(context.Context, string) error {
	s.partialDownloadCalled = true
	return nil
}

func (s *fakeScenario) ProcessResults(context.Context, string, []*ProcessingPod) error {
	return nil
}

func (s *fakeScenario) DeleteAllPods(ctx context.Context) error {
	s.deleteCalled = true
	s.deleteCtxErr = ctx.Err()
	return nil
}

func (s *fakeScenario) Execute(context.Context, string) error {
	return nil
}

func (s *fakeScenario) GetName() string {
	return "fake"
}
