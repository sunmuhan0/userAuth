package trace

import (
	"context"
	"testing"
)

func TestNewTraceID(t *testing.T) {
	id := NewTraceID()
	if id == "" {
		t.Fatal("trace ID should not be empty")
	}
	if len(id) != 32 {
		t.Fatalf("expected 32 hex chars, got %d (len=%d)", len(id), len(id))
	}
}

func TestNewTraceIDUnique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewTraceID()
		if ids[id] {
			t.Fatal("trace IDs should be unique")
		}
		ids[id] = true
	}
}

func TestWithAndGetTraceID(t *testing.T) {
	ctx := context.Background()
	id := NewTraceID()

	ctx = WithTraceID(ctx, id)
	got := GetTraceID(ctx)

	if got != id {
		t.Fatalf("expected '%s', got '%s'", id, got)
	}
}

func TestGetTraceIDNoContext(t *testing.T) {
	got := GetTraceID(nil)
	if got != "" {
		t.Fatalf("expected empty string, got '%s'", got)
	}
}

func TestGetTraceIDNoValue(t *testing.T) {
	ctx := context.Background()
	got := GetTraceID(ctx)
	if got != "" {
		t.Fatalf("expected empty string, got '%s'", got)
	}
}

func TestConstants(t *testing.T) {
	if MetadataKey != "x-trace-id" {
		t.Fatalf("expected MetadataKey='x-trace-id', got '%s'", MetadataKey)
	}
	if HeaderKey != "X-Trace-Id" {
		t.Fatalf("expected HeaderKey='X-Trace-Id', got '%s'", HeaderKey)
	}
	if PropertyKey != "trace_id" {
		t.Fatalf("expected PropertyKey='trace_id', got '%s'", PropertyKey)
	}
}
