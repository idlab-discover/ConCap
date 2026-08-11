package kubernetes

import (
	"context"
	"strings"
	"testing"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestValidateRuntimePrerequisites(t *testing.T) {
	tests := []struct {
		name      string
		objects   []runtime.Object
		wantError string
	}{
		{
			name: "valid namespace and pull secret",
			objects: []runtime.Object{
				&apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: WorkloadNamespace}},
				&apiv1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: ImagePullSecretName, Namespace: WorkloadNamespace},
					Type:       apiv1.SecretTypeDockerConfigJson,
					Data:       map[string][]byte{apiv1.DockerConfigJsonKey: []byte(`{"auths":{"ghcr.io":{"auth":"dXNlcjpwYXNz"}}}`)},
				},
			},
		},
		{
			name:      "missing namespace",
			wantError: `required namespace "concap" does not exist`,
		},
		{
			name: "missing pull secret",
			objects: []runtime.Object{
				&apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: WorkloadNamespace}},
			},
			wantError: "required image pull secret concap/ghcr-creds does not exist",
		},
		{
			name: "wrong pull secret type",
			objects: []runtime.Object{
				&apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: WorkloadNamespace}},
				&apiv1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: ImagePullSecretName, Namespace: WorkloadNamespace},
					Type:       apiv1.SecretTypeOpaque,
					Data:       map[string][]byte{apiv1.DockerConfigJsonKey: []byte(`{"auths":{}}`)},
				},
			},
			wantError: `has type "Opaque"`,
		},
		{
			name: "missing pull secret data",
			objects: []runtime.Object{
				&apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: WorkloadNamespace}},
				&apiv1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: ImagePullSecretName, Namespace: WorkloadNamespace},
					Type:       apiv1.SecretTypeDockerConfigJson,
				},
			},
			wantError: "has no .dockerconfigjson data",
		},
		{
			name: "missing ghcr credentials",
			objects: []runtime.Object{
				&apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: WorkloadNamespace}},
				&apiv1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: ImagePullSecretName, Namespace: WorkloadNamespace},
					Type:       apiv1.SecretTypeDockerConfigJson,
					Data:       map[string][]byte{apiv1.DockerConfigJsonKey: []byte(`{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}}`)},
				},
			},
			wantError: "has no ghcr.io credentials",
		},
		{
			name: "empty ghcr credentials",
			objects: []runtime.Object{
				&apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: WorkloadNamespace}},
				&apiv1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: ImagePullSecretName, Namespace: WorkloadNamespace},
					Type:       apiv1.SecretTypeDockerConfigJson,
					Data:       map[string][]byte{apiv1.DockerConfigJsonKey: []byte(`{"auths":{"ghcr.io":{}}}`)},
				},
			},
			wantError: "has empty ghcr.io credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := kubefake.NewSimpleClientset(tt.objects...)
			err := validateRuntimePrerequisites(context.Background(), client)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateRuntimePrerequisites returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateRuntimePrerequisites error = %v, want substring %q", err, tt.wantError)
			}
		})
	}
}
