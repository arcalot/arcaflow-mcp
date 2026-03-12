package workflow

import (
	"testing"
)

func TestNewInputSchemaResolver(t *testing.T) {
	resolver := NewInputSchemaResolver()
	if resolver == nil {
		t.Fatalf("expected resolver")
	}
}

// NOTE: Full engine resolution tests are omitted here as they require a 
// working container runtime or complex mocking of the Arcaflow engine SDK.
