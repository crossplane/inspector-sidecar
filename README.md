# Pipeline Inspector Sidecar

A minimal reference implementation of a sidecar container for Crossplane that captures
function pipeline execution data (requests and responses) and logs them to stdout.

The full design of this feature can be found in the [design doc](https://github.com/crossplane/crossplane/blob/main/design/one-pager-pipeline-inspector.md).

## Features

- Captures `RunFunctionRequest` and `RunFunctionResponse` data for each function invocation
- Supports JSON and human-readable text output formats
- Correlates pipeline steps using trace IDs, span IDs, and step indices
- Runs as a non-root user in a minimal distroless container

## CLI Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--socket-path` | `PIPELINE_INSPECTOR_SOCKET` | `/var/run/pipeline-inspector/socket` | Unix socket path to listen on |
| `--format` | - | `json` | Output format (`json` or `text`) |
| `--max-recv-msg-size` | `MAX_RECV_MSG_SIZE` | `4194304` (4MB) | Maximum gRPC receive message size in bytes |
| `--shutdown-timeout` | `SHUTDOWN_TIMEOUT` | `5s` | Graceful shutdown timeout |

## Usage

This repository publishes release images to
`xpkg.crossplane.io/crossplane/inspector-sidecar`. This image can be
included as a sidecar container for Crossplane through the Helm chart's values.

```yaml
# Example:
# helm upgrade --install crossplane crossplane/crossplane \
#   -n crossplane-system --create-namespace \
#   -f pipeline-inspector-values.yaml

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
    image: xpkg.crossplane.io/crossplane/inspector-sidecar:latest
    args:
      - --format=json
      # Increase if your function payloads exceed 4MB
      # - --max-recv-msg-size=8388608
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

## Output Formats

### JSON Format (default)

```json
{"meta":{"compositeResourceApiVersion":"example.org/v1","compositeResourceKind":"XDatabase","compositeResourceName":"my-db","compositeResourceNamespace":"default","compositeResourceUid":"abc-123","compositionName":"my-composition","functionName":"function-patch-and-transform","iteration":0,"spanId":"span-456","stepIndex":0,"timestamp":"2026-01-15T10:30:00Z","traceId":"trace-789"},"payload":{...},"type":"REQUEST"}
```

### Text Format

Use `--format=text` for human-readable output:

```
=== REQUEST ===
  XR:          example.org/v1/XDatabase (my-db)
  XR UID:      abc-123
  XR NS:       default
  Composition: my-composition
  Function:    function-patch-and-transform (step 0, iteration 0)
  Trace ID:    trace-789
  Span ID:     span-456
  Timestamp:   2026-01-15T10:30:00.000Z
  Payload:
    apiVersion: apiextensions.crossplane.io/v1
    ...
```

## Building

```bash
# Build locally
go build -o inspector-sidecar .

# Build Docker image
docker build -t inspector-sidecar .
```

## Testing

```bash
go test ./...
```
