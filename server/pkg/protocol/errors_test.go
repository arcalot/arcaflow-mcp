package protocol

import "testing"

func TestErrorObjectImplementsError(t *testing.T) {
	err := &ErrorObject{Code: ErrInvalidParams, Message: "bad params"}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
}

func TestNewErrorBuildsPayload(t *testing.T) {
	payload := newError(ErrInternal, "oops", map[string]string{"why": "test"})
	if payload.Code != ErrInternal || payload.Message != "oops" {
		t.Fatalf("unexpected error payload")
	}
	if payload.Data == nil {
		t.Fatalf("expected data payload")
	}
}
