/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License. You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
*/

// Package server implements a gRPC server for the Pipeline Inspector.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"

	pipelinev1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/pipelineinspector/proto/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

// Inspector implements the PipelineInspectorService by logging to a writer.
type Inspector struct {
	pipelinev1alpha1.UnimplementedPipelineInspectorServiceServer

	format string
	out    io.Writer
	log    logging.Logger
}

// Option configures an Inspector.
type Option func(*Inspector)

// WithOutput sets the output writer (default: os.Stdout).
func WithOutput(w io.Writer) Option {
	return func(i *Inspector) {
		i.out = w
	}
}

// WithLogger sets the logger for the Inspector.
func WithLogger(l logging.Logger) Option {
	return func(i *Inspector) {
		i.log = l
	}
}

// NewInspector creates a new Inspector with the given output format.
func NewInspector(format string, opts ...Option) *Inspector {
	i := &Inspector{
		format: format,
		out:    os.Stdout,
		log:    logging.NewNopLogger(),
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// EmitRequest logs the function request before execution.
func (i *Inspector) EmitRequest(_ context.Context, req *pipelinev1alpha1.EmitRequestRequest) (*pipelinev1alpha1.EmitRequestResponse, error) {
	// Decode JSON payload from bytes.
	payload := decodeJSONPayload(req.GetRequest())
	i.logEvent("REQUEST", req.GetMeta(), payload, "")
	return &pipelinev1alpha1.EmitRequestResponse{}, nil
}

// EmitResponse logs the function response after execution.
func (i *Inspector) EmitResponse(_ context.Context, req *pipelinev1alpha1.EmitResponseRequest) (*pipelinev1alpha1.EmitResponseResponse, error) {
	// Decode JSON payload from bytes.
	payload := decodeJSONPayload(req.GetResponse())
	i.logEvent("RESPONSE", req.GetMeta(), payload, req.GetError())
	return &pipelinev1alpha1.EmitResponseResponse{}, nil
}

// decodeJSONPayload decodes JSON bytes into a map for display.
func decodeJSONPayload(data []byte) any {
	if len(data) == 0 {
		return nil
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		// If we can't decode, return the raw string.
		return string(data)
	}
	return result
}

func (i *Inspector) logEvent(eventType string, meta *pipelinev1alpha1.StepMeta, payload any, errMsg string) {
	if i.format == "text" {
		i.logText(eventType, meta, payload, errMsg)
		return
	}
	i.logJSON(eventType, meta, payload, errMsg)
}

func (i *Inspector) logJSON(eventType string, meta *pipelinev1alpha1.StepMeta, payload any, errMsg string) {
	// Marshal meta using protojson to preserve proto field names.
	metaJSON, err := protojson.Marshal(meta)
	if err != nil {
		i.log.Debug("Cannot marshal meta", "error", err)
		return
	}

	// Unmarshal meta into a map so we can include it in the final event.
	var metaMap map[string]any
	if err := json.Unmarshal(metaJSON, &metaMap); err != nil {
		i.log.Debug("Cannot unmarshal meta", "error", err)
		return
	}

	event := map[string]any{
		"type":    eventType,
		"meta":    metaMap,
		"payload": payload,
	}
	if errMsg != "" {
		event["error"] = errMsg
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		i.log.Debug("Cannot marshal event", "error", err)
		return
	}

	_, _ = fmt.Fprintln(i.out, string(eventJSON))
}

func (i *Inspector) logText(eventType string, meta *pipelinev1alpha1.StepMeta, payload any, errMsg string) {
	_, _ = fmt.Fprintf(i.out, "=== %s ===\n", eventType)

	// Handle context-specific fields using type switch (idiomatic for oneofs).
	switch ctx := meta.GetContext().(type) {
	case *pipelinev1alpha1.StepMeta_CompositionMeta:
		cm := ctx.CompositionMeta
		_, _ = fmt.Fprintf(i.out, "  XR:          %s/%s (%s)\n", cm.GetCompositeResourceApiVersion(), cm.GetCompositeResourceKind(), cm.GetCompositeResourceName())
		_, _ = fmt.Fprintf(i.out, "  XR UID:      %s\n", cm.GetCompositeResourceUid())
		if ns := cm.GetCompositeResourceNamespace(); ns != "" {
			_, _ = fmt.Fprintf(i.out, "  XR NS:       %s\n", ns)
		}
		_, _ = fmt.Fprintf(i.out, "  Composition: %s\n", cm.GetCompositionName())
	case *pipelinev1alpha1.StepMeta_OperationMeta:
		om := ctx.OperationMeta
		_, _ = fmt.Fprintf(i.out, "  Operation:   %s\n", om.GetOperationName())
		_, _ = fmt.Fprintf(i.out, "  Op UID:      %s\n", om.GetOperationUid())
	}

	_, _ = fmt.Fprintf(i.out, "  Step:        %s (index %d, iteration %d)\n", meta.GetStepName(), meta.GetStepIndex(), meta.GetIteration())
	_, _ = fmt.Fprintf(i.out, "  Function:    %s\n", meta.GetFunctionName())
	_, _ = fmt.Fprintf(i.out, "  Trace ID:    %s\n", meta.GetTraceId())
	_, _ = fmt.Fprintf(i.out, "  Span ID:     %s\n", meta.GetSpanId())
	_, _ = fmt.Fprintf(i.out, "  Timestamp:   %s\n", meta.GetTimestamp().AsTime().Format("2006-01-02T15:04:05.000Z07:00"))
	if errMsg != "" {
		_, _ = fmt.Fprintf(i.out, "  Error:       %s\n", errMsg)
	}

	// Pretty-print payload as YAML for readability.
	if payload != nil {
		payloadYAML, err := yaml.Marshal(payload)
		if err == nil {
			_, _ = fmt.Fprintf(i.out, "  Payload:\n%s\n", indentLines(string(payloadYAML), "    "))
		}
	}
	_, _ = fmt.Fprintln(i.out)
}

// indentLines adds the given prefix to each line of the input string.
func indentLines(s, prefix string) string {
	var result strings.Builder
	for line := range strings.SplitSeq(strings.TrimSuffix(s, "\n"), "\n") {
		result.WriteString(prefix + line + "\n")
	}
	return result.String()
}
