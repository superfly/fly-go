fly-go
======

Go client for the Fly.io API. This library is primarily used by [flyctl][] but
it can be used by any project that wants to automate its [Fly.io] deployment.

[flyctl]: https://github.com/superfly/flyctl
[Fly.io]: https://fly.io


## Development

If you are making changes in another project and need to test `fly-go` changes
locally, you can enable a [Go workspace][]. For example, if you have a directory
structure like this:

```
superfly/
├── fly-go
└── flyctl
```

Then you can initialize a Go workspace in the `superfly` parent directory and
add your project directories so that `flyctl` can use your local `fly-go`:

```sh
go work init
go work use ./flyctl
go work use ./fly-go
```

[Go workspace]: https://go.dev/blog/get-familiar-with-workspaces
