package querygo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
)

// DefaultMaxRedirects is the redirect limit used when Client.MaxRedirects is
// zero or negative.
const DefaultMaxRedirects = 10

// ErrTooManyRedirects is returned when a QUERY request exceeds the redirect
// limit (see Client.MaxRedirects).
var ErrTooManyRedirects = errors.New("querygo: too many redirects")

// ErrRedirectBodyNotReplayable is returned when following a redirect requires
// resending the query body but the request has no GetBody to replay it. The
// Body constructors populate GetBody automatically.
var ErrRedirectBodyNotReplayable = errors.New("querygo: cannot follow redirect without a replayable body")

func (c *Client) maxRedirects() int {
	if c != nil && c.MaxRedirects > 0 {
		return c.MaxRedirects
	}

	return DefaultMaxRedirects
}

// follow performs request and follows redirects per RFC 10008 Section 2.5: for
// 301, 302, 307 and 308 the QUERY method and body are preserved (the standard
// library would downgrade 301/302 to GET, which the RFC explicitly forbids for
// QUERY); a 303 switches to GET. orig supplies the replayable body for QUERY
// hops.
func (c *Client) follow(ctx context.Context, request *http.Request, orig Request) (*http.Response, error) {
	base := c.httpClient()

	// Copy the client so redirects are surfaced to this loop instead of being
	// followed by the standard library. The copy shares the Transport and Jar.
	redirectClient := *base
	redirectClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	limit := c.maxRedirects()

	for redirects := 0; ; redirects++ {
		resp, err := redirectClient.Do(request)
		if err != nil {
			return nil, err
		}

		if !isRedirectStatus(resp.StatusCode) || resp.Header.Get(HeaderLocation) == "" {
			return resp, nil
		}

		if redirects >= limit {
			drainClose(resp)
			return nil, ErrTooManyRedirects
		}

		next, err := c.nextRedirectRequest(ctx, request, resp, orig)
		drainClose(resp)
		if err != nil {
			return nil, err
		}

		request = next
	}
}

func isRedirectStatus(code int) bool {
	switch code {
	case http.StatusMovedPermanently, // 301
		http.StatusFound,             // 302
		http.StatusSeeOther,          // 303
		http.StatusTemporaryRedirect, // 307
		http.StatusPermanentRedirect: // 308
		return true
	default:
		return false
	}
}

// nextRedirectRequest builds the follow-up request for a redirect response,
// resolving Location against the current URL and applying the RFC 10008
// Section 2.5 method rules.
func (c *Client) nextRedirectRequest(ctx context.Context, current *http.Request, resp *http.Response, orig Request) (*http.Request, error) {
	parsed, err := url.Parse(resp.Header.Get(HeaderLocation))
	if err != nil {
		return nil, err
	}
	target := current.URL.ResolveReference(parsed)

	method := current.Method
	var body io.Reader

	toGet := resp.StatusCode == http.StatusSeeOther
	switch {
	case toGet:
		method = http.MethodGet
	case current.Method == MethodQuery:
		// 301, 302, 307, 308: keep QUERY and resend the query content.
		if orig.GetBody == nil {
			return nil, ErrRedirectBodyNotReplayable
		}

		reader, err := orig.GetBody()
		if err != nil {
			return nil, err
		}
		body = reader
	}

	next, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}

	// Carry the headers forward, then adjust for the (possibly dropped) body and
	// for a change of host.
	next.Header = current.Header.Clone()

	if toGet {
		next.Header.Del(HeaderContentType)
		next.Header.Del("Content-Length")
	}

	if !sameHost(current.URL, target) {
		// Match the standard library: drop sensitive headers on host change.
		next.Header.Del("Authorization")
		next.Header.Del("Www-Authenticate")
		next.Header.Del("Cookie")
		next.Header.Del("Cookie2")
	}

	if body != nil && orig.GetBody != nil {
		getBody := orig.GetBody
		next.GetBody = func() (io.ReadCloser, error) {
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

	return next, nil
}

func sameHost(a, b *url.URL) bool {
	return a.Hostname() == b.Hostname()
}
