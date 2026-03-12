package e2e

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/openmcp-project/openmcp-testing/pkg/clusterutils"
	"github.com/openmcp-project/openmcp-testing/pkg/conditions"
	"github.com/openmcp-project/openmcp-testing/pkg/providers"
	"github.com/openmcp-project/openmcp-testing/pkg/resources"
)

const targetNamespace = "observability"

func TestServiceProvider(t *testing.T) {
	var onboardingList unstructured.UnstructuredList
	basicProviderTest := features.New("otel-collector provider test").
		Setup(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			if _, err := resources.CreateObjectsFromDir(ctx, c, "platform"); err != nil {
				t.Errorf("failed to create platform cluster objects: %v", err)
			}
			return ctx
		}).
		Setup(providers.CreateMCP("test-mcp")).
		Assess("verify service stays Progressing without prerequisites",
			func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
				// Retry creating onboarding objects to allow CRD discovery to refresh
				var objList *unstructured.UnstructuredList
				var lastErr error
				deadline := time.Now().Add(2 * time.Minute)
				for time.Now().Before(deadline) {
					onboardingConfig, err := clusterutils.OnboardingConfig()
					if err != nil {
						t.Error(err)
						return ctx
					}
					objList, lastErr = resources.CreateObjectsFromDir(ctx, onboardingConfig, "onboarding")
					if lastErr == nil {
						break
					}
					time.Sleep(5 * time.Second)
				}
				if lastErr != nil {
					t.Errorf("failed to create onboarding cluster objects: %v", lastErr)
					return ctx
				}
				// Service should be Progressing (waiting for ConfigMap and Secret)
				onboardingConfig, err := clusterutils.OnboardingConfig()
				if err != nil {
					t.Error(err)
					return ctx
				}
				for _, obj := range objList.Items {
					if err := wait.For(conditions.Match(&obj, onboardingConfig, "Ready", corev1.ConditionFalse),
						wait.WithTimeout(2*time.Minute)); err != nil {
						t.Error(err)
					}
				}
				objList.DeepCopyInto(&onboardingList)
				return ctx
			},
		).
		Assess("create prerequisites and verify Ready",
			func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
				// Get MCP config with observability namespace so fixtures are created there
				mcpConfig, err := clusterutils.ConfigByPrefix("mcp", targetNamespace)
				if err != nil {
					t.Error(err)
					return ctx
				}
				// Create the observability namespace in the MCP first
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: targetNamespace}}
				if err := mcpConfig.Client().Resources().Create(ctx, ns); err != nil {
					t.Errorf("failed to create namespace %s: %v", targetNamespace, err)
					return ctx
				}
				// Create ConfigMap and Secret in MCP
				if _, err := resources.CreateObjectsFromDir(ctx, mcpConfig, "mcp"); err != nil {
					t.Errorf("failed to create MCP prerequisites: %v", err)
					return ctx
				}
				// Wait for the service to become Ready
				onboardingConfig, err := clusterutils.OnboardingConfig()
				if err != nil {
					t.Error(err)
					return ctx
				}
				for _, obj := range onboardingList.Items {
					if err := wait.For(conditions.Match(&obj, onboardingConfig, "Ready", corev1.ConditionTrue),
						wait.WithTimeout(3*time.Minute)); err != nil {
						t.Error(err)
					}
				}
				return ctx
			},
		).
		Assess("verify Deployment and Service exist in MCP",
			func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
				mcpConfig, err := clusterutils.McpConfig()
				if err != nil {
					t.Error(err)
					return ctx
				}
				mcpClient, err := mcpConfig.NewClient()
				if err != nil {
					t.Error(err)
					return ctx
				}
				// Check Deployment
				var deployment appsv1.Deployment
				if err := mcpClient.Resources(targetNamespace).Get(ctx, "otel-collector", targetNamespace, &deployment); err != nil {
					t.Errorf("deployment not found: %v", err)
				}
				// Check Service
				var svc corev1.Service
				if err := mcpClient.Resources(targetNamespace).Get(ctx, "otel-collector", targetNamespace, &svc); err != nil {
					t.Errorf("service not found: %v", err)
				}
				return ctx
			},
		).
		Teardown(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			onboardingConfig, err := clusterutils.OnboardingConfig()
			if err != nil {
				t.Error(err)
				return ctx
			}
			for _, obj := range onboardingList.Items {
				if err := resources.DeleteObject(ctx, onboardingConfig, &obj, wait.WithTimeout(time.Minute)); err != nil {
					t.Errorf("failed to delete onboarding object: %v", err)
				}
			}
			return ctx
		}).
		Teardown(providers.DeleteMCP("test-mcp", wait.WithTimeout(5*time.Minute)))
	testenv.Test(t, basicProviderTest.Feature())
}
