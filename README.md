# Telegraf Sidecar Operator

Use Kubernetes Pod annotations to automatically inject and configure Telegraf sidecar containers.

## Description

The Telegraf Sidecar Operator allows Kubernetes users to automatically inject and configure a telegraf sidecar container into their Pods through the use of Pod annotations.

The operator enables platform operators to centrally manage agent configuration and output credentials, while enabling platform users to customize the aspects of the telegraf configuration the care about - the inputs and tags.

The operator works best when paired with applications that expose metrics through a Prometheus-style HTTP metrics endpoints, but it is not limited to this usecase.

Pod annotations used to customize the sidecar container and configure telegraf are largely compatible with the existing [telegraf-operator](https://github.com/influxdata/telegraf-operator) project from InfluxData which seems to [no longer be maintained](https://github.com/influxdata/telegraf/issues/15192#issuecomment-2066730210), resulting in bug reports and feature requests going unanswered.

> [!NOTE]
> This project is **not** a fork of [influxdata/telegraf-operator](https://github.com/influxdata/telegraf-operator), but rather an alternate implementation that aims to address multiple issues fundamental to that projects design.

## Components

The operator consists of two primary components:

1. Mutating Admission Webhook used to inject the sidecar container into newly created Pods.
2. A controller, which detects when a Pod has been admitted into the cluster with a telegraf sidecar container, and creates a corresponding Kubernetes secret with the telegraf configuration values.

`telegraf-config` secrets are automatically removed by the Kubernetes garbage collection process when pods are deleted, no longer relying on specific event ordering or internal logic to ensure these secrets are properly cleaned up.

## Status

This project is still in active development, and may not be ready for production use cases. The project supports the majority of the pod annotations available in the InfluxData Telegraf Operator, with some notable exceptions:

- The `telegraf.influxdata.com/port` annotation has been deprecated and will be removed in a future release. Use`telegraf.influxdata.com/ports` instead.
- Istio annotations: The separate Telegraf sidecar specifically for Istio is currently not supported. Injecting a completely separate Telegraf sidecar container just to monitor the istio proxy sidecar doesn't feel like the correct solution. Please [open an issue](https://github.com/jmickey/telegraf-sidecar-operator/issues/new) if you use Istio and the current annotations are not sufficient.

## Deployment

### Prerequisites

**Kubernetes 1.19 or later**.

### Helm

The simplest way to deploy the `telegraf-sidecar-operator` is via [Helm](https://helm.sh). An alternative deployment is also available via Kustomize.

To install the most recent version of the operator via Helm:

```shell
helm repo add tso https://telegraf-sidecar-operator.mickey.dev
helm repo update telegraf-sidecar-operator
helm install telegraf-sidecar-operator tso/telegraf-sidecar-operator
```

Further documentation on the Helm installation and available customizations is available [here](./charts/telegraf-sidecar-operator).

## Kustomize

To install via Kustomize additional resources are required.

- A `Secret`, named `telegraf-classes`, with relevant class data.
- TLS certificates for the `MutatingAdmissionWebhook`.

An example script to generate TLS certificates can be found in [scripts/gencerts.sh](./scripts/gencerts.sh). An example can be found in [config/local](config/local) in this repository.

Create a file to patch the `MutatingWebhookConfiguration`:

```yaml
# webhook-cert-patch
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: telegraf-sidecar-operator
webhooks:
  - name: telegraf.mickey.dev
    clientConfig:
      caBundle: < YOUR CA BUNDLE >
```

Create a `kubernetes.io/tls` secret to store and mount the webhook serving certificates:

```yaml
# secret-tls.yaml
apiVersion: v1
data:
  tls.crt: < YOUR CRT FILE >
  tls.key: < YOUR CERT KEY>
kind: Secret
metadata:
  name: webhook-server-cert
  namespace: telegraf-sidecar-operator
type: kubernetes.io/tls
```

Create a `secret-classes.yaml` file to configure the Telegraf classes:

```yaml
# secret-classes.yaml
apiVersion: v1
kind: Secret
metadata:
  name: telegraf-classes
  namespace: telegraf-sidecar-operator
  labels:
    app: telegraf-sidecar-operator
type: Opaque
stringData:
  default: |
    [[outputs.file]]
      files = ["stdout"]
```

Create a `Kustomization` to combine the resources along with the install bundle:

```yaml
# kustomization.yaml
resources:
  - github.com/jmickey/telegraf-sidecar-operator/releases/download/<version>/telegraf-sidecar-operator-<version>.yaml
  - secret-tls.yaml
  - secret-classes.yaml

patches:
  - webhook-cert-patch.yaml
```

Deploy to the cluster:

```yaml
kubectl apply -k <PATH-TO-KUSTOMIZATION>
```

## Usage

There are two parts to configuring the `telegraf-sidecar-operator`:

1. Global configuration - in the form of "classes", mounted as files into the operator filesystem. Usually classes will contain `agent`, `output`, and `global_tags` configuration, allowing platform administrators to centrally manage this common configuration.
2. Pod configuration - via pod annotations. Allowing application developers to specify telegraf's `input` config specific to their application, such as Prometheus ports/paths to monitor.

### Classes

Each class contains a subset of Telegraf configuration and will generally define aspects such as [`agent` configuration](https://docs.influxdata.com/telegraf/v1/configuration/#agent-configuration), [`output` plugins](https://docs.influxdata.com/telegraf/v1/plugins/#output-plugins), and [`global_tags`](https://docs.influxdata.com/telegraf/v1/configuration/#global-tags).

Classes can be defined within a `Secret` or `ConfigMap` where each key maps to a separate class. For example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: telegraf-classes
  namespace: telegraf-sidecar-operator
  labels:
    app: telegraf-sidecar-operator
type: Opaque
stringData:
  default: |
    [[outputs.influxdb]]
      urls = ["http://influxdb.influxdb:8086"]
    [[outputs.file]]
      files = ["stdout"]
    [global_tags]
      hostname = "$HOSTNAME"
      nodename = "$NODENAME"
      type = "app"]
```

## Pod Annotations

Pod annotations can be used to configure both the sidecar container itself, as well as the Telegraf application configuration.

### Sidecar Annotations

| Annotation                                          | Default                | Description                                                                                                                                            |
| --------------------------------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `telegraf.influxdata.com/image`                     | `telegraf:1.30-alpine` | Override the telegraf sidecar image.                                                                                                                   |
| `telegraf.influxdata.com/requests-cpu`              | `10m`                  | Override the sidecar CPU resource requests.                                                                                                            |
| `telegraf.influxdata.com/requests-memory`           | `56Mi`                 | Override the sidecar memory resource requests.                                                                                                         |
| `telegraf.influxdata.com/limits-cpu`                | `100m`                 | Override the sidecar CPU resource limits.                                                                                                              |
| `telegraf.influxdata.com/limits-memory`             | `128Mi`                | Override the sidecar memory resource limits.                                                                                                           |
| `telegraf.influxdata.com/secret-env`                | `nil`                  | Can be used to mount a secret and all its keys and environment variables in the sidecar container.                                                     |
| `telegraf.influxdata.com/configmap-env`             | `nil`                  | Can be used to mount a configmap and all its keys and environment variables in the sidecar container.                                                  |
| `telegraf.influxdata.com/volume-mounts`             | `nil`                  | Can be used to mount additional volumes into the sidecar container. Must be in the format: `'{ "<volumeName>": "<mountPath>" }'`                         |
| `telegraf.influxdata.com/env-literal-<VAR>`         | `nil`                  | Can be used to add a literal value as an environment variable to the sidecar container.                                                                |
| `telegraf.influxdata.com/env-fieldred-<VAR>`        | `nil`                  | Can be used to add a Kubernetes downstream API FieldRef as an environment variable to the sidecar container.                                           |
| `telegraf.influxdata.com/env-secretkeyref-<VAR>`    | `nil`                  | Can be used to a Secret key value as an environment variable to the sidecar container. Must be in the format `"<secretName>.<secretKey>"`              |
| `telegraf.influxdata.com/env-configmapkeyraf-<VAR>` | `nil`                  | Can be used to add a ConfigMap key value as an environment variable to the sidecar container. Must be in the format `"<configMapName>.<configMapKey>"` |

### Telegraf Configuration Annotations

| Annotation                                         | Default             | Description                                                                                                                                                                                                                                                                                                 |
| -------------------------------------------------- | ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `telegraf.influxdata.com/class`                    | `default`           | Specifies which telegraf config class to use. Classes are configured in the operator.                                                                                                                                                                                                                       |
| `telegraf.influxdata.com/ports`                    | `nil`               | Can be used to configure one or more ports to be scraped by the Prometheus input plugin. Must be a string of comma separated values.                                                                                                                                                                        |
| `telegraf.influxdata.com/path`                     | `/metrics`          | Can be used to override the HTTP path to be scraped by the Prometheus input plugin. Applies to all ports if multiple are provided.                                                                                                                                                                          |
| `telegraf.influxdata.com/scheme`                   | `http`              | Can be used to override the request scheme when scraping metrics with the Prometheus input plugin. Valid values are [ `http`, `https` ].                                                                                                                                                                    |
| `telegraf.influxdata.com/interval`                 | `10s`               | Can be used to configure the scraping interval. Value must be a value to Go style duration string, e.g. `10s`, `30s`, `1m`.                                                                                                                                                                                 |
| `telegraf.influxdata.com/metric-version`           | `"1"`               | Can be used to override which metrics parsing version to use. Valid values are [ `"1"`, `"2"`].                                                                                                                                                                                                             |
| `telegraf.influxdata.com/namepass`                 | `nil`               | Can be used to configure the namepass setting for the Prometheus input plugin. Namepass accepts an array of glob pattern strings. Only metrics whose measurement name matches a pattern in this list are emitted. Annotation value must be specified as a comma-separated string, e.g. `"metric1, metric2"` |
| `telegraf.influxdata.com/inputs`                   | `nil`               | Can be used to configure a raw telegraf input TOML block. Can be provided as a multiline block of raw TOML configuration.                                                                                                                                                                                   |
| `telegraf.influxdata.com/internal`                 | Configured globally | Enables the "internal" telegraf plugin if it is configured to be globally disabled by default. Any non-empty string value is accepted.                                                                                                                                                                      |
| `telegraf.influxdata.com/global-tag-literal-<TAG>` | `nil`               | Can be used to a literal value to the `global_tags` in the telegraf configuration.                                                                                                                                                                                                                          |

### Example

```yaml
apiVersion: apps/v1
kind: StatefulSet
# ...
spec:
  template:
    metadata:
      annotations:
        telegraf.influxdata.com/class: infra # User defined output class
        telegraf.influxdata.com/interval: 30s
        telegraf.influxdata.com/path: /prometheus/metrics
        telegraf.influxdata.com/port: "8086"
        telegraf.influxdata.com/scheme: https
        telegraf.influxdata.com/metric-version: "2"
        telegraf.influxdata.com/env-fieldref-APP: metadata.labels['app']
        telegraf.influxdata.com/global-tag-literal-app: "$APP"
      # ...
```

## Contributing

// TODO

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
