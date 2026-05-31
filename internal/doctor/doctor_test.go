package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"filepilot/internal/backend"
	"filepilot/internal/paths"
)

func TestRunReportsBackendAndWritableDirectories(t *testing.T) {
	root := t.TempDir()
	backendPath := filepath.Join(root, "transfer-engine.cmd")
	writeDoctorExecutable(t, backendPath, "FilePilot test backend 1.0")

	report := Run(Request{
		Paths: paths.Paths{
			ConfigPath:  filepath.Join(root, "config.toml"),
			CacheDir:    filepath.Join(root, "cache"),
			LogDir:      filepath.Join(root, "logs"),
			DownloadDir: filepath.Join(root, "downloads"),
		},
		Backend: backend.ResolveRequest{
			ConfiguredPath: backendPath,
			CandidateNames: []string{"transfer-engine"},
		},
		Getenv: func(string) string { return "" },
	})

	if report.Fatal != nil {
		t.Fatalf("expected no fatal error, got %#v", report.Fatal)
	}
	if report.Backend.Source != backend.SourceConfigured || report.Backend.Path != backendPath {
		t.Fatalf("backend report mismatch: %#v", report.Backend)
	}
	if !strings.Contains(report.Backend.Version, "1.0") {
		t.Fatalf("backend version mismatch: %#v", report.Backend)
	}
	if len(report.DirectoryChecks) != 3 {
		t.Fatalf("expected cache/log/download checks, got %#v", report.DirectoryChecks)
	}
	for _, check := range report.DirectoryChecks {
		if !check.Writable {
			t.Fatalf("expected writable check, got %#v", check)
		}
	}
}

func TestRunReportsFatalMissingBackend(t *testing.T) {
	root := t.TempDir()
	report := Run(Request{
		Paths: paths.Paths{
			ConfigPath:  filepath.Join(root, "config.toml"),
			CacheDir:    filepath.Join(root, "cache"),
			LogDir:      filepath.Join(root, "logs"),
			DownloadDir: filepath.Join(root, "downloads"),
		},
		Backend: backend.ResolveRequest{
			BundledDir:     filepath.Join(root, "bundled"),
			PathDirs:       []string{filepath.Join(root, "path")},
			CandidateNames: []string{"transfer-engine"},
		},
		Getenv: func(string) string { return "" },
	})

	if report.Fatal == nil || report.Fatal.Code != "BACKEND_NOT_FOUND" {
		t.Fatalf("expected BACKEND_NOT_FOUND fatal, got %#v", report.Fatal)
	}
}

func TestRunFindsBundledBackendFromReleaseLayoutWithoutPath(t *testing.T) {
	root := t.TempDir()
	bundledDir := filepath.Join(root, "bin", "backend")
	bundledBackend := filepath.Join(bundledDir, runtime.GOOS+"-"+runtime.GOARCH, "filepilot-backend"+scriptExt())
	writeDoctorExecutable(t, bundledBackend, "FilePilot bundled backend 1.0")

	report := Run(Request{
		Paths: paths.Paths{
			ConfigPath:  filepath.Join(root, "config.toml"),
			CacheDir:    filepath.Join(root, "cache"),
			LogDir:      filepath.Join(root, "logs"),
			DownloadDir: filepath.Join(root, "downloads"),
		},
		Backend: backend.ResolveRequest{
			BundledDir: bundledDir,
			PathDirs:   nil,
			Platform:   runtime.GOOS,
			Arch:       runtime.GOARCH,
		},
		Getenv: func(string) string { return "" },
	})

	if report.Fatal != nil {
		t.Fatalf("expected bundled backend to avoid fatal error, got %#v", report.Fatal)
	}
	if report.Backend.Source != backend.SourceBundled || report.Backend.Path != filepath.Clean(bundledBackend) {
		t.Fatalf("backend report mismatch: %#v", report.Backend)
	}
	if !strings.Contains(report.Backend.Version, "bundled backend 1.0") {
		t.Fatalf("backend version mismatch: %#v", report.Backend)
	}
}

func scriptExt() string {
	if runtime.GOOS == "windows" {
		return ".cmd"
	}
	return ""
}

func TestRunProxyWarningsDoNotBecomeFatal(t *testing.T) {
	root := t.TempDir()
	backendPath := filepath.Join(root, "transfer-engine.cmd")
	writeDoctorExecutable(t, backendPath, "FilePilot test backend 1.0")

	report := Run(Request{
		Paths: paths.Paths{
			ConfigPath:  filepath.Join(root, "config.toml"),
			CacheDir:    filepath.Join(root, "cache"),
			LogDir:      filepath.Join(root, "logs"),
			DownloadDir: filepath.Join(root, "downloads"),
		},
		Backend: backend.ResolveRequest{
			ConfiguredPath: backendPath,
			CandidateNames: []string{"transfer-engine"},
		},
		Getenv: func(key string) string {
			if key == "HTTPS_PROXY" {
				return "http://127.0.0.1:7890"
			}
			return ""
		},
	})

	if report.Fatal != nil {
		t.Fatalf("proxy warning should not be fatal: %#v", report.Fatal)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected proxy warning")
	}
}

func writeDoctorExecutable(t *testing.T, path string, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	content := "@echo off\r\necho " + version + "\r\n"
	if filepath.Ext(path) != ".cmd" {
		content = "#!/bin/sh\necho '" + version + "'\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
