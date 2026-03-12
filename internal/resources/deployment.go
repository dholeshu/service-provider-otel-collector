package resources

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentOpts holds configuration for the OTEL collector deployment.
type DeploymentOpts struct {
	Image            string
	Version          string
	Resources        *corev1.ResourceRequirements
	ImagePullSecrets []corev1.LocalObjectReference
	ConfigHash       string
}

// ReconcileDeployment creates or updates the OTEL collector Deployment in the MCP.
func ReconcileDeployment(ctx context.Context, c client.Client, ns string, opts DeploymentOpts) error {
	replicas := int32(1)
	imageRef := fmt.Sprintf("%s:%s", opts.Image, opts.Version)

	labels := map[string]string{
		AppLabel:       AppValue,
		ManagedByLabel: ManagedByValue,
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName,
			Namespace: ns,
		},
	}

	_, err := ctrl.CreateOrUpdate(ctx, c, deployment, func() error {
		deployment.Labels = labels
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					AppLabel: AppValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						ConfigHashAnnotation: opts.ConfigHash,
					},
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: opts.ImagePullSecrets,
					Containers: []corev1.Container{
						{
							Name:  ContainerName,
							Image: imageRef,
							Command: []string{
								"/otelcol-contrib",
								"--config=/conf/otel-collector-config.yaml",
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: SecretName},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{Name: "otlp-grpc", ContainerPort: 4317, Protocol: corev1.ProtocolTCP},
								{Name: "otlp-http", ContainerPort: 4318, Protocol: corev1.ProtocolTCP},
								{Name: "metrics", ContainerPort: 8888, Protocol: corev1.ProtocolTCP},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      ConfigVolumeName,
									MountPath: "/conf",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: ConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ConfigMapName,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  ConfigMapKey,
											Path: "otel-collector-config.yaml",
										},
									},
								},
							},
						},
					},
				},
			},
		}
		if opts.Resources != nil {
			deployment.Spec.Template.Spec.Containers[0].Resources = *opts.Resources
		}
		return nil
	})
	return err
}

// ReconcileService creates or updates the ClusterIP Service for the OTEL collector.
func ReconcileService(ctx context.Context, c client.Client, ns string) error {
	labels := map[string]string{
		AppLabel:       AppValue,
		ManagedByLabel: ManagedByValue,
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: ns,
		},
	}

	_, err := ctrl.CreateOrUpdate(ctx, c, svc, func() error {
		svc.Labels = labels
		svc.Spec = corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				AppLabel: AppValue,
			},
			Ports: []corev1.ServicePort{
				{Name: "otlp-grpc", Port: 4317, TargetPort: intstr.FromInt(4317), Protocol: corev1.ProtocolTCP},
				{Name: "otlp-http", Port: 4318, TargetPort: intstr.FromInt(4318), Protocol: corev1.ProtocolTCP},
				{Name: "metrics", Port: 8888, TargetPort: intstr.FromInt(8888), Protocol: corev1.ProtocolTCP},
			},
		}
		return nil
	})
	return err
}
