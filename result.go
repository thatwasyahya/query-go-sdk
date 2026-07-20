package querygo

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// ErrNoQueryResult is returned when a QUERY response advertises no result URI
// (neither Location nor Content-Location).
var ErrNoQueryResult = errors.New("querygo: response has no query result location")

// FetchResult performs a GET on the query-result URI advertised by a QUERY
// response (Location, falling back to Content-Location). The URI is resolved
// relative to the original request URL. This models the QUERY pattern in which
// the server materializes the result at a cacheable, GET-retrievable URI.
//
// The caller is responsible for closing the returned response body.
func (c *Client) FetchResult(ctx context.Context, resp *http.Response, header http.Header) (*http.Response, error) {
	if resp == nil {
		return nil, ErrNoQueryResult
	}

	ref := ParseResponseInfo(resp).QueryURI()
	if ref == "" {
		return nil, ErrNoQueryResult
	}

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
