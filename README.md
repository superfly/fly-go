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

## Cutting a Release

If you have write access to this repo, you can ship a release with:

`scripts/bump_version.sh`

Or a prerelease with:

`scripts/bump_version.sh prerel`

Then:
* Go to the Releases page https://github.com/superfly/fly-go/releases
* Click "Draft a New Release"
* Go to "Choose a tag", type the new tag (from the `script/bump_version.sh` output above)
* Click "Generate Release Notes"
* Click "Publish release" at the bottom
