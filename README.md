# querygo

A small, dependency-free Go SDK for the **HTTP `QUERY` method**
([`draft-ietf-httpbis-safe-method-w-body`](https://datatracker.ietf.org/doc/draft-ietf-httpbis-safe-method-w-body/)).

`QUERY` is a **safe** and **idempotent** request method that carries a request
body describing a query to run against the target resource. It combines the
best of `GET` (safe, cacheable, retryable) and `POST` (arbitrary request body):
the query travels in the body instead of the URI.

```sh
go get github.com/thatwasyahya/query-go-sdk
```

```go
import querygo "github.com/thatwasyahya/query-go-sdk"
```

## Features

| Area | API |
| --- | --- |
| Send a query | `Client.Query`, `Client.QueryString`, `Client.QueryBytes`, `Client.QueryForm`, `Client.QueryJSON` |
| Build request bodies | `NewBodyString`, `NewBodyBytes`, `NewBodyForm`, `NewBodyJSON` |
| Decode responses | `ReadBody`, `DecodeText`, `DecodeJSON`, `Client.QueryJSONInto` |
| Error handling | `CheckStatus`, `StatusError`, `AsStatusError` |
| Result location | `Client.FetchResult`, `ResponseInfo.QueryURI` |
| Discovery | `Client.Options`, `Discovery`, `SupportsQuery` |
| Retries | `Client.DoWithRetry`, `DefaultRetryPolicy`, `RetryOnTransient`, `ExponentialBackoff` |
| Client options | `WithBaseURL`, `WithUserAgent`, `WithDefaultHeader`, `WithHeader` |
| **Serve** QUERY | `Handler`, `NewHandler`, `SetResultLocation`, `AdvertiseQuery` |
| Caching | `CacheKey`, `CacheKeyForBody`, `Body.Bytes` |

## Quick start

```go
client := querygo.NewClient(nil,
    querygo.WithBaseURL("https://example.org"),
    querygo.WithUserAgent("my-app/1.0"),
)

var result struct {
    Hits int `json:"hits"`
}
err := client.QueryJSONInto(ctx, "/search",
    map[string]any{"q": "golang"}, &result, nil)
```

## Sending a query

Pick the helper that matches your body's content type:

```go
// Raw reader
resp, err := client.Query(ctx, url, body, "application/sql")

// String / bytes / form / JSON
resp, err := client.QueryString(ctx, url, "application/sql", "SELECT 1", "application/json", nil)
resp, err := client.QueryForm(ctx, url, url.Values{"q": {"golang"}}, "application/json", nil)
resp, err := client.QueryJSON(ctx, url, payload, "application/json", nil)
```

All helpers return the raw `*http.Response`; **you must close the body**.

## Handling responses

```go
resp, err := client.QueryJSON(ctx, url, payload, querygo.MediaTypeJSON, nil)
if err != nil {
    return err
}

var out SearchResult
if err := querygo.DecodeJSON(resp, &out); err != nil {
    // Non-2xx responses become a *StatusError.
    if se, ok := querygo.AsStatusError(err); ok {
        log.Printf("server returned %d: %s", se.StatusCode, se.Body)
    }
    return err
}
```

`ReadBody`, `DecodeText` and `DecodeJSON` validate the status and always close
the body.

## Fetching the materialized result

A `QUERY` response may point to a cacheable, `GET`-retrievable result resource
via the `Content-Location` (or `Location`) header field. `FetchResult` follows
it:

```go
resp, _ := client.QueryJSON(ctx, url, payload, querygo.MediaTypeJSON, nil)

result, err := client.FetchResult(ctx, resp, nil)
if errors.Is(err, querygo.ErrNoQueryResult) {
    // The response carried the result inline instead.
}
```

## Discovering QUERY support

```go
discovery, err := client.Options(ctx, url, nil)
if discovery.SupportsQuery {
    fmt.Println("accepted query formats:", discovery.AcceptQuery)
}
```

## Retries

Because `QUERY` is safe and idempotent, transient failures can be retried
transparently. The request body is replayed via `Request.GetBody`, which the
`Body` constructors populate automatically.

```go
req := querygo.RequestFromBody(url,
    querygo.NewBodyString("application/sql", "SELECT 1"),
    "application/json", nil)

resp, err := client.DoWithRetry(ctx, req, querygo.DefaultRetryPolicy())
```

`DefaultRetryPolicy` makes up to 3 attempts with exponential backoff, retrying
transport errors and the `429`, `502`, `503`, `504` status codes.

## Serving QUERY

`Handler` implements `http.Handler`. It dispatches QUERY requests, answers
`OPTIONS` with the advertised support, and returns `405` (with an `Allow`
header) for other methods. It mounts on the standard `http.ServeMux`.

```go
mux := http.NewServeMux()
mux.Handle("/search", querygo.Handler{
    AcceptQuery:    []string{querygo.MediaTypeJSON},
    AllowedMethods: []string{http.MethodGet},
    Query: func(w http.ResponseWriter, r *http.Request) {
        // r.Body holds the query.
        querygo.SetResultLocation(w.Header(), "/results/42")
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"hits":2}`))
    },
})
```

When `AcceptQuery` is set, requests with an unlisted `Content-Type` are
rejected with `415 Unsupported Media Type`. Use `AdvertiseQuery` from any
handler (e.g. a `GET`) to surface QUERY support inline.

## Caching

Unlike `GET`, a QUERY cache entry must be keyed on the request body. `CacheKey`
computes a stable key over the method, URL, content type and body:

```go
key := querygo.CacheKey(url, querygo.MediaTypeJSON, []byte(`{"q":"go"}`))
// or, from a Body:
key, _ := querygo.CacheKeyForBody(url, body)
```

## Testing

```sh
go test ./... -race -cover
```
