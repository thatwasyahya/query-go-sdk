package querygo

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Header field names used by the HTTP QUERY method.
const (
	HeaderAcceptQuery     = "Accept-Query"
	HeaderAllow           = "Allow"
	HeaderContentType     = "Content-Type"
	HeaderContentLocation = "Content-Location"
	HeaderLocation        = "Location"
	HeaderAccept          = "Accept"
)

// Common media types used for QUERY request bodies.
const (
	MediaTypeJSON = "application/json"
	MediaTypeForm = "application/x-www-form-urlencoded"
)

// Body pairs a request content reader with its media type. When GetReader is
// set the body can be produced again, enabling redirect and retry replay.
type Body struct {
	Reader      io.Reader
	GetReader   func() (io.Reader, error)
	ContentType string
}

// ResponseInfo captures the QUERY-relevant metadata of a response.
type ResponseInfo struct {
	StatusCode      int
	ContentLocation string
	Location        string
	AcceptQuery     []string
	Allow           []string
	Header          http.Header
}

// NewBodyString builds a Body from a string with the given content type.
func NewBodyString(contentType, value string) Body {
	return Body{
		Reader:      strings.NewReader(value),
		GetReader:   func() (io.Reader, error) { return strings.NewReader(value), nil },
		ContentType: contentType,
	}
}

// NewBodyBytes builds a Body from a byte slice with the given content type.
func NewBodyBytes(contentType string, value []byte) Body {
	return Body{
		Reader:      bytes.NewReader(value),
		GetReader:   func() (io.Reader, error) { return bytes.NewReader(value), nil },
		ContentType: contentType,
	}
}

// NewBodyForm builds an application/x-www-form-urlencoded Body from form
// values.
func NewBodyForm(values url.Values) Body {
	return NewBodyString(MediaTypeForm, values.Encode())
}

// NewBodyJSON marshals value to JSON and builds an application/json Body.
func NewBodyJSON(value any) (Body, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return Body{}, err
	}

	return NewBodyBytes(MediaTypeJSON, encoded), nil
}

// RequestFromBody converts a Body into a Request, propagating GetReader so the
// request can be replayed.
func RequestFromBody(targetURL string, body Body, accept string, header http.Header) Request {
	req := Request{
		URL:         targetURL,
		Body:        body.Reader,
		ContentType: body.ContentType,
		Accept:      accept,
		Header:      header,
	}

	if body.GetReader != nil {
		req.GetBody = body.GetReader
	}

	return req
}

// ParseResponseInfo extracts QUERY-relevant metadata from a response.
func ParseResponseInfo(resp *http.Response) ResponseInfo {
	info := ResponseInfo{StatusCode: resp.StatusCode, Header: resp.Header}
	info.ContentLocation = resp.Header.Get(HeaderContentLocation)
	info.Location = resp.Header.Get(HeaderLocation)
	info.AcceptQuery = parseHeaderList(resp.Header.Values(HeaderAcceptQuery))
	info.Allow = parseHeaderList(resp.Header.Values(HeaderAllow))
	return info
}

// SupportsQuery reports whether a response advertises support for the QUERY
// method, either by listing QUERY in Allow or by carrying an Accept-Query
// header field.
func SupportsQuery(resp *http.Response) bool {
	for _, method := range parseHeaderList(resp.Header.Values(HeaderAllow)) {
		if method == MethodQuery {
			return true
		}
	}

	return len(parseHeaderList(resp.Header.Values(HeaderAcceptQuery))) > 0
}

// QueryURI returns the URI at which the query result can be retrieved with a
// GET request, preferring Location over Content-Location.
func (info ResponseInfo) QueryURI() string {
	if info.Location != "" {
		return info.Location
	}

	return info.ContentLocation
}

// QueryRequest sends a QUERY request built from a Body.
func (c *Client) QueryRequest(ctx context.Context, targetURL string, body Body, accept string, header http.Header) (*http.Response, error) {
	return c.Do(ctx, RequestFromBody(targetURL, body, accept, header))
}

// QueryString sends a QUERY request with a string body.
func (c *Client) QueryString(ctx context.Context, targetURL, contentType, value, accept string, header http.Header) (*http.Response, error) {
	return c.QueryRequest(ctx, targetURL, NewBodyString(contentType, value), accept, header)
}

// QueryBytes sends a QUERY request with a byte-slice body.
func (c *Client) QueryBytes(ctx context.Context, targetURL, contentType string, value []byte, accept string, header http.Header) (*http.Response, error) {
	return c.QueryRequest(ctx, targetURL, NewBodyBytes(contentType, value), accept, header)
}

// QueryForm sends a QUERY request with a form-encoded body.
func (c *Client) QueryForm(ctx context.Context, targetURL string, values url.Values, accept string, header http.Header) (*http.Response, error) {
	return c.QueryRequest(ctx, targetURL, NewBodyForm(values), accept, header)
}

// QueryJSON sends a QUERY request whose body is value marshaled to JSON.
func (c *Client) QueryJSON(ctx context.Context, targetURL string, value any, accept string, header http.Header) (*http.Response, error) {
	body, err := NewBodyJSON(value)
	if err != nil {
		return nil, err
	}

	return c.QueryRequest(ctx, targetURL, body, accept, header)
}

// parseHeaderList splits comma-separated header field values into trimmed,
// non-empty items.
func parseHeaderList(values []string) []string {
	items := make([]string, 0)
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				items = append(items, trimmed)
			}
		}
	}

	return items
}
