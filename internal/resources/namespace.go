package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileNamespace creates the target namespace in the MCP if it doesn't exist.
func ReconcileNamespace(ctx context.Context, c client.Client, ns string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	err := c.Get(ctx, client.ObjectKeyFromObject(namespace), namespace)
	if apierrors.IsNotFound(err) {
		return c.Create(ctx, namespace)
	}
	return err
}
