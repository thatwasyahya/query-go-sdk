package querygo

import (
	"context"
	"net/http"
)

// Discovery describes a server's advertised support for the QUERY method, as
// reported by an OPTIONS response (or any response carrying the relevant
// header fields).
type Discovery struct {
	// SupportsQuery is true when the response advertises the QUERY method via
	// the Allow header field or carries an Accept-Query header field.
	SupportsQuery bool

	// Allow lists the methods advertised in the Allow header field.
	Allow []string

	// AcceptQuery lists the query media types advertised in the Accept-Query
	// header field.
	AcceptQuery []string

	// Header is the raw response header.
	Header http.Header
}

// AcceptsMediaType reports whether mediaType is covered by the advertised
// Accept-Query media ranges, honoring the "*/*" and "type/*" wildcards.
func (d Discovery) AcceptsMediaType(mediaType string) bool {
	for _, item := range d.AcceptQuery {
		if matchMediaRange(item, mediaType) {
			return true
		}
	}

	return false
}

// DiscoveryFromResponse interprets a response's header fields as QUERY support
// metadata.
func DiscoveryFromResponse(resp *http.Response) Discovery {
	allow := parseHeaderList(resp.Header.Values(HeaderAllow))
	acceptQuery := parseAcceptQuery(resp.Header.Values(HeaderAcceptQuery))

	supports := len(acceptQuery) > 0
	for _, method := range allow {
		if method == MethodQuery {
			supports = true
			break
		}
	}

	return Discovery{
		SupportsQuery: supports,
		Allow:         allow,
		AcceptQuery:   acceptQuery,
		Header:        resp.Header,
	}
}

// Options issues an HTTP OPTIONS request to targetURL and interprets the
// response header fields to determine QUERY support. The response body is
// drained and closed.
func (c *Client) Options(ctx context.Context, targetURL string, header http.Header) (Discovery, error) {
	request, err := c.newHTTPRequest(ctx, http.MethodOptions, targetURL, nil, header)
	if err != nil {
		return Discovery{}, err
	}

	resp, err := c.httpClient().Do(request)
	if err != nil {
		return Discovery{}, err
	}
	defer drainClose(resp)

	return DiscoveryFromResponse(resp), nil
}
