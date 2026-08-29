package apperr

import (
	"errors"
	"testing"
)

func TestNewPreservesSemanticType(t *testing.T) {
	err := New(KindBadParameter, "invalid name")

	var target *Error
	if !errors.As(err, &target) {
		t.Fatalf("errors.As(%v) did not find Error", err)
	}
	if target.Kind != KindBadParameter {
		t.Errorf("Kind = %q, want %q", target.Kind, KindBadParameter)
	}
	if target.Message != "invalid name" {
		t.Errorf("Message = %q, want invalid name", target.Message)
	}
}

func TestWrapPreservesCause(t *testing.T) {
	cause := errors.New("duplicate key")
	err := Wrap(KindAlreadyExists, cause, "user already exists")

	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, cause) = false", err)
	}
	var target *Error
	if !errors.As(err, &target) {
		t.Fatalf("errors.As(%v) did not find Error", err)
	}
	if target.Kind != KindAlreadyExists {
		t.Errorf("Kind = %q, want %q", target.Kind, KindAlreadyExists)
	}
}
