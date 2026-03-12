package resources

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteManagedResources deletes the Deployment and Service managed by this service provider.
// It leaves the user-created ConfigMap and Secret intact.
func DeleteManagedResources(ctx context.Context, c client.Client, ns string) error {
	deployment := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: DeploymentName}, deployment); err == nil {
		if err := c.Delete(ctx, deployment); client.IgnoreNotFound(err) != nil {
			return err
		}
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	svc := &corev1.Service{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: ServiceName}, svc); err == nil {
		if err := c.Delete(ctx, svc); client.IgnoreNotFound(err) != nil {
			return err
		}
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
