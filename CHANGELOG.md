# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-21

Initial release: a dependency-free Go SDK for the HTTP QUERY method
([RFC 10008](https://www.rfc-editor.org/rfc/rfc10008.html)).

### Added

- **Client**: `QUERY` requests via `Do`, `Query`, and typed helpers
  (`QueryString`, `QueryBytes`, `QueryForm`, `QueryJSON`, `QueryJSONInto`).
- **Request bodies**: `NewBodyString`, `NewBodyBytes`, `NewBodyForm`,
  `NewBodyJSON`, replayable via `GetReader` / `Request.GetBody`.
- **Response handling**: `CheckStatus`, `StatusError`, `AsStatusError`,
  `ReadBody`, `DecodeText`, `DecodeJSON`.
- **Result and equivalent resource** (RFC 10008 §2.3–2.4): `FetchResult`
  (follows `Content-Location`), `FetchEquivalent` (follows `Location`),
  `ResponseInfo.ResultURI`, `ResponseInfo.EquivalentResourceURI`.
- **Discovery**: `Options`, `Discovery`, `SupportsQuery`, with `Accept-Query`
  parsed and serialized as an RFC 9651 Structured Fields List, including
  `*/*` and `type/*` wildcard matching.
- **Server**: `Handler` (implements `http.Handler`), `NewHandler`,
  `SetResultLocation`, `AdvertiseQuery`. Rejects missing `Content-Type` with
  `400` and unsupported query media types with `415` (RFC 10008 §2).
- **Retries**: `DoWithRetry`, `RetryPolicy`, `DefaultRetryPolicy`,
  `RetryOnTransient`, `ExponentialBackoff`.
- **Caching**: `CacheKey`, `CacheKeyForBody`, `Body.Bytes`.
- **Client options**: `WithBaseURL`, `WithUserAgent`, `WithDefaultHeader`,
  `WithHeader`.

[Unreleased]: https://github.com/thatwasyahya/query-go-sdk/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/thatwasyahya/query-go-sdk/releases/tag/v0.1.0
