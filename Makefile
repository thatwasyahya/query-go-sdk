.PHONY: all build test race cover vet fmt fmtcheck tidy check

all: check

build:
	go build ./...

test:
	go test ./... -count=1

race:
	go test ./... -race -count=1

cover:
	go test ./... -cover

vet:
	go vet ./...

fmt:
	gofmt -w .

fmtcheck:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)

tidy:
	go mod tidy

check: fmtcheck vet build race
