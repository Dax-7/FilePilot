package cacheclean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeleteItemRefusesOutsideCacheRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.tar.gz")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := deleteItem(root, Item{Path: outsideFile})
	if err == nil {
		t.Fatal("expected outside-root deletion to be refused")
	}
	if !strings.Contains(err.Error(), "outside FilePilot cache root") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(outsideFile); statErr != nil {
		t.Fatalf("outside file should remain after refused deletion: %v", statErr)
	}
}
