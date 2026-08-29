package input

import (
	"mime"
	"mime/multipart"
	"net/http"

	"github.com/tae2089/go-template/internal/apperr"
)

const (
	MaxMultipartBodyBytes   int64 = 32 << 20
	MaxMultipartMemoryBytes int64 = 8 << 20
)

func DecodeMultipart(w http.ResponseWriter, r *http.Request) (*multipart.Form, error) {
	mediaType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		return nil, apperr.New(apperr.KindBadParameter, "content type must be multipart/form-data")
	}

	boundary := parameters["boundary"]
	if boundary == "" {
		return nil, apperr.New(apperr.KindBadParameter, "multipart boundary is required")
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxMultipartBodyBytes)
	form, err := multipart.NewReader(r.Body, boundary).ReadForm(MaxMultipartMemoryBytes)
	if err != nil {
		return nil, apperr.Wrap(apperr.KindBadParameter, err, "invalid request body")
	}

	return form, nil
}
