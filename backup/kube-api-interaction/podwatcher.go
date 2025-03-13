package kubeapi

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
	podsClient  v1.PodInterface
	eventsChans map[string]chan *apiv1.Pod // Map of channels to send events to, dynamically changing pods we are watching, use mutex to protect
	mu          sync.Mutex                 // Mutex to protect concurrent access eventsChans
}

func NewPodWatcher(podsClient v1.PodInterface) PodWatcher {
	return PodWatcher{
		podsClient:  podsClient,
		eventsChans: make(map[string]chan *apiv1.Pod),
	}
}

func (pw *PodWatcher) Start(ctx context.Context) {
	go func() {
		if err := pw.WatchPods(ctx); err != nil {
			log.Fatalf("Error watching pods: %v", err)
		}
	}()
}

// WaitForPodRunning blocks until the pod is in the running phase.
func (pw *PodWatcher) WaitForPodRunning(ctx context.Context, podName string) (*apiv1.Pod, error) {
	return pw.WaitForPodPhase(ctx, podName, apiv1.PodRunning)
}

// WaitForPodPhase waits until the pod reaches the desired phase.
// First it adds the pod with a corresponding channel to the eventsChans map for PodWatcher.
// Then it waits for updates from the PodWatcher for the desired phase to be reached.
func (pw *PodWatcher) WaitForPodPhase(ctx context.Context, podName string, desiredPhase apiv1.PodPhase) (*apiv1.Pod, error) {
	pw.mu.Lock()
	if _, exists := pw.eventsChans[podName]; exists {
		fmt.Printf("Already watching pod %s", podName)
	} else {
		pw.eventsChans[podName] = make(chan *apiv1.Pod)
	}
	eventsChan := pw.eventsChans[podName]
	pw.mu.Unlock()

	defer func() {
		pw.mu.Lock()
		close(eventsChan)
		delete(pw.eventsChans, podName)
		pw.mu.Unlock()
	}()

	for {
		select {
		case pod := <-eventsChan:
			if pod.Status.Phase == desiredPhase {
				// Pod has reached desired phase and stop watching the pod
				return pod, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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
			pw.mu.Lock()
			if eventsChan, exists := pw.eventsChans[pod.Name]; exists {
				// Send pod event to the channel if we want to receive updates for this pod
				eventsChan <- pod
			}
			pw.mu.Unlock()
		}
	}
}
