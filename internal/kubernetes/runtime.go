package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// WorkloadNamespace contains every pod managed by Concap.
	WorkloadNamespace = "concap"
	// ImagePullSecretName is the preconfigured GHCR credential used by Concap pods.
	ImagePullSecretName = "ghcr-creds"
)

func validateRuntimePrerequisites(ctx context.Context, client kubernetes.Interface) error {
	if _, err := client.CoreV1().Namespaces().Get(ctx, WorkloadNamespace, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("required namespace %q does not exist", WorkloadNamespace)
		}
		return fmt.Errorf("get namespace %q: %w", WorkloadNamespace, err)
	}

	secret, err := client.CoreV1().Secrets(WorkloadNamespace).Get(ctx, ImagePullSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("required image pull secret %s/%s does not exist", WorkloadNamespace, ImagePullSecretName)
		}
		return fmt.Errorf("get image pull secret %s/%s: %w", WorkloadNamespace, ImagePullSecretName, err)
	}
	if secret.Type != apiv1.SecretTypeDockerConfigJson {
		return fmt.Errorf("image pull secret %s/%s has type %q, want %q", WorkloadNamespace, ImagePullSecretName, secret.Type, apiv1.SecretTypeDockerConfigJson)
	}
	dockerConfigJSON := secret.Data[apiv1.DockerConfigJsonKey]
	if len(dockerConfigJSON) == 0 {
		return fmt.Errorf("image pull secret %s/%s has no %s data", WorkloadNamespace, ImagePullSecretName, apiv1.DockerConfigJsonKey)
	}
	var dockerConfig struct {
		Auths map[string]struct {
			Auth          string `json:"auth"`
			Username      string `json:"username"`
			Password      string `json:"password"`
			IdentityToken string `json:"identitytoken"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(dockerConfigJSON, &dockerConfig); err != nil {
		return fmt.Errorf("image pull secret %s/%s contains invalid Docker config JSON: %w", WorkloadNamespace, ImagePullSecretName, err)
	}
	credentials, ok := dockerConfig.Auths["ghcr.io"]
	if !ok {
		return fmt.Errorf("image pull secret %s/%s has no ghcr.io credentials", WorkloadNamespace, ImagePullSecretName)
	}
	if credentials.Auth == "" && credentials.IdentityToken == "" && (credentials.Username == "" || credentials.Password == "") {
		return fmt.Errorf("image pull secret %s/%s has empty ghcr.io credentials", WorkloadNamespace, ImagePullSecretName)
	}

	return nil
}
