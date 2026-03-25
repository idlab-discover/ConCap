package kubernetes

import (
	"context"
	"fmt"
	"log"
	"sync"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
)

type PodWatcher struct {
	podsClient v1.PodInterface
	waiters    map[string]*podWaiter
	mu         sync.Mutex
}

type podWaiter struct {
	events chan *apiv1.Pod
	done   chan struct{}
}

func NewPodWatcher(podsClient v1.PodInterface) PodWatcher {
	return PodWatcher{
		podsClient: podsClient,
		waiters:    make(map[string]*podWaiter),
	}
}

func (pw *PodWatcher) Start(ctx context.Context) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		if err := pw.WatchPods(ctx); err != nil {
			errCh <- err
		}
	}()
	return errCh
}

// WaitForPodReady blocks until the pod is in the running phase and all the containers are ready.
//
// First it adds the pod with a corresponding channel to the eventsChans map for PodWatcher.
// Then it waits for updates from the PodWatcher for the desired phase to be reached.
func (pw *PodWatcher) WaitForPodReady(ctx context.Context, podName string) (*apiv1.Pod, error) {
	pw.mu.Lock()
	waiter, exists := pw.waiters[podName]
	if exists {
		pw.mu.Unlock()
		return nil, fmt.Errorf("pod %s is already being watched", podName)
	}
	waiter = &podWaiter{
		events: make(chan *apiv1.Pod, 1),
		done:   make(chan struct{}),
	}
	pw.waiters[podName] = waiter
	pw.mu.Unlock()

	defer func() {
		pw.mu.Lock()
		if current, ok := pw.waiters[podName]; ok && current == waiter {
			delete(pw.waiters, podName)
			close(waiter.done)
		}
		pw.mu.Unlock()
	}()

	for {
		select {
		case pod := <-waiter.events:
			if pod.Status.Phase == apiv1.PodRunning && isPodReady(pod) {
				// Pod has reached desired phase and stop watching the pod
				return pod, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// isPodReady checks if all the containers in the pod are ready.
func isPodReady(pod *apiv1.Pod) bool {
	for _, container := range pod.Status.ContainerStatuses {
		if !container.Ready {
			return false
		}
	}
	return true
}

// WatchPods watches all pods in the namespace and sends events to the corresponding channels.
func (pw *PodWatcher) WatchPods(ctx context.Context) error {
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		timeout := int64(60 * 60 * 2) // 2 hours
		return pw.podsClient.Watch(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	}
	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			// The context is done, so we stop the watcher and exit
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("Error: Watch channel is closed")
			}
			pod, ok := event.Object.(*apiv1.Pod)
			if !ok {
				log.Printf("Unexpected event object while watching pods: %T", event.Object)
				continue // should not happen, only listen to pod events
			}
			pw.notify(pod)
		}
	}
}

func (pw *PodWatcher) notify(pod *apiv1.Pod) {
	pw.mu.Lock()
	waiter, exists := pw.waiters[pod.Name]
	pw.mu.Unlock()
	if !exists {
		return
	}

	select {
	case <-waiter.done:
		return
	default:
	}

	select {
	case waiter.events <- pod:
		return
	default:
	}

	select {
	case <-waiter.events:
	default:
	}

	select {
	case waiter.events <- pod:
	case <-waiter.done:
	}
}
