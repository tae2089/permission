package input

import (
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/tae2089/go-template/internal/apperr"
)

const MaxFormBodyBytes int64 = 64 << 10

func DecodeForm(w http.ResponseWriter, r *http.Request) (url.Values, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return nil, apperr.New(apperr.KindBadParameter, "content type must be application/x-www-form-urlencoded")
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxFormBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, apperr.Wrap(apperr.KindBadParameter, err, "invalid request body")
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, apperr.Wrap(apperr.KindBadParameter, err, "invalid request body")
	}

	return values, nil
}
