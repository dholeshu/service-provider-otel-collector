[![REUSE status](https://api.reuse.software/badge/github.com/openmcp-project/service-provider-otel-collector)](https://api.reuse.software/info/github.com/openmcp-project/service-provider-otel-collector)

# Service Provider: OTEL Collector

An [OpenMCP](https://github.com/openmcp-project) Service Provider that automates the deployment and lifecycle management of [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) instances into Managed Control Planes (MCPs).

## Overview

This service provider installs an OpenTelemetry Collector into each MCP that requests one. Instead of generating or templating the collector configuration, it follows a **bring-your-own-config** approach: the user provides the full OTEL collector configuration via a ConfigMap and any credentials via a generic Secret directly in the MCP. All secret keys are injected as environment variables into the collector container and can be referenced in the OTEL config using `${env:KEY_NAME}`. The service provider waits for both resources to exist, then deploys the collector.

### Architecture

```
Platform Cluster                  MCP (per tenant)
┌─────────────────────┐           ┌──────────────────────────────────┐
│  ProviderConfig     │           │  namespace: observability        │
│  (cluster-scoped)   │           │                                  │
│  - defaultImage     │           │  ConfigMap: otel-collector-conf  │ ← user creates
│  - defaultVersion   │           │  Secret: otel-collector-secret   │ ← user creates
│  - imagePullSecrets │           │                                  │
└─────────────────────┘           │  Deployment: otel-collector      │ ← SP creates
                                  │  Service: otel-collector         │ ← SP creates
Onboarding Cluster                └──────────────────────────────────┘
┌─────────────────────┐
│  OtelCollectorService│ ← one per MCP
│  (per-MCP overrides) │
└─────────────────────┘
```

### Reconciliation Flow

1. Set status to `Progressing`
2. Ensure the target namespace exists in the MCP
3. Sync image pull secrets from the platform cluster to the MCP (if configured)
4. **Check prerequisites** — does ConfigMap `otel-collector-conf` and Secret `otel-collector-secret` exist?
   - **No** — stay `Progressing` ("Waiting for ConfigMap and Secret"), requeue after 30s
   - **Yes** — continue
5. Create/update the Deployment (referencing the user-provided ConfigMap and Secret)
6. Create/update the ClusterIP Service (ports 4317, 4318, 8888)
7. Set status to `Ready`

On **deletion**, the service provider removes the Deployment and Service but leaves the user-managed ConfigMap and Secret intact.

### Config Change Detection

The service provider computes a SHA-256 hash of both the ConfigMap and Secret data and stores it as a pod template annotation (`otelcollector.services.openmcp.cloud/config-hash`). When either changes, the hash changes on the next reconciliation, which triggers a rolling restart of the collector pods to pick up the new configuration.

## API

### OtelCollectorService (onboarding cluster)

Created per MCP to request an OTEL collector installation. All fields are optional overrides — defaults come from the ProviderConfig.

```yaml
apiVersion: otelcollector.services.openmcp.cloud/v1alpha1
kind: OtelCollectorService
metadata:
  name: my-mcp
spec:
  # All fields are optional — defaults from ProviderConfig are used if omitted
  collectorImage: "otel/opentelemetry-collector-contrib"
  collectorVersion: "0.146.1"
  namespace: "observability"
  resources:
    requests:
      cpu: "200m"
      memory: "256Mi"
    limits:
      cpu: "1"
      memory: "512Mi"
```

### ProviderConfig (platform cluster)

Cluster-scoped resource that provides default values for all MCPs.

```yaml
apiVersion: otelcollector.services.openmcp.cloud/v1alpha1
kind: ProviderConfig
metadata:
  name: otelcollectorservice
spec:
  pollInterval: 1m
  defaultImage: "otel/opentelemetry-collector-contrib"
  defaultVersion: "0.146.1"
  defaultNamespace: "observability"
  imagePullSecrets:
    - name: my-registry-secret
  defaultResources:
    requests:
      cpu: "200m"
      memory: "256Mi"
```

## Prerequisites in the MCP

The service provider **does not create or manage** these resources. They must be created by the user or another system directly in the MCP before the collector can be deployed.

### ConfigMap: `otel-collector-conf`

Contains the full OpenTelemetry Collector configuration under the key `otel-collector-config`.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-conf
  namespace: observability
data:
  otel-collector-config: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    processors:
      batch: {}
    exporters:
      debug:
        verbosity: detailed
    extensions:
      health_check:
        endpoint: 0.0.0.0:13133
    service:
      extensions: [health_check]
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [debug]
```

> **Important:** The `health_check` extension on port 13133 is required. The service provider configures liveness and readiness probes that check this endpoint. Without it, the collector pods will be restarted by Kubernetes.

### Secret: `otel-collector-secret`

Contains credentials or other sensitive values needed by the collector. All keys from this secret are injected as environment variables into the collector container. You can reference them in your OTEL config using the collector's `${env:KEY_NAME}` syntax.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: otel-collector-secret
  namespace: observability
type: Opaque
stringData:
  # Example: XSUAA OAuth credentials
  TOKEN_URL: "https://your-auth-server.example.com/oauth/token"
  CLIENT_ID: "your-client-id"
  CLIENT_SECRET: "your-client-secret"
  # Example: API key for a different exporter
  API_KEY: "your-api-key"
```

Then reference them in the ConfigMap config:

```yaml
extensions:
  oauth2client:
    client_id: "${env:CLIENT_ID}"
    client_secret: "${env:CLIENT_SECRET}"
    token_url: "${env:TOKEN_URL}"
exporters:
  otlphttp:
    endpoint: https://my-endpoint
    headers:
      Authorization: "Bearer ${env:API_KEY}"
```

## Exposed Ports

The collector Deployment and Service expose:

| Port | Protocol | Name | Purpose |
|------|----------|------|---------|
| 4317 | TCP | otlp-grpc | OTLP gRPC receiver |
| 4318 | TCP | otlp-http | OTLP HTTP receiver |
| 8888 | TCP | metrics | Collector internal metrics |

Health probes use port 13133 (the collector's built-in health check extension).

## Project Structure

```
├── api/
│   ├── v1alpha1/                    # API types (OtelCollectorService, ProviderConfig)
│   └── crds/                        # Embedded CRD manifests
├── cmd/
│   └── service-provider-otel-collector/  # Entrypoint
├── internal/
│   ├── controller/                  # Reconciler (CreateOrUpdate / Delete)
│   └── resources/                   # Kubernetes resource helpers
│       ├── constants.go             # Well-known names and labels
│       ├── namespace.go             # Namespace reconciliation
│       ├── prerequisites.go         # ConfigMap/Secret existence check + config hash
│       ├── deployment.go            # Deployment and Service reconciliation
│       ├── imagepullsecret.go       # Image pull secret sync (platform → MCP)
│       └── cleanup.go              # Managed resource deletion
├── pkg/
│   └── spruntime/                   # Generic SP/PC reconciler framework
├── test/
│   └── e2e/                         # End-to-end tests
└── hack/                            # Build tooling
```

## Development

### Prerequisites

- Go 1.24+
- [Task](https://taskfile.dev/) (build system)
- Access to an OpenMCP environment (for e2e tests)

### Build

```bash
go build ./...
```

### Run Tests

```bash
# Unit tests
task test

# End-to-end tests (requires cluster access)
task test-e2e
```

### Generate CRDs and DeepCopy

```bash
task generate
```

### Validate (lint + formatting)

```bash
task validate
```

### CLI Flags

The service provider binary supports the following runtime flags:

- `--environment`: Name of the environment
- `--provider-name`: Name of the provider resource
- `--metrics-bind-address`: Address for the metrics endpoint (default: `0`)
- `--health-probe-bind-address`: Address for health probe endpoint (default: `:8081`)
- `--leader-elect`: Enable leader election (default: `false`)
- `--metrics-secure`: Serve metrics via HTTPS (default: `true`)
- `--enable-http2`: Enable HTTP/2 (default: `false`)
- `--verbosity`: Logging verbosity level

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/openmcp-project/service-provider-otel-collector/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure

If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/openmcp-project/service-provider-otel-collector/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and service-provider-otel-collector contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/service-provider-otel-collector).
