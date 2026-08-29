package input

import (
	"encoding/xml"
	"io"
	"mime"
	"net/http"

	"github.com/tae2089/go-template/internal/apperr"
)

const MaxXMLBodyBytes int64 = 64 << 10

func DecodeXML(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/xml" {
		return apperr.New(apperr.KindBadParameter, "content type must be application/xml")
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxXMLBodyBytes)
	decoder := xml.NewDecoder(r.Body)

	if err := decoder.Decode(target); err != nil {
		return apperr.Wrap(apperr.KindBadParameter, err, "invalid request body")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return apperr.New(apperr.KindBadParameter, "request body must contain one XML document")
		}
		return apperr.Wrap(apperr.KindBadParameter, err, "invalid request body")
	}

	return nil
}
