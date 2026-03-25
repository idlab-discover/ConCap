package kubernetes

import (
	"context"
	"errors"
	"testing"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodWatcherWaitForPodReadyReturnsReadyPod(t *testing.T) {
	pw := NewPodWatcher(nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resultCh := make(chan *apiv1.Pod, 1)
	errCh := make(chan error, 1)
	go func() {
		pod, err := pw.WaitForPodReady(ctx, "pod-a")
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- pod
	}()

	waitForWaiter(t, &pw, "pod-a")

	pw.notify(testPod("pod-a", apiv1.PodPending, false))
	pw.notify(testPod("pod-a", apiv1.PodRunning, true))

	select {
	case err := <-errCh:
		t.Fatalf("WaitForPodReady returned error: %v", err)
	case pod := <-resultCh:
		if got := pod.Name; got != "pod-a" {
			t.Fatalf("returned pod name = %q, want %q", got, "pod-a")
		}
	case <-time.After(time.Second):
		t.Fatal("WaitForPodReady did not return a ready pod")
	}
}

func TestPodWatcherNotifyReplacesStaleBufferedEvent(t *testing.T) {
	pw := NewPodWatcher(nil)
	waiter := &podWaiter{
		events: make(chan *apiv1.Pod, 1),
		done:   make(chan struct{}),
	}

	pw.mu.Lock()
	pw.waiters["pod-a"] = waiter
	pw.mu.Unlock()

	stale := testPod("pod-a", apiv1.PodPending, false)
	latest := testPod("pod-a", apiv1.PodRunning, true)

	pw.notify(stale)
	pw.notify(latest)

	select {
	case pod := <-waiter.events:
		if pod.Status.Phase != apiv1.PodRunning || !isPodReady(pod) {
			t.Fatalf("received stale pod event: phase=%s ready=%v", pod.Status.Phase, isPodReady(pod))
		}
	case <-time.After(time.Second):
		t.Fatal("notify did not deliver the latest pod event")
	}
}

func TestPodWatcherWaitForPodReadyRemovesWaiterOnCancel(t *testing.T) {
	pw := NewPodWatcher(nil)
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := pw.WaitForPodReady(ctx, "pod-a")
		errCh <- err
	}()

	waitForWaiter(t, &pw, "pod-a")
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForPodReady error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitForPodReady did not return after cancellation")
	}

	waitForWaiterRemoval(t, &pw, "pod-a")
}

func TestPodWatcherWaitForPodReadyRejectsDuplicateWatchers(t *testing.T) {
	pw := NewPodWatcher(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := pw.WaitForPodReady(ctx, "pod-a")
		errCh <- err
	}()

	waitForWaiter(t, &pw, "pod-a")

	_, err := pw.WaitForPodReady(context.Background(), "pod-a")
	if err == nil {
		t.Fatal("WaitForPodReady returned nil error for duplicate watcher")
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("original WaitForPodReady did not return after cancellation")
	}
}

func testPod(name string, phase apiv1.PodPhase, ready bool) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: apiv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []apiv1.ContainerStatus{
				{Ready: ready},
			},
		},
	}
}

func waitForWaiter(t *testing.T, pw *PodWatcher, podName string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		pw.mu.Lock()
		_, exists := pw.waiters[podName]
		pw.mu.Unlock()
		if exists {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for waiter %q to register", podName)
}

func waitForWaiterRemoval(t *testing.T, pw *PodWatcher, podName string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		pw.mu.Lock()
		_, exists := pw.waiters[podName]
		pw.mu.Unlock()
		if !exists {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for waiter %q to be removed", podName)
}
