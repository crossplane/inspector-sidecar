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

	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/pipelineinspector/proto/v1alpha1"
)

func TestEmitRequest_JSON(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("json", WithOutput(&buf))

	meta := &pipelinev1alpha1.StepMeta{
		TraceId:                     "trace-123",
		SpanId:                      "span-456",
		StepIndex:                   0,
		Iteration:                   0,
		FunctionName:                "function-patch-and-transform",
		CompositionName:             "my-composition",
		CompositeResourceUid:        "uid-789",
		CompositeResourceName:       "my-xr",
		CompositeResourceNamespace:  "default",
		CompositeResourceApiVersion: "example.org/v1",
		CompositeResourceKind:       "XDatabase",
		Timestamp:                   timestamppb.New(time.Now()),
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
		CompositeResourceApiVersion: "example.org/v1",
		CompositeResourceKind:       "XDatabase",
		CompositeResourceName:       "my-xr",
		CompositeResourceUid:        "uid-123",
		CompositeResourceNamespace:  "my-namespace",
		CompositionName:             "my-composition",
		FunctionName:                "my-function",
		TraceId:                     "trace-abc",
		SpanId:                      "span-def",
		StepIndex:                   1,
		Iteration:                   2,
		Timestamp:                   timestamppb.New(time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)),
	}

	req := &pipelinev1alpha1.EmitRequestRequest{
		Request: []byte(`{"apiVersion":"apiextensions.crossplane.io/v1"}`),
		Meta:    meta,
	}

	_, _ = inspector.EmitRequest(context.Background(), req)

	output := buf.String()

	// Verify header.
	if !strings.Contains(output, "=== REQUEST ===") {
		t.Errorf("expected REQUEST header, got: %s", output)
	}

	// Verify XR info.
	if !strings.Contains(output, "XR:          example.org/v1/XDatabase (my-xr)") {
		t.Errorf("expected XR info in output, got: %s", output)
	}

	// Verify UID is included.
	if !strings.Contains(output, "XR UID:      uid-123") {
		t.Errorf("expected XR UID in output, got: %s", output)
	}

	// Verify namespace is included.
	if !strings.Contains(output, "XR NS:       my-namespace") {
		t.Errorf("expected XR namespace in output, got: %s", output)
	}

	// Verify composition name.
	if !strings.Contains(output, "Composition: my-composition") {
		t.Errorf("expected composition name in output, got: %s", output)
	}

	// Verify function info.
	if !strings.Contains(output, "Function:    my-function (step 1, iteration 2)") {
		t.Errorf("expected function info in output, got: %s", output)
	}

	// Verify trace and span IDs.
	if !strings.Contains(output, "Trace ID:    trace-abc") {
		t.Errorf("expected trace ID in output, got: %s", output)
	}
	if !strings.Contains(output, "Span ID:     span-def") {
		t.Errorf("expected span ID in output, got: %s", output)
	}
}

func TestEmitRequest_Text_NoNamespace(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("text", WithOutput(&buf))

	// Cluster-scoped resource has empty namespace.
	meta := &pipelinev1alpha1.StepMeta{
		CompositeResourceApiVersion: "example.org/v1",
		CompositeResourceKind:       "XClusterDatabase",
		CompositeResourceName:       "my-cluster-xr",
		CompositeResourceUid:        "uid-456",
		CompositeResourceNamespace:  "", // Empty for cluster-scoped.
		CompositionName:             "cluster-composition",
		FunctionName:                "my-function",
		Timestamp:                   timestamppb.New(time.Now()),
	}

	req := &pipelinev1alpha1.EmitRequestRequest{
		Request: []byte(`{}`),
		Meta:    meta,
	}

	_, _ = inspector.EmitRequest(context.Background(), req)

	output := buf.String()

	// Verify namespace line is NOT included for cluster-scoped resources.
	if strings.Contains(output, "XR NS:") {
		t.Errorf("expected no XR NS line for cluster-scoped resource, got: %s", output)
	}
}

func TestEmitResponse_Text_WithError(t *testing.T) {
	var buf bytes.Buffer
	inspector := NewInspector("text", WithOutput(&buf))

	meta := &pipelinev1alpha1.StepMeta{
		FunctionName: "failing-function",
		Timestamp:    timestamppb.New(time.Now()),
	}

	req := &pipelinev1alpha1.EmitResponseRequest{
		Response: nil,
		Error:    "something went wrong",
		Meta:     meta,
	}

	_, _ = inspector.EmitResponse(context.Background(), req)

	output := buf.String()
	if !strings.Contains(output, "Error:       something went wrong") {
		t.Errorf("expected error in output, got: %s", output)
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
			result := indentLines(tt.input, tt.prefix)
			if result != tt.expected {
				t.Errorf("indentLines(%q, %q) = %q, want %q", tt.input, tt.prefix, result, tt.expected)
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
