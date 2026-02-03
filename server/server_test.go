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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/pipelineinspector/proto/v1alpha1"
)

func TestEmitRequest_JSON(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("json", WithOutput(&buf))

	meta := &pipelinev1alpha1.StepMeta{
		TraceId:      "trace-123",
		SpanId:       "span-456",
		StepIndex:    0,
		Iteration:    0,
		FunctionName: "function-patch-and-transform",
		Timestamp:    timestamppb.New(time.Now()),
		Context: &pipelinev1alpha1.StepMeta_CompositionMeta{
			CompositionMeta: &pipelinev1alpha1.CompositionMeta{
				CompositionName:             "my-composition",
				CompositeResourceUid:        "uid-789",
				CompositeResourceName:       "my-xr",
				CompositeResourceNamespace:  "default",
				CompositeResourceApiVersion: "example.org/v1",
				CompositeResourceKind:       "XDatabase",
			},
		},
	}

	req := &pipelinev1alpha1.EmitRequestRequest{
		Request: []byte(`{"apiVersion":"apiextensions.crossplane.io/v1"}`),
		Meta:    meta,
	}

	_, err := inspector.EmitRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("EmitRequest failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"type":"REQUEST"`) {
		t.Errorf("expected type REQUEST in output, got: %s", output)
	}
	if !strings.Contains(output, `"functionName":"function-patch-and-transform"`) {
		t.Errorf("expected functionName in output, got: %s", output)
	}

	// Verify it's valid JSON.
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify meta is included as a nested object.
	if _, ok := result["meta"]; !ok {
		t.Errorf("expected meta field in output, got: %s", output)
	}
}

func TestEmitResponse_JSON(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("json", WithOutput(&buf))

	meta := &pipelinev1alpha1.StepMeta{
		TraceId:      "trace-123",
		SpanId:       "span-456",
		FunctionName: "my-function",
		Timestamp:    timestamppb.New(time.Now()),
	}

	req := &pipelinev1alpha1.EmitResponseRequest{
		Response: []byte(`{"desired":{}}`),
		Error:    "",
		Meta:     meta,
	}

	_, err := inspector.EmitResponse(context.Background(), req)
	if err != nil {
		t.Fatalf("EmitResponse failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"type":"RESPONSE"`) {
		t.Errorf("expected type RESPONSE in output, got: %s", output)
	}

	// Verify it's valid JSON.
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestEmitResponse_WithError(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("json", WithOutput(&buf))

	meta := &pipelinev1alpha1.StepMeta{
		FunctionName: "my-function",
		Timestamp:    timestamppb.New(time.Now()),
	}
	req := &pipelinev1alpha1.EmitResponseRequest{
		Response: nil,
		Error:    "function execution failed",
		Meta:     meta,
	}

	_, _ = inspector.EmitResponse(context.Background(), req)

	output := buf.String()
	if !strings.Contains(output, `"error":"function execution failed"`) {
		t.Errorf("expected error field in output, got: %s", output)
	}
}

func TestEmitRequest_Text(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("text", WithOutput(&buf))

	meta := &pipelinev1alpha1.StepMeta{
		StepName:     "my-step",
		FunctionName: "my-function",
		TraceId:      "trace-abc",
		SpanId:       "span-def",
		StepIndex:    1,
		Iteration:    2,
		Timestamp:    timestamppb.New(time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)),
		Context: &pipelinev1alpha1.StepMeta_CompositionMeta{
			CompositionMeta: &pipelinev1alpha1.CompositionMeta{
				CompositeResourceApiVersion: "example.org/v1",
				CompositeResourceKind:       "XDatabase",
				CompositeResourceName:       "my-xr",
				CompositeResourceUid:        "uid-123",
				CompositeResourceNamespace:  "my-namespace",
				CompositionName:             "my-composition",
			},
		},
	}

	req := &pipelinev1alpha1.EmitRequestRequest{
		Request: []byte(`{"apiVersion":"apiextensions.crossplane.io/v1"}`),
		Meta:    meta,
	}

	_, _ = inspector.EmitRequest(context.Background(), req)

	want := `=== REQUEST ===
  XR:          example.org/v1/XDatabase (my-xr)
  XR UID:      uid-123
  XR NS:       my-namespace
  Composition: my-composition
  Step:        my-step (index 1, iteration 2)
  Function:    my-function
  Trace ID:    trace-abc
  Span ID:     span-def
  Timestamp:   2026-01-15T10:30:00.000Z
  Payload:
    apiVersion: apiextensions.crossplane.io/v1


`
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("EmitRequest text output mismatch (-want +got):\n%s", diff)
	}
}

func TestEmitRequest_Text_NoNamespace(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("text", WithOutput(&buf))

	// Cluster-scoped resource has empty namespace.
	meta := &pipelinev1alpha1.StepMeta{
		StepName:     "my-step",
		FunctionName: "my-function",
		Timestamp:    timestamppb.New(time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)),
		Context: &pipelinev1alpha1.StepMeta_CompositionMeta{
			CompositionMeta: &pipelinev1alpha1.CompositionMeta{
				CompositeResourceApiVersion: "example.org/v1",
				CompositeResourceKind:       "XClusterDatabase",
				CompositeResourceName:       "my-cluster-xr",
				CompositeResourceUid:        "uid-456",
				CompositeResourceNamespace:  "", // Empty for cluster-scoped.
				CompositionName:             "cluster-composition",
			},
		},
	}

	req := &pipelinev1alpha1.EmitRequestRequest{
		Request: []byte(`{}`),
		Meta:    meta,
	}

	_, _ = inspector.EmitRequest(context.Background(), req)

	want := "=== REQUEST ===\n" +
		"  XR:          example.org/v1/XClusterDatabase (my-cluster-xr)\n" +
		"  XR UID:      uid-456\n" +
		"  Composition: cluster-composition\n" +
		"  Step:        my-step (index 0, iteration 0)\n" +
		"  Function:    my-function\n" +
		"  Trace ID:    \n" +
		"  Span ID:     \n" +
		"  Timestamp:   2026-01-15T10:30:00.000Z\n" +
		"  Payload:\n" +
		"    {}\n" +
		"\n\n"
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("EmitRequest text output mismatch (-want +got):\n%s", diff)
	}
}

func TestEmitResponse_Text_WithError(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("text", WithOutput(&buf))

	meta := &pipelinev1alpha1.StepMeta{
		StepName:     "failing-step",
		FunctionName: "failing-function",
		Timestamp:    timestamppb.New(time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)),
		Context: &pipelinev1alpha1.StepMeta_CompositionMeta{
			CompositionMeta: &pipelinev1alpha1.CompositionMeta{
				CompositeResourceApiVersion: "example.org/v1",
				CompositeResourceKind:       "XDatabase",
				CompositeResourceName:       "my-xr",
			},
		},
	}

	req := &pipelinev1alpha1.EmitResponseRequest{
		Response: nil,
		Error:    "something went wrong",
		Meta:     meta,
	}

	_, _ = inspector.EmitResponse(context.Background(), req)

	want := "=== RESPONSE ===\n" +
		"  XR:          example.org/v1/XDatabase (my-xr)\n" +
		"  XR UID:      \n" +
		"  Composition: \n" +
		"  Step:        failing-step (index 0, iteration 0)\n" +
		"  Function:    failing-function\n" +
		"  Trace ID:    \n" +
		"  Span ID:     \n" +
		"  Timestamp:   2026-01-15T10:30:00.000Z\n" +
		"  Error:       something went wrong\n" +
		"\n"
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("EmitResponse text output mismatch (-want +got):\n%s", diff)
	}
}

func TestDecodeJSONPayload(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantNil  bool
		wantType string
	}{
		{
			name:    "empty input",
			input:   nil,
			wantNil: true,
		},
		{
			name:     "valid json object",
			input:    []byte(`{"key":"value"}`),
			wantType: "map[string]interface {}",
		},
		{
			name:     "valid json array",
			input:    []byte(`[1,2,3]`),
			wantType: "[]interface {}",
		},
		{
			name:     "invalid json returns string",
			input:    []byte(`{not valid`),
			wantType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeJSONPayload(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			gotType := typeString(result)
			if gotType != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, gotType)
			}
		})
	}
}

func typeString(v any) string {
	if v == nil {
		return "<nil>"
	}
	switch v.(type) {
	case map[string]any:
		return "map[string]interface {}"
	case []any:
		return "[]interface {}"
	case string:
		return "string"
	default:
		return "unknown"
	}
}

func TestIndentLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prefix   string
		expected string
	}{
		{
			name:     "single line",
			input:    "hello",
			prefix:   "  ",
			expected: "  hello\n",
		},
		{
			name:     "multiple lines",
			input:    "line1\nline2\nline3",
			prefix:   "  ",
			expected: "  line1\n  line2\n  line3\n",
		},
		{
			name:     "with trailing newline",
			input:    "line1\nline2\n",
			prefix:   "    ",
			expected: "    line1\n    line2\n",
		},
		{
			name:     "empty string",
			input:    "",
			prefix:   "  ",
			expected: "  \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentLines(tt.input, tt.prefix)
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("indentLines(%q, %q) mismatch (-want +got):\n%s", tt.input, tt.prefix, diff)
			}
		})
	}
}

func TestNewInspector_Defaults(t *testing.T) {
	inspector := NewInspector("json")

	if inspector.format != "json" {
		t.Errorf("expected format 'json', got %s", inspector.format)
	}
	if inspector.out == nil {
		t.Error("expected out to be set")
	}
	if inspector.log == nil {
		t.Error("expected log to be set")
	}
}

func TestNewInspector_WithOptions(t *testing.T) {
	var out bytes.Buffer
	inspector := NewInspector("text", WithOutput(&out))

	if inspector.format != "text" {
		t.Errorf("expected format 'text', got %s", inspector.format)
	}

	// Verify custom writers are used.
	inspector.EmitRequest(context.Background(), &pipelinev1alpha1.EmitRequestRequest{
		Meta: &pipelinev1alpha1.StepMeta{
			Timestamp: timestamppb.New(time.Now()),
		},
	})

	if out.Len() == 0 {
		t.Error("expected output to be written to custom writer")
	}
}
