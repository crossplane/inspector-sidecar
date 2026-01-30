module github.com/crossplane/inspector-sidecar

go 1.24.9

require (
	github.com/alecthomas/kong v1.10.0
	github.com/crossplane/crossplane-runtime/v2 v2.2.0-rc.0.0.20260130110818-b375c81880a3
	github.com/go-logr/zapr v1.3.0
	github.com/google/go-cmp v0.7.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.11
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
)
