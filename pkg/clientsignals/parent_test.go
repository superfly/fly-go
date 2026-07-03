package clientsignals

import "testing"

func TestClassifyParentName(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"node", "node"},
		{"/usr/bin/node", "node"},
		{"node.exe", "node"},
		{"python", "python"},
		{"python3", "python"},
		{"python2", "python"},
		{"/usr/bin/python3", "python"},
		{"bash", "shell"},
		{"zsh", "shell"},
		{"fish", "shell"},
		{"sh", "shell"},
		{"dash", "shell"},
		{"ksh", "shell"},
		{"tcsh", "shell"},
		{"csh", "shell"},
		{"cmd.exe", "shell"},
		{"powershell.exe", "shell"},
		{"pwsh", "shell"},
		{"/bin/bash", "shell"},
		{"NODE", "node"},
		{"ruby", "other"},
		{"", "other"},
		{"unknownproc", "other"},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got := classifyParentName(tc.raw)
			if got != tc.want {
				t.Fatalf("classifyParentName(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestParentBucket_NeverPanics(t *testing.T) {
	// Smoke test: ensure the real OS-specific lookup path doesn't panic and
	// always yields one of the finite buckets.
	got := parentBucket()
	switch got {
	case "node", "python", "shell", "other":
	default:
		t.Fatalf("parentBucket() returned non-finite value %q", got)
	}
}
