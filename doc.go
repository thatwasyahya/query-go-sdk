// Package querygo is a small, dependency-free SDK for the HTTP QUERY method
// (RFC 10008).
//
// QUERY is a safe, idempotent request method that carries a request body
// describing a query to be evaluated against the target resource. Unlike GET,
// the query parameters travel in the body rather than the URI, and unlike
// POST, the request is safe and may be retried.
//
// # Sending a query
//
//	client := querygo.NewClient(nil)
//	resp, err := client.QueryJSON(ctx, "https://example.org/search",
//		map[string]any{"q": "golang"}, querygo.MediaTypeJSON, nil)
//	if err != nil {
//		return err
//	}
//	defer resp.Body.Close()
//
// The typed helpers QueryString, QueryBytes, QueryForm and QueryJSON build the
// request body for common content types. QueryJSONInto additionally decodes a
// JSON response:
//
//	var result SearchResult
//	err := client.QueryJSONInto(ctx, url, request, &result, nil)
//
// # Response handling
//
// CheckStatus turns a non-2xx response into a *StatusError. ReadBody,
// DecodeText and DecodeJSON validate the status and consume the body.
//
// # Result location
//
// A QUERY response may advertise two distinct URIs (RFC 10008 Sections 2.3 and
// 2.4): Content-Location identifies the stored result of the query just run
// (FetchResult follows it), while Location identifies the equivalent resource
// whose GET re-runs the query for a current result (FetchEquivalent follows
// it).
//
//	resp, _ := client.QueryJSON(ctx, url, request, querygo.MediaTypeJSON, nil)
//	result, err := client.FetchResult(ctx, resp, nil)
//
// # Discovery
//
// Options issues an OPTIONS request and reports QUERY support based on the
// Allow and Accept-Query header fields.
//
// # Retries
//
// Because QUERY is safe and idempotent, DoWithRetry can transparently retry
// transient failures, replaying the request body via Request.GetBody (which
// the Body constructors populate).
package querygo
