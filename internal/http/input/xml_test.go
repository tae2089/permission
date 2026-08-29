package input

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeXMLDecodesValidDocument(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{
			name:        "canonical content type",
			contentType: "application/xml",
		},
		{
			name:        "content type with parameters",
			contentType: "application/xml; charset=utf-8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(
				`<request><name>Alice</name><email>alice@example.com</email></request>`,
			))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			var target struct {
				Name  string `xml:"name"`
				Email string `xml:"email"`
			}

			if err := DecodeXML(response, request, &target); err != nil {
				t.Fatalf("DecodeXML() error = %v", err)
			}
			if target.Name != "Alice" {
				t.Errorf("Name = %q, want %q", target.Name, "Alice")
			}
			if target.Email != "alice@example.com" {
				t.Errorf("Email = %q, want %q", target.Email, "alice@example.com")
			}
		})
	}
}

func TestDecodeXMLRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "missing content type",
			contentType: "",
			body:        `<request/>`,
		},
		{
			name:        "wrong content type",
			contentType: "text/xml",
			body:        `<request/>`,
		},
		{
			name:        "empty body",
			contentType: "application/xml",
			body:        "",
		},
		{
			name:        "malformed document",
			contentType: "application/xml",
			body:        `<request>`,
		},
		{
			name:        "multiple documents",
			contentType: "application/xml",
			body:        `<request/><request/>`,
		},
		{
			name:        "oversized body",
			contentType: "application/xml",
			body:        `<request><value>` + strings.Repeat("a", int(MaxXMLBodyBytes)) + `</value></request>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()

			err := DecodeXML(response, request, &struct{}{})
			if err == nil {
				t.Fatal("DecodeXML() error = nil, want bad parameter")
			}
			assertBadParameter(t, err)
		})
	}
}
