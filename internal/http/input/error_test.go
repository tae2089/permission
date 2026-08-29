package input

import (
	"errors"
	"testing"

	"github.com/tae2089/go-template/internal/apperr"
)

func assertBadParameter(t testing.TB, err error) {
	t.Helper()

	var target *apperr.Error
	if !errors.As(err, &target) || target.Kind != apperr.KindBadParameter {
		t.Fatalf("error = %v, want bad parameter", err)
	}
}
