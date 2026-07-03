package clientsignals

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsInteractiveFile_RegularFileIsNotInteractive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-tty")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()

	if isInteractiveFile(f) {
		t.Fatal("expected a regular file to not be reported as interactive")
	}
}
