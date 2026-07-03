package clientsignals

import "os"

// isCI reports whether a CI environment is detected. This checks presence
// only (via LookupEnv, not Getenv() != ""), so e.g. CI="" set by some CI
// systems still counts as present.
func isCI() bool {
	if _, ok := os.LookupEnv("CI"); ok {
		return true
	}
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return true
	}

	return false
}
