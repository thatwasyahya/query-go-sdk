package querygo

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// ErrNoQueryResult is returned when a QUERY response advertises no result
// resource (no Content-Location header field).
var ErrNoQueryResult = errors.New("querygo: response has no query result location")

// ErrNoEquivalentResource is returned when a QUERY response advertises no
// equivalent resource (no Location header field).
var ErrNoEquivalentResource = errors.New("querygo: response has no equivalent resource location")

// FetchResult performs a GET on the resource holding the result of the query
// that was just performed, as advertised by the Content-Location response field
// (RFC 10008 Section 2.3). The URI is resolved relative to the original request
// URL. It returns ErrNoQueryResult when the response carries no Content-Location.
//
// The caller is responsible for closing the returned response body.
func (c *Client) FetchResult(ctx context.Context, resp *http.Response, header http.Header) (*http.Response, error) {
	if resp == nil {
		return nil, ErrNoQueryResult
	}

	ref := ParseResponseInfo(resp).ResultURI()
	if ref == "" {
		return nil, ErrNoQueryResult
	}

	return c.getRelative(ctx, resp, ref, header)
}

// FetchEquivalent performs a GET on the equivalent resource for the query, as
// advertised by the Location response field (RFC 10008 Section 2.4). Unlike
// FetchResult, this re-runs the query without resending its content and yields
// the current result. The URI is resolved relative to the original request URL.
// It returns ErrNoEquivalentResource when the response carries no Location.
//
// The caller is responsible for closing the returned response body.
func (c *Client) FetchEquivalent(ctx context.Context, resp *http.Response, header http.Header) (*http.Response, error) {
	if resp == nil {
		return nil, ErrNoEquivalentResource
	}

	ref := ParseResponseInfo(resp).EquivalentResourceURI()
	if ref == "" {
		return nil, ErrNoEquivalentResource
	}

	return c.getRelative(ctx, resp, ref, header)
}

// getRelative issues a GET for ref resolved against the original request URL of
// resp.
func (c *Client) getRelative(ctx context.Context, resp *http.Response, ref string, header http.Header) (*http.Response, error) {
	var base *url.URL
	if resp.Request != nil {
		base = resp.Request.URL
	}

	target, err := resolveReference(base, ref)
	if err != nil {
		return nil, err
	}

	request, err := c.newHTTPRequest(ctx, http.MethodGet, target, nil, header)
	if err != nil {
		return nil, err
	}

	return c.httpClient().Do(request)
}

// resolveReference resolves ref against base. When base is nil, ref is returned
// as an absolute reference.
func resolveReference(base *url.URL, ref string) (string, error) {
	parsed, err := url.Parse(ref)
	if err != nil {
		return "", err
	}

	if base != nil {
		return base.ResolveReference(parsed).String(), nil
	}

	return parsed.String(), nil
}
