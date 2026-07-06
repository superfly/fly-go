package fly

import (
	"fmt"
	"path"
	"runtime/debug"
)

const modulePath = "github.com/superfly/fly-go"

func versionOrUnknown(v string) string {
	if v == "" {
		return "unknown"
	}

	return v
}

// flyGoVersion returns the version of this module as resolved in the
// consuming binary (e.g. "v0.7.0"), or "unknown" if it can't be determined -
// for example when built without module support, or when this module is
// replaced with a local filesystem path.
func flyGoVersion(info *debug.BuildInfo) string {
	if info.Main.Path == modulePath {
		return versionOrUnknown(info.Main.Version)
	}

	for _, dep := range info.Deps {
		if dep.Path != modulePath {
			continue
		}
		if dep.Replace != nil {
			dep = dep.Replace
		}

		return versionOrUnknown(dep.Version)
	}

	return "unknown"
}

// DefaultUserAgent returns the User-Agent clients should fall back to when
// no caller-supplied name/version is available. It identifies the consuming
// binary using its main module's build info, with fly-go's own resolved
// version appended as a suffix, e.g. "flyctl/v1.2.3 fly-go/v0.7.0". If
// fly-go is itself the main module (running its own tests/binaries), it
// returns just "fly-go/<version>".
func DefaultUserAgent() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "fly-go/unknown"
	}

	version := flyGoVersion(info)
	if info.Main.Path == modulePath {
		return "fly-go/" + version
	}

	name := "unknown"
	if info.Main.Path != "" {
		name = path.Base(info.Main.Path)
	}

	return fmt.Sprintf("%s/%s fly-go/%s", name, versionOrUnknown(info.Main.Version), version)
}
