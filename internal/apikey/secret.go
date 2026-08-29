package apikey

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/tae2089/trace/v3"
)

const secretBytes = 32

func NewSecret() (string, error) {
	bytes := make([]byte, secretBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", trace.Wrap(err, "read API key random bytes")
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
