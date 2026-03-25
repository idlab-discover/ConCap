package kubernetes

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestDeletePodWaitsForPodToDisappear(t *testing.T) {
	clientset := kubefake.NewSimpleClientset()
	originalPodsClient := podsClient
	podsClient = clientset.CoreV1().Pods(apiv1.NamespaceDefault)
	defer func() {
		podsClient = originalPodsClient
	}()

	if _, err := podsClient.Create(context.Background(), &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-a"},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create test pod: %v", err)
	}

	var getCalls int32
	clientset.PrependReactor("get", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		if action.(clienttesting.GetAction).GetName() != "pod-a" {
			return false, nil, nil
		}

		call := atomic.AddInt32(&getCalls, 1)
		if call < 3 {
			return true, &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod-a"},
			}, nil
		}

		return true, nil, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "pod-a")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := DeletePod(ctx, "pod-a"); err != nil {
		t.Fatalf("DeletePod returned error: %v", err)
	}
	if got := atomic.LoadInt32(&getCalls); got < 3 {
		t.Fatalf("DeletePod returned before the pod disappeared, get calls = %d", got)
	}
}

func TestDeletePodTreatsMissingPodAsDeleted(t *testing.T) {
	clientset := kubefake.NewSimpleClientset()
	originalPodsClient := podsClient
	podsClient = clientset.CoreV1().Pods(apiv1.NamespaceDefault)
	defer func() {
		podsClient = originalPodsClient
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := DeletePod(ctx, "missing-pod"); err != nil {
		t.Fatalf("DeletePod returned error for missing pod: %v", err)
	}
}
