package file

import (
	"os"
	"testing"
)

func TestTmpFile(t *testing.T) {
	content := []byte("test")
	pattern := "test"

	tmpFile, cleanup, err := TmpFile(content, pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer cleanup()

	if tmpFile == "" {
		t.Fatalf("failed to create temp file")
	}
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatalf("temp file does not exist")
	}
}
