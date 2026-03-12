package resources

const (
	// ConfigMapName is the well-known name for the OTEL collector config in the MCP
	ConfigMapName = "otel-collector-conf"
	// ConfigMapKey is the key in the ConfigMap that holds the OTEL collector config
	ConfigMapKey = "otel-collector-config"

	// SecretName is the well-known name for the XSUAA credentials secret in the MCP
	SecretName = "otel-router-xsuaa-secret"

	// DeploymentName is the name for the OTEL collector deployment
	DeploymentName = "otel-collector"
	// ServiceName is the name for the OTEL collector service
	ServiceName = "otel-collector"

	// ContainerName is the name of the collector container
	ContainerName = "otel-collector"

	// ConfigVolumeName is the volume name for the config mount
	ConfigVolumeName = "otel-collector-config-vol"

	// ManagedByLabel is the label used to identify resources managed by this service provider
	ManagedByLabel = "app.kubernetes.io/managed-by"
	// ManagedByValue is the value for the managed-by label
	ManagedByValue = "service-provider-otel-collector"

	// AppLabel is the label used to identify the app
	AppLabel = "app"
	// AppValue is the value for the app label
	AppValue = "otel-collector"

	// ConfigHashAnnotation is the annotation used to trigger pod restarts on config changes
	ConfigHashAnnotation = "otelcollector.services.openmcp.cloud/config-hash"
)
