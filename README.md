# Pipeline Inspector Sidecar

This repo is a small and streamlined implementation of a sidecar container for Crossplane,
capturing Functions' Requests and Responses and printing them to the pod logs.

The full design of this feature can be found in the [design doc](https://github.com/crossplane/crossplane/blob/main/design/one-pager-pipeline-inspector.md).

## Usage

This repository publishes release images to
`xpkg.crossplane.io/crossplane/inspector-sidecar`. This image can then be
included as a sidecar container for Crossplane through the Helm chart's
values.

```yaml
# Example:
# helm upgrade --install crossplane crossplane/crossplane \
#   -n crossplane-system --create-namespace \
#   -f pipeline-inspector-values.yaml \
#   --set image.tag=v0.0.0-hack

args:
  - --enable-pipeline-inspector
  - --pipeline-inspector-socket=/var/run/pipeline-inspector/socket

extraVolumesCrossplane:
  - name: pipeline-inspector-socket
    emptyDir: {}

extraVolumeMountsCrossplane:
  - name: pipeline-inspector-socket
    mountPath: /var/run/pipeline-inspector

sidecarsCrossplane:
  - name: pipeline-inspector
    image: crossplane/inspector-sidecar
    command:
      - /usr/local/bin/pipeline-inspector
    volumeMounts:
      - name: pipeline-inspector-socket
        mountPath: /var/run/pipeline-inspector
    resources:
      requests:
        cpu: 10m
        memory: 64Mi
      limits:
        cpu: 100m
        memory: 128Mi

```

When this container starts up, it starts a gRPC server that listens on a unix
domain socket at the default path of `/var/run/pipeline-inspector/socket`,
which Crossplane is going to send RunFunctionRequests and RunFunctionResponses
from Functions.

The gRPC server implementation in this repo accepts incoming payloads
and simply writes them to `stdout` so they will be included in the provider
pod's logs.
