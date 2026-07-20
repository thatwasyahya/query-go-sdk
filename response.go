package querygo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// MaxErrorBodySnippet bounds how many bytes of a response body are captured in
// a StatusError message.
const MaxErrorBodySnippet = 4 << 10 // 4 KiB

// StatusError is returned when a QUERY (or result) response carries a status
// code outside the 2xx success range.
type StatusError struct {
	StatusCode int
	Status     string
	Method     string
	URL        string
	Body       []byte
	Header     http.Header
}

// Error implements the error interface.
func (e *StatusError) Error() string {
	prefix := "querygo: unexpected status " + e.Status
	if e.Method != "" && e.URL != "" {
		prefix = fmt.Sprintf("querygo: %s %s: unexpected status %s", e.Method, e.URL, e.Status)
	}

	if len(e.Body) == 0 {
		return prefix
	}

	return prefix + ": " + string(e.Body)
}

// AsStatusError reports whether err wraps a *StatusError and returns it.
func AsStatusError(err error) (*StatusError, bool) {
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr, true
	}

	return nil, false
}

// CheckStatus returns a *StatusError when resp carries a non-2xx status code,
// otherwise nil. On error it reads a bounded snippet of the body for the error
// message but does not close the body; the caller remains responsible for
// closing resp.Body.
func CheckStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	var snippet []byte
	if resp.Body != nil {
		snippet, _ = io.ReadAll(io.LimitReader(resp.Body, MaxErrorBodySnippet))
	}

	method, target := requestInfo(resp)

	return &StatusError{
		StatusCode: resp.StatusCode,
		Status:     statusText(resp),
		Method:     method,
		URL:        target,
		Body:       snippet,
		Header:     resp.Header,
	}
}

// ReadBody validates the response status and returns the full response body.
// The body is always closed.
func ReadBody(resp *http.Response) ([]byte, error) {
	defer drainClose(resp)

	if err := CheckStatus(resp); err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

// DecodeText validates the response status and returns the body as a string.
// The body is always closed.
func DecodeText(resp *http.Response) (string, error) {
	data, err := ReadBody(resp)
	return string(data), err
}

// DecodeJSON validates the response status and decodes the JSON body into v.
// The body is always closed. A nil v drains and validates only.
func DecodeJSON(resp *http.Response, v any) error {
	defer drainClose(resp)

	if err := CheckStatus(resp); err != nil {
		return err
	}

	if v == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// QueryJSONInto sends a QUERY request with a JSON body and decodes the JSON
// response into result. It is a convenience over QueryJSON + DecodeJSON.
func (c *Client) QueryJSONInto(ctx context.Context, targetURL string, body any, result any, header http.Header) error {
	resp, err := c.QueryJSON(ctx, targetURL, body, MediaTypeJSON, header)
	if err != nil {
		return err
	}

	return DecodeJSON(resp, result)
}

func requestInfo(resp *http.Response) (method, target string) {
	if resp == nil || resp.Request == nil {
		return "", ""
	}

	method = resp.Request.Method
	if resp.Request.URL != nil {
		target = resp.Request.URL.String()
	}

	return method, target
}

func statusText(resp *http.Response) string {
	if resp.Status != "" {
		return resp.Status
	}

	if text := http.StatusText(resp.StatusCode); text != "" {
		return strconv.Itoa(resp.StatusCode) + " " + text
	}

	return strconv.Itoa(resp.StatusCode)
}

func drainClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
