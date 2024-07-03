package kubeapi

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/appengine/log"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
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
			log.Errorf(ctx, "Error watching pods: %v", err)
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
		log.Warningf(ctx, "Already watching pod %s", podName)
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
	watch, err := pw.podsClient.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	defer watch.Stop()

	for {
		select {
		case event, ok := <-watch.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed unexpectedly")
			}
			pod, ok := event.Object.(*apiv1.Pod)
			if !ok {
				log.Warningf(ctx, "Unexpected event object while watching pods: %T", event.Object)
				continue // should not happen, only listen to pod events
			}
			pw.mu.Lock()
			if eventsChan, exists := pw.eventsChans[pod.Name]; exists {
				// Send pod event to the channel if we want to receive updates for this pod
				eventsChan <- pod
			}
			pw.mu.Unlock()
		case <-ctx.Done():
			return nil
		}
	}
}
