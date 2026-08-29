package input

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeMultipartDecodesFieldsAndFiles(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("name", "Alice"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.WriteField("tag", "go"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.WriteField("tag", "api"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	file, err := writer.CreateFormFile("document", "hello.txt")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := file.Write([]byte("hello")); err != nil {
		t.Fatalf("file.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	request := httptest.NewRequest("POST", "/", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	form, err := DecodeMultipart(response, request)
	if err != nil {
		t.Fatalf("DecodeMultipart() error = %v", err)
	}
	t.Cleanup(func() {
		if err := form.RemoveAll(); err != nil {
			t.Errorf("form.RemoveAll() error = %v", err)
		}
	})

	if got := form.Value["name"]; len(got) != 1 || got[0] != "Alice" {
		t.Errorf("name = %v, want [Alice]", got)
	}
	if got := form.Value["tag"]; len(got) != 2 || got[0] != "go" || got[1] != "api" {
		t.Errorf("tag = %v, want [go api]", got)
	}

	files := form.File["document"]
	if len(files) != 1 {
		t.Fatalf("document files = %d, want 1", len(files))
	}
	if files[0].Filename != "hello.txt" {
		t.Errorf("Filename = %q, want %q", files[0].Filename, "hello.txt")
	}
	opened, err := files[0].Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	content, readErr := io.ReadAll(opened)
	closeErr := opened.Close()
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	if string(content) != "hello" {
		t.Errorf("file content = %q, want %q", content, "hello")
	}
}

func TestDecodeMultipartDecodesEmptyForm(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	request := httptest.NewRequest("POST", "/", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	form, err := DecodeMultipart(response, request)
	if err != nil {
		t.Fatalf("DecodeMultipart() error = %v", err)
	}
	t.Cleanup(func() {
		if err := form.RemoveAll(); err != nil {
			t.Errorf("form.RemoveAll() error = %v", err)
		}
	})
	if len(form.Value) != 0 {
		t.Errorf("form.Value = %v, want empty values", form.Value)
	}
	if len(form.File) != 0 {
		t.Errorf("form.File = %v, want empty files", form.File)
	}
}

func TestDecodeMultipartRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "missing content type",
			contentType: "",
			body:        "",
		},
		{
			name:        "wrong content type",
			contentType: "application/octet-stream",
			body:        "",
		},
		{
			name:        "missing boundary",
			contentType: "multipart/form-data",
			body:        "",
		},
		{
			name:        "malformed body",
			contentType: "multipart/form-data; boundary=test-boundary",
			body:        "not a multipart body",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()

			_, err := DecodeMultipart(response, request)
			if err == nil {
				t.Fatal("DecodeMultipart() error = nil, want bad parameter")
			}
			assertBadParameter(t, err)
		})
	}
}

func TestDecodeMultipartRejectsOversizedBody(t *testing.T) {
	const boundary = "test-boundary"
	prefix := "--" + boundary + "\r\n" +
		`Content-Disposition: form-data; name="document"; filename="large.txt"` + "\r\n" +
		"Content-Type: application/octet-stream\r\n\r\n"
	body := io.MultiReader(
		strings.NewReader(prefix),
		io.LimitReader(repeatedByteReader('a'), MaxMultipartBodyBytes+1),
	)
	request := httptest.NewRequest("POST", "/", body)
	request.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	response := httptest.NewRecorder()

	_, err := DecodeMultipart(response, request)
	if err == nil {
		t.Fatal("DecodeMultipart() error = nil, want bad parameter")
	}
	assertBadParameter(t, err)
}

type repeatedByteReader byte

func (r repeatedByteReader) Read(buffer []byte) (int, error) {
	for i := range buffer {
		buffer[i] = byte(r)
	}
	return len(buffer), nil
}
