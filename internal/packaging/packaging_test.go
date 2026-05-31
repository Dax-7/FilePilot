package packaging

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateDirectoryPackageIncludesManifestAndFiles(t *testing.T) {
	source := filepath.Join(t.TempDir(), "results")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "metrics.txt"), []byte("accuracy=0.99\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "results.tar.gz")
	result, err := CreateDirectoryPackage(Request{
		SourceDir:  source,
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("CreateDirectoryPackage returned error: %v", err)
	}

	if result.PackagePath != outputPath {
		t.Fatalf("PackagePath = %q, want %q", result.PackagePath, outputPath)
	}
	if result.Manifest.PayloadType != PayloadTypeDirectoryPackage {
		t.Fatalf("payload type = %q", result.Manifest.PayloadType)
	}
	if result.Manifest.SourceName != "results" {
		t.Fatalf("source_name = %q, want results", result.Manifest.SourceName)
	}
	if result.Manifest.CreatedBy != "filepilot" {
		t.Fatalf("created_by = %q, want filepilot", result.Manifest.CreatedBy)
	}
	if _, err := time.Parse(time.RFC3339, result.Manifest.CreatedAt); err != nil {
		t.Fatalf("created_at is not RFC3339: %q", result.Manifest.CreatedAt)
	}

	entries := readTarGz(t, outputPath)
	if _, ok := entries[".filepilot/manifest.json"]; !ok {
		t.Fatalf("package missing manifest; entries=%v", keys(entries))
	}
	if string(entries["results/nested/metrics.txt"]) != "accuracy=0.99\n" {
		t.Fatalf("source file content mismatch; entries=%v", keys(entries))
	}

	var manifest Manifest
	if err := json.Unmarshal(entries[".filepilot/manifest.json"], &manifest); err != nil {
		t.Fatalf("manifest is not JSON: %v", err)
	}
	if manifest.SchemaVersion != 1 || manifest.PayloadType != PayloadTypeDirectoryPackage || manifest.SourceName != "results" {
		t.Fatalf("manifest mismatch: %#v", manifest)
	}
}

func readTarGz(t *testing.T, path string) map[string][]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("NewReader returned error: %v", err)
	}
	defer gzipReader.Close()

	entries := map[string][]byte{}
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar Next returned error: %v", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		entries[header.Name] = content
	}
	return entries
}

func keys(entries map[string][]byte) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}
