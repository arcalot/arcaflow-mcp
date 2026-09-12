package workflow

import (
	"math"
	"testing"
)

func TestMarshalDeterministicYAMLScalars(t *testing.T) {
	payload, err := marshalDeterministicYAML(map[string]interface{}{
		"flag":    true,
		"ratio":   1.5,
		"note":    "ok",
		"nothing": nil,
		"items":   []interface{}{1, "two"},
	})
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
    if len(payload) == 0 {
        t.Fatalf("expected payload")
    }
}

func TestMarshalDeterministicJSONOrdering(t *testing.T) {
	payload, err := marshalDeterministicJSON(map[string]interface{}{
		"b": "two",
		"a": []interface{}{1, 2},
	})
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if string(payload) != `{"a":[1,2],"b":"two"}` {
		t.Fatalf("unexpected json output: %s", string(payload))
	}
}

func TestMarshalDeterministicJSONRejectsNaN(t *testing.T) {
	if _, err := marshalDeterministicJSON(math.NaN()); err == nil {
		t.Fatalf("expected NaN marshal error")
	}
}
