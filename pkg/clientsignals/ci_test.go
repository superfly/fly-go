package clientsignals

import (
	"os"
	"testing"
)

// unsetEnv forcibly unsets an env var for the duration of the test,
// restoring its prior value (or absence) afterward. Needed because
// t.Setenv can only set values, not unset them.
func unsetEnv(t *testing.T, key string) {
	t.Helper()

	prev, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		}
	})
}

func TestIsCI(t *testing.T) {
	t.Run("neither set", func(t *testing.T) {
		unsetEnv(t, "CI")
		unsetEnv(t, "GITHUB_ACTIONS")

		if isCI() {
			t.Fatal("expected isCI() to be false with no CI env vars set")
		}
	})

	t.Run("CI set to true", func(t *testing.T) {
		t.Setenv("CI", "true")
		if !isCI() {
			t.Fatal("expected isCI() to be true when CI=true")
		}
	})

	t.Run("CI set to empty string still counts as present", func(t *testing.T) {
		t.Setenv("CI", "")
		if !isCI() {
			t.Fatal("expected isCI() to be true when CI is present but empty")
		}
	})

	t.Run("GITHUB_ACTIONS set", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		if !isCI() {
			t.Fatal("expected isCI() to be true when GITHUB_ACTIONS is set")
		}
	})
}
