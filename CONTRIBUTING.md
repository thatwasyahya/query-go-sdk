# Contributing to querygo

Thanks for your interest in improving this SDK for the HTTP QUERY method
([RFC 10008](https://www.rfc-editor.org/rfc/rfc10008.html)).

## Ground rules

- **Stay dependency-free.** The module intentionally uses only the Go standard
  library. Please do not add third-party dependencies without discussion.
- **Follow the RFC.** Behavior changes should cite the relevant section of
  RFC 10008 (or the referenced RFCs, e.g. RFC 9110 / RFC 9651).
- **Keep the public API small and consistent** with the existing helpers.

## Before you open a pull request

Run the full local check:

```sh
make check    # gofmt, go vet, go build, race tests
```

or manually:

```sh
gofmt -l .            # must print nothing
go vet ./...
go test ./... -race -cover
```

- Format all code with `gofmt`.
- Add or update tests for any behavior change. Tests based on the examples in
  RFC 10008 are especially welcome.
- Update `CHANGELOG.md` under an "Unreleased" heading.
- Keep commits focused and write clear commit messages.

## Reporting issues

Please include the Go version, a minimal reproduction, and the relevant
request/response (headers included) when reporting a bug.
