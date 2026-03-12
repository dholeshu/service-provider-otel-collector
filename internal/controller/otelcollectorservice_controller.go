/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "github.com/openmcp-project/service-provider-otel-collector/api/v1alpha1"
	"github.com/openmcp-project/service-provider-otel-collector/internal/resources"
	"github.com/openmcp-project/service-provider-otel-collector/pkg/spruntime"
)

const defaultRequeueInterval = 30 * time.Second

// OtelCollectorServiceReconciler reconciles a OtelCollectorService object
type OtelCollectorServiceReconciler struct {
	OnboardingCluster *clusters.Cluster
	PlatformCluster   *clusters.Cluster
	PodNamespace      string
}

// CreateOrUpdate is called on every add or update event
func (r *OtelCollectorServiceReconciler) CreateOrUpdate(ctx context.Context, svcobj *apiv1alpha1.OtelCollectorService, pc *apiv1alpha1.ProviderConfig, clusterCtx spruntime.ClusterContext) (ctrl.Result, error) {
	l := logf.FromContext(ctx)
	mcpClient := clusterCtx.MCPCluster.Client()

	spruntime.StatusProgressing(svcobj, "Reconciling", "Reconcile in progress")

	// Resolve namespace
	ns := resolveNamespace(svcobj, pc)

	// Ensure namespace exists
	if err := resources.ReconcileNamespace(ctx, mcpClient, ns); err != nil {
		l.Error(err, "failed to reconcile namespace", "namespace", ns)
		return ctrl.Result{}, err
	}

	// Sync image pull secrets from platform to MCP
	if len(pc.Spec.ImagePullSecrets) > 0 {
		if err := resources.ReconcileImagePullSecrets(ctx, mcpClient, ns,
			r.PlatformCluster.Client(), pc.Spec.ImagePullSecrets, r.PodNamespace); err != nil {
			l.Error(err, "failed to sync image pull secrets")
			return ctrl.Result{}, err
		}
	}

	// Check prerequisites
	prereq, err := resources.CheckPrerequisites(ctx, mcpClient, ns)
	if err != nil {
		l.Error(err, "failed to check prerequisites")
		return ctrl.Result{}, err
	}
	if !prereq.Ready {
		spruntime.StatusProgressing(svcobj, "WaitingForPrerequisites", prereq.Message)
		l.Info("prerequisites not met, requeueing", "message", prereq.Message)
		return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
	}

	// Resolve image, version, resources
	image := resolveImage(svcobj, pc)
	version := resolveVersion(svcobj, pc)
	res := resolveResources(svcobj, pc)

	// Reconcile Deployment
	if err := resources.ReconcileDeployment(ctx, mcpClient, ns, resources.DeploymentOpts{
		Image:            image,
		Version:          version,
		Resources:        res,
		ImagePullSecrets: pc.Spec.ImagePullSecrets,
		ConfigHash:       prereq.ConfigHash,
	}); err != nil {
		l.Error(err, "failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := resources.ReconcileService(ctx, mcpClient, ns); err != nil {
		l.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	spruntime.StatusReady(svcobj)
	return ctrl.Result{}, nil
}

// Delete is called on every delete event
func (r *OtelCollectorServiceReconciler) Delete(ctx context.Context, obj *apiv1alpha1.OtelCollectorService, pc *apiv1alpha1.ProviderConfig, clusterCtx spruntime.ClusterContext) (ctrl.Result, error) {
	l := logf.FromContext(ctx)
	spruntime.StatusTerminating(obj)

	ns := resolveNamespace(obj, pc)

	if err := resources.DeleteManagedResources(ctx, clusterCtx.MCPCluster.Client(), ns); err != nil {
		l.Error(err, "failed to delete managed resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func resolveNamespace(obj *apiv1alpha1.OtelCollectorService, pc *apiv1alpha1.ProviderConfig) string {
	if obj.Spec.Namespace != nil {
		return *obj.Spec.Namespace
	}
	if pc.Spec.DefaultNamespace != nil {
		return *pc.Spec.DefaultNamespace
	}
	return "observability"
}

func resolveImage(obj *apiv1alpha1.OtelCollectorService, pc *apiv1alpha1.ProviderConfig) string {
	if obj.Spec.CollectorImage != nil {
		return *obj.Spec.CollectorImage
	}
	if pc.Spec.DefaultImage != nil {
		return *pc.Spec.DefaultImage
	}
	return "otel/opentelemetry-collector-contrib"
}

func resolveVersion(obj *apiv1alpha1.OtelCollectorService, pc *apiv1alpha1.ProviderConfig) string {
	if obj.Spec.CollectorVersion != nil {
		return *obj.Spec.CollectorVersion
	}
	if pc.Spec.DefaultVersion != nil {
		return *pc.Spec.DefaultVersion
	}
	return "0.146.1"
}

func resolveResources(obj *apiv1alpha1.OtelCollectorService, pc *apiv1alpha1.ProviderConfig) *corev1.ResourceRequirements {
	if obj.Spec.Resources != nil {
		return obj.Spec.Resources
	}
	return pc.Spec.DefaultResources
}
