# Build the manager binary
FROM golang:1.22 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT=HEAD
ARG BUILD_VERSION=main

WORKDIR /workspace

COPY Makefile Makefile

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY hack/ hack/
COPY internal/ internal/

RUN VERSION=${BUILD_VERSION} GIT_COMMIT=${GIT_COMMIT} TARGET_OS=$TARGETOS TARGET_ARCH=$TARGETARCH \
  make build

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager", "--zap-log-level=info", "--zap-encoder=console"]
