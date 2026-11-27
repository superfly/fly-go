# AGENTS.md

`fly-go` is the Go client for the Fly.io API, used mainly by
[flyctl](https://github.com/superfly/flyctl). The Go version is in `go.mod`. The GraphQL client is
in the root package; the Machines API client is in `flaps/`.

## Test

```sh
make test              # go test ./...
make test T=TestName   # one test (passed to --run)
```

Tests sit next to the code they test (`foo_test.go` beside `foo.go`). CI runs `go test ./...` on
Linux, macOS and Windows.

## Generated code

```sh
make generate   # go generate ./...
```

This regenerates `genqlient.go` from `schema.graphql` and the GraphQL operations in `*.go`, and
`flaps/flapsaction_string.go` with stringer. CI runs `go generate ./... && git diff --exit-code`,
so commit whatever it changes. Don't edit generated files by hand.

## Lint

```sh
make lint                   # golangci-lint run ./...
pre-commit run --all-files  # whitespace, go mod tidy, golangci-lint
```

## Trying changes in flyctl

To build flyctl against a local fly-go, make a Go workspace in the directory that holds both
checkouts:

```sh
go work init
go work use ./flyctl
go work use ./fly-go
```

## Releases

Maintainers release with `scripts/bump_version.sh` (or `scripts/bump_version.sh prerel`); see the
README. Don't cut a release unless asked.
