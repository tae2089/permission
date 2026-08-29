package input

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeFormDecodesBodyWithoutQueryValues(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "canonical content type",
			contentType: "application/x-www-form-urlencoded",
			body:        "name=Alice&tag=go&tag=api",
		},
		{
			name:        "content type with parameters",
			contentType: "application/x-www-form-urlencoded; charset=utf-8",
			body:        "name=Alice&tag=go&tag=api",
		},
		{
			name:        "empty form",
			contentType: "application/x-www-form-urlencoded",
			body:        "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"/?name=Mallory&query=only",
				strings.NewReader(test.body),
			)
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()

			values, err := DecodeForm(response, request)
			if err != nil {
				t.Fatalf("DecodeForm() error = %v", err)
			}
			if test.body == "" {
				if len(values) != 0 {
					t.Fatalf("DecodeForm() = %v, want empty values", values)
				}
				return
			}
			if got := values.Get("name"); got != "Alice" {
				t.Errorf("name = %q, want %q", got, "Alice")
			}
			if got := values["tag"]; len(got) != 2 || got[0] != "go" || got[1] != "api" {
				t.Errorf("tag = %v, want [go api]", got)
			}
			if values.Has("query") {
				t.Errorf("query value leaked into body values: %v", values["query"])
			}
		})
	}
}

func TestDecodeFormRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "missing content type",
			contentType: "",
			body:        "name=Alice",
		},
		{
			name:        "wrong content type",
			contentType: "text/plain",
			body:        "name=Alice",
		},
		{
			name:        "malformed encoding",
			contentType: "application/x-www-form-urlencoded",
			body:        "name=%zz",
		},
		{
			name:        "oversized body",
			contentType: "application/x-www-form-urlencoded",
			body:        "value=" + strings.Repeat("a", int(MaxFormBodyBytes)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()

			_, err := DecodeForm(response, request)
			if err == nil {
				t.Fatal("DecodeForm() error = nil, want bad parameter")
			}
			assertBadParameter(t, err)
		})
	}
}
