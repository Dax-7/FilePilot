package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrefersConfiguredBackendPath(t *testing.T) {
	configured := makeBackendFile(t, t.TempDir(), "configured-engine")
	bundledDir := t.TempDir()
	pathDir := t.TempDir()
	_ = makeBackendFile(t, bundledDir, "transfer-engine")
	_ = makeBackendFile(t, pathDir, "transfer-engine")

	result, err := Resolve(ResolveRequest{
		ConfiguredPath: configured,
		BundledDir:     bundledDir,
		PathDirs:       []string{pathDir},
		CandidateNames: []string{"transfer-engine"},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if result.Source != SourceConfigured || result.Path != configured {
		t.Fatalf("expected configured backend, got %#v", result)
	}
}

func TestResolveFallsBackToBundledBeforePath(t *testing.T) {
	bundledDir := t.TempDir()
	pathDir := t.TempDir()
	bundled := makeBackendFile(t, bundledDir, "transfer-engine")
	_ = makeBackendFile(t, pathDir, "transfer-engine")

	result, err := Resolve(ResolveRequest{
		BundledDir:     bundledDir,
		PathDirs:       []string{pathDir},
		CandidateNames: []string{"transfer-engine"},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if result.Source != SourceBundled || result.Path != bundled {
		t.Fatalf("expected bundled backend, got %#v", result)
	}
}

func TestResolveFindsBundledBackendInPlatformArchReleaseLayout(t *testing.T) {
	bundleRoot := t.TempDir()
	bundled := makeBackendFile(t, filepath.Join(bundleRoot, "backend", "linux-amd64"), "filepilot-backend")

	result, err := Resolve(ResolveRequest{
		BundledDir:     filepath.Join(bundleRoot, "backend"),
		PathDirs:       nil,
		CandidateNames: []string{"filepilot-backend"},
		Platform:       "linux",
		Arch:           "amd64",
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if result.Source != SourceBundled || result.Path != bundled {
		t.Fatalf("expected platform bundled backend, got %#v", result)
	}
}

func TestReleaseBundledDirUsesExecutableSiblingBackendDirectory(t *testing.T) {
	executable := filepath.Join("dist", "filepilot", "filepilot.exe")
	got := ReleaseBundledDir(executable)
	want := filepath.Join("dist", "filepilot", "backend")
	if got != want {
		t.Fatalf("ReleaseBundledDir(%q) = %q, want %q", executable, got, want)
	}
}

func TestBundledPlatformDirUsesStablePlatformArchName(t *testing.T) {
	got := BundledPlatformDir(filepath.Join("dist", "filepilot", "backend"), "windows", "amd64")
	want := filepath.Join("dist", "filepilot", "backend", "windows-amd64")
	if got != want {
		t.Fatalf("BundledPlatformDir returned %q, want %q", got, want)
	}
}

func TestResolveFallsBackToPath(t *testing.T) {
	pathDir := t.TempDir()
	pathBackend := makeBackendFile(t, pathDir, "transfer-engine")

	result, err := Resolve(ResolveRequest{
		BundledDir:     t.TempDir(),
		PathDirs:       []string{pathDir},
		CandidateNames: []string{"transfer-engine"},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if result.Source != SourcePath || result.Path != pathBackend {
		t.Fatalf("expected PATH backend, got %#v", result)
	}
}

func TestResolveReturnsBackendNotFound(t *testing.T) {
	_, err := Resolve(ResolveRequest{
		BundledDir:     t.TempDir(),
		PathDirs:       []string{t.TempDir()},
		CandidateNames: []string{"transfer-engine"},
	})
	if err == nil {
		t.Fatal("expected backend not found error")
	}
	if err.Code != "BACKEND_NOT_FOUND" {
		t.Fatalf("error code = %s, want BACKEND_NOT_FOUND", err.Code)
	}
}

func makeBackendFile(t *testing.T, dir string, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("fake backend"), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return path
}
