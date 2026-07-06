package fly

import "runtime/debug"

const modulePath = "github.com/superfly/fly-go"

// moduleVersion returns the version of this module as resolved in the
// consuming binary (e.g. "v0.7.0"), or "unknown" if it can't be determined -
// for example when built without module support, or when this module is
// replaced with a local filesystem path.
func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	if info.Main.Path == modulePath {
		if info.Main.Version != "" {
			return info.Main.Version
		}

		return "unknown"
	}

	for _, dep := range info.Deps {
		if dep.Path != modulePath {
			continue
		}
		if dep.Replace != nil {
			dep = dep.Replace
		}
		if dep.Version != "" {
			return dep.Version
		}

		return "unknown"
	}

	return "unknown"
}

// DefaultUserAgent returns the User-Agent clients should fall back to when
// no caller-supplied name/version is available: "fly-go/<version>".
func DefaultUserAgent() string {
	return "fly-go/" + moduleVersion()
}
