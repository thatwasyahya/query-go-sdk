package querygo

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// MethodQuery is the HTTP QUERY method token defined by RFC 10008. QUERY is a
// safe, idempotent request method that carries a request body describing a
// query to be evaluated against the target resource.
const MethodQuery = "QUERY"

// Client sends HTTP QUERY requests. The zero value is usable and falls back to
// http.DefaultClient, but NewClient is the preferred constructor.
type Client struct {
	// HTTPClient performs the underlying HTTP round trips. When nil,
	// http.DefaultClient is used.
	HTTPClient *http.Client

	// BaseURL, when set, is used to resolve relative request URLs. Absolute
	// request URLs are used as-is.
	BaseURL string

	// DefaultHeader holds header fields applied to every request unless the
	// request already carries a value for that field.
	DefaultHeader http.Header

	// UserAgent, when set, is applied as the User-Agent header field unless a
	// request already provides one.
	UserAgent string

	// MaxRedirects caps the number of redirects followed for a QUERY request.
	// When zero or negative, DefaultMaxRedirects is used. Redirects are handled
	// per RFC 10008 Section 2.5: 301, 302, 307 and 308 re-issue QUERY (replaying
	// the body), while 303 switches to GET.
	MaxRedirects int
}

// Request describes a single HTTP QUERY request.
type Request struct {
	// URL is the target URL. It may be relative when the client has a BaseURL.
	URL string

	// Body is the request content (the query). It may be nil for an empty
	// query.
	Body io.Reader

	// GetBody, when set, returns a fresh Body reader. It enables the request
	// to be replayed for redirects and retries. Body constructors such as
	// NewBodyString populate it automatically via RequestFromBody.
	GetBody func() (io.Reader, error)

	// ContentType sets the Content-Type header field describing Body.
	ContentType string

	// Accept sets the Accept header field advertising acceptable response
	// media types.
	Accept string

	// Header holds additional header fields to add to the request.
	Header http.Header
}

// Option customizes a Client during construction.
type Option func(*Client)

// WithBaseURL sets the base URL used to resolve relative request URLs.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.BaseURL = baseURL }
}

// WithUserAgent sets the default User-Agent header field.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.UserAgent = userAgent }
}

// WithMaxRedirects sets the maximum number of redirects followed for a QUERY
// request. A value of zero or less selects DefaultMaxRedirects.
func WithMaxRedirects(n int) Option {
	return func(c *Client) { c.MaxRedirects = n }
}

// WithDefaultHeader sets header fields applied to every request unless already
// present. The provided header is cloned.
func WithDefaultHeader(header http.Header) Option {
	return func(c *Client) { c.DefaultHeader = header.Clone() }
}

// WithHeader adds a single default header field, allocating the header map if
// needed.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		if c.DefaultHeader == nil {
			c.DefaultHeader = http.Header{}
		}
		c.DefaultHeader.Add(key, value)
	}
}

// NewClient returns a Client using the provided HTTP client (or
// http.DefaultClient when nil) configured with the given options.
func NewClient(httpClient *http.Client, opts ...Option) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	client := &Client{HTTPClient: httpClient}
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// NewRequest builds an *http.Request that uses the QUERY method for the given
// Request. When GetBody is set it is wired onto the request so the standard
// library can replay the body across redirects.
func NewRequest(ctx context.Context, req Request) (*http.Request, error) {
	body := req.Body
	if body == nil && req.GetBody != nil {
		reader, err := req.GetBody()
		if err != nil {
			return nil, err
		}

		body = reader
	}

	request, err := http.NewRequestWithContext(ctx, MethodQuery, req.URL, body)
	if err != nil {
		return nil, err
	}

	if req.GetBody != nil {
		getBody := req.GetBody
		request.GetBody = func() (io.ReadCloser, error) {
			reader, err := getBody()
			if err != nil {
				return nil, err
			}

			if rc, ok := reader.(io.ReadCloser); ok {
				return rc, nil
			}

			return io.NopCloser(reader), nil
		}
	}

	if req.ContentType != "" {
		request.Header.Set(HeaderContentType, req.ContentType)
	}

	if req.Accept != "" {
		request.Header.Set(HeaderAccept, req.Accept)
	}

	for key, values := range req.Header {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	return request, nil
}

// Do sends a QUERY request and returns the raw HTTP response. The caller is
// responsible for closing the response body.
func (c *Client) Do(ctx context.Context, req Request) (*http.Response, error) {
	resolved, err := c.resolveURL(req.URL)
	if err != nil {
		return nil, err
	}
	req.URL = resolved

	request, err := NewRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	c.applyDefaults(request)

	return c.follow(ctx, request, req)
}

// Query is a convenience wrapper that sends a QUERY request with a body and
// Content-Type. Use Do or the typed Query* helpers for more control.
func (c *Client) Query(ctx context.Context, targetURL string, body io.Reader, contentType string) (*http.Response, error) {
	return c.Do(ctx, Request{
		URL:         targetURL,
		Body:        body,
		ContentType: contentType,
	})
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}

	return http.DefaultClient
}

// resolveURL resolves ref against the client's BaseURL. Absolute references
// are returned unchanged.
func (c *Client) resolveURL(ref string) (string, error) {
	if c == nil || c.BaseURL == "" {
		return ref, nil
	}

	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(ref)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(parsed).String(), nil
}

// applyDefaults adds the client's default header fields and User-Agent to the
// request without overwriting values it already carries.
func (c *Client) applyDefaults(request *http.Request) {
	if c == nil {
		return
	}

	for key, values := range c.DefaultHeader {
		if request.Header.Get(key) != "" {
			continue
		}

		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	if c.UserAgent != "" && request.Header.Get("User-Agent") == "" {
		request.Header.Set("User-Agent", c.UserAgent)
	}
}

// newHTTPRequest builds a request for a non-QUERY method (used by discovery
// and result fetching) applying base-URL resolution and default headers.
func (c *Client) newHTTPRequest(ctx context.Context, method, ref string, body io.Reader, header http.Header) (*http.Request, error) {
	target, err := c.resolveURL(ref)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}

	for key, values := range header {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	c.applyDefaults(request)

	return request, nil
}
