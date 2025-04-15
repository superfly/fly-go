NOW_RFC3339 = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_BRANCH = $(shell git symbolic-ref --short HEAD 2>/dev/null ||:)

default: generate

generate:
	go generate ./...

test: FORCE
	go test ./... -ldflags="-X 'github.com/superfly/flyctl/internal/buildinfo.buildDate=$(NOW_RFC3339)'" --run=$(T)

FORCE:

lint:
	golangci-lint run ./...
