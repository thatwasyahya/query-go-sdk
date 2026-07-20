package querygo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

// ErrBodyNotReplayable is returned when a Body without a GetReader is asked to
// produce its bytes. The Body constructors (NewBodyString, NewBodyBytes,
// NewBodyForm, NewBodyJSON) always set GetReader.
var ErrBodyNotReplayable = errors.New("querygo: body cannot be read without a GetReader")

// Bytes materializes the body content. It requires GetReader to be set;
// otherwise it returns ErrBodyNotReplayable.
func (b Body) Bytes() ([]byte, error) {
	if b.GetReader == nil {
		return nil, ErrBodyNotReplayable
	}

	reader, err := b.GetReader()
	if err != nil {
		return nil, err
	}

	return io.ReadAll(reader)
}

// CacheKey computes a stable cache key for a QUERY request. Unlike GET, a QUERY
// cache entry must be keyed on the request content in addition to the method
// and URI, because the query travels in the body. The returned value is the
// hex-encoded SHA-256 of those inputs.
func CacheKey(targetURL, contentType string, body []byte) string {
	sum := sha256.New()
	writeField(sum, MethodQuery)
	writeField(sum, targetURL)
	writeField(sum, contentType)
	sum.Write(body)
	return hex.EncodeToString(sum.Sum(nil))
}

// CacheKeyForBody is CacheKey using a Body, materializing its content.
func CacheKeyForBody(targetURL string, body Body) (string, error) {
	data, err := body.Bytes()
	if err != nil {
		return "", err
	}

	return CacheKey(targetURL, body.ContentType, data), nil
}

func writeField(w io.Writer, value string) {
	_, _ = io.WriteString(w, value)
	_, _ = w.Write([]byte{0})
}
