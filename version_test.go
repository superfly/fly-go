package fly

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionOrUnknown(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "unknown"},
		{"v1.2.3", "v1.2.3"},
		{"(devel)", "(devel)"},
	}

	for _, c := range cases {
		if got := versionOrUnknown(c.in); got != c.want {
			t.Errorf("versionOrUnknown(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFlyGoVersion(t *testing.T) {
	cases := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "fly-go is the main module",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "v0.7.0"},
			},
			want: "v0.7.0",
		},
		{
			name: "fly-go is the main module with no version",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: ""},
			},
			want: "unknown",
		},
		{
			name: "fly-go is a regular dependency",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/superfly/flyctl", Version: "v1.2.3"},
				Deps: []*debug.Module{
					{Path: "github.com/some/other-dep", Version: "v9.9.9"},
					{Path: modulePath, Version: "v0.7.0"},
				},
			},
			want: "v0.7.0",
		},
		{
			name: "fly-go is replaced with a local filesystem path",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/superfly/flyctl", Version: "v1.2.3"},
				Deps: []*debug.Module{
					{
						Path:    modulePath,
						Version: "v0.0.0-00010101000000-000000000000",
						Replace: &debug.Module{Path: "../fly-go", Version: ""},
					},
				},
			},
			want: "unknown",
		},
		{
			name: "fly-go is replaced with another pinned module version",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/superfly/flyctl", Version: "v1.2.3"},
				Deps: []*debug.Module{
					{
						Path:    modulePath,
						Version: "v0.0.0-00010101000000-000000000000",
						Replace: &debug.Module{Path: "github.com/example/fly-go-fork", Version: "v0.1.0"},
					},
				},
			},
			want: "v0.1.0",
		},
		{
			name: "fly-go is not present at all",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/superfly/flyctl", Version: "v1.2.3"},
				Deps: []*debug.Module{
					{Path: "github.com/some/other-dep", Version: "v9.9.9"},
				},
			},
			want: "unknown",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := flyGoVersion(c.info); got != c.want {
				t.Errorf("flyGoVersion() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDefaultUserAgentFromBuildInfo(t *testing.T) {
	cases := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "fly-go is the main module",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "v0.7.0"},
			},
			want: "fly-go/v0.7.0",
		},
		{
			name: "fly-go is the main module in devel mode",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: ""},
			},
			want: "fly-go/unknown",
		},
		{
			name: "consumer with fly-go as a dependency",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/superfly/flyctl", Version: "v1.2.3"},
				Deps: []*debug.Module{
					{Path: modulePath, Version: "v0.7.0"},
				},
			},
			want: "flyctl/v1.2.3 fly-go/v0.7.0",
		},
		{
			name: "consumer built in devel mode with locally-replaced fly-go",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/superfly/flyctl", Version: ""},
				Deps: []*debug.Module{
					{
						Path:    modulePath,
						Version: "v0.0.0-00010101000000-000000000000",
						Replace: &debug.Module{Path: "../fly-go", Version: ""},
					},
				},
			},
			want: "flyctl/unknown fly-go/unknown",
		},
		{
			name: "consumer with a nested module path",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/superfly/flyctl/internal/tool", Version: "v1.2.3"},
				Deps: []*debug.Module{
					{Path: modulePath, Version: "v0.7.0"},
				},
			},
			want: "tool/v1.2.3 fly-go/v0.7.0",
		},
		{
			name: "consumer with no main path",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "", Version: ""},
			},
			want: "unknown/unknown fly-go/unknown",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := defaultUserAgent(c.info); got != c.want {
				t.Errorf("defaultUserAgent() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDefaultUserAgent(t *testing.T) {
	// Running as fly-go's own test binary, fly-go is the main module, so the
	// real build info should collapse to a bare "fly-go/<version>".
	ua := DefaultUserAgent()
	if !strings.HasPrefix(ua, "fly-go/") {
		t.Fatalf("DefaultUserAgent() = %q, want it to start with %q", ua, "fly-go/")
	}
}
