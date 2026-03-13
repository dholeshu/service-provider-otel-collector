# Production Dockerfile for service-provider-otel-collector
#
# Security: Uses minimal base image from SAP internal registry
# Multi-stage build for smaller final image

# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -ldflags="-s -w" \
    -o service-provider-otel-collector \
    ./cmd/service-provider-otel-collector

# Production stage
# Using SAP internal registry for security and compliance
FROM crimson-prod.common.repositories.cloud.sap/distroless/base:nonroot-amd64

LABEL maintainer="SAP AICore CloudOps EU Team"
LABEL description="Service Provider for OpenTelemetry Collector on OpenMCP"
LABEL org.opencontainers.image.source="https://github.com/openmcp-project/service-provider-otel-collector"

WORKDIR /
COPY --from=builder /workspace/service-provider-otel-collector /service-provider-otel-collector

# Use non-root user (65532 = nonroot user in distroless)
USER 65532:65532

ENTRYPOINT ["/service-provider-otel-collector"]
