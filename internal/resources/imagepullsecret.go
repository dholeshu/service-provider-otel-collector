package resources

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileImagePullSecrets copies the specified image pull secrets from the platform cluster
// to the target namespace in the MCP.
func ReconcileImagePullSecrets(ctx context.Context, mcpClient client.Client, ns string, platformClient client.Client, secrets []corev1.LocalObjectReference, sourceNs string) error {
	for _, ref := range secrets {
		src := &corev1.Secret{}
		if err := platformClient.Get(ctx, client.ObjectKey{Namespace: sourceNs, Name: ref.Name}, src); err != nil {
			return fmt.Errorf("getting image pull secret %q from platform: %w", ref.Name, err)
		}

		dst := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ref.Name,
				Namespace: ns,
			},
		}
		err := mcpClient.Get(ctx, client.ObjectKeyFromObject(dst), dst)
		if apierrors.IsNotFound(err) {
			dst.Type = corev1.SecretTypeDockerConfigJson
			dst.Data = src.Data
			dst.Labels = map[string]string{
				ManagedByLabel: ManagedByValue,
			}
			return mcpClient.Create(ctx, dst)
		}
		if err != nil {
			return err
		}
		dst.Data = src.Data
		return mcpClient.Update(ctx, dst)
	}
	return nil
}
