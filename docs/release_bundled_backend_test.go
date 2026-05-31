package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestBundledBackendReleaseDocCoversLayoutAndHITLPolicy(t *testing.T) {
	content, err := os.ReadFile("release-bundled-backend.md")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"backend/<goos>-<goarch>/filepilot-backend",
		"source: bundled",
		"backend binary source",
		"license",
		"Checksums",
		"must not download backend binaries at runtime",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release doc missing %q:\n%s", want, text)
		}
	}
}
