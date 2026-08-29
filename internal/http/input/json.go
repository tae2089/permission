package input

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"

	"github.com/tae2089/go-template/internal/apperr"
)

const MaxJSONBodyBytes int64 = 64 << 10

func DecodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return apperr.New(apperr.KindBadParameter, "content type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return apperr.Wrap(apperr.KindBadParameter, err, "invalid request body")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return apperr.New(apperr.KindBadParameter, "request body must contain one JSON document")
		}
		return apperr.Wrap(apperr.KindBadParameter, err, "invalid request body")
	}

	return nil
}
