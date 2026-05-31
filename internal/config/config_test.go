package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsWhenConfigFileDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	effective, err := Load(path, filepath.Join(t.TempDir(), "Downloads", "FilePilot"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if effective.Values.BackendPath != "" {
		t.Fatalf("default backend_path = %q, want empty", effective.Values.BackendPath)
	}
	if effective.Values.DownloadDir != effective.DefaultDownloadDir {
		t.Fatalf("default download_dir = %q, want %q", effective.Values.DownloadDir, effective.DefaultDownloadDir)
	}
	if !effective.Values.AutoUnpack {
		t.Fatal("default auto_unpack = false, want true")
	}
	if effective.Values.KeepPackages {
		t.Fatal("default keep_packages = true, want false")
	}
	if effective.Values.JSONOutput {
		t.Fatal("default json_output = true, want false")
	}
}

func TestSaveAndLoadTOMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	values := Values{
		BackendPath:  filepath.Join("tools", "backend-bin"),
		DownloadDir:  filepath.Join("D:", "Downloads", "FilePilot"),
		AutoUnpack:   false,
		KeepPackages: true,
		JSONOutput:   true,
	}

	if err := Save(path, values); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(content), `backend_path = "tools`) {
		t.Fatalf("expected TOML backend_path in file, got:\n%s", string(content))
	}

	effective, err := Load(path, filepath.Join(t.TempDir(), "Downloads", "FilePilot"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if effective.Values != values {
		t.Fatalf("loaded values mismatch:\ngot  %#v\nwant %#v", effective.Values, values)
	}
}

func TestSetSupportedConfigKeys(t *testing.T) {
	values := Defaults("")
	cases := []struct {
		key   string
		value string
		check func(Values) bool
	}{
		{"backend_path", "backend-bin", func(v Values) bool { return v.BackendPath == "backend-bin" }},
		{"download_dir", "downloads", func(v Values) bool { return v.DownloadDir == "downloads" }},
		{"auto_unpack", "false", func(v Values) bool { return !v.AutoUnpack }},
		{"keep_packages", "true", func(v Values) bool { return v.KeepPackages }},
		{"json_output", "true", func(v Values) bool { return v.JSONOutput }},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			next, err := Set(values, tc.key, tc.value)
			if err != nil {
				t.Fatalf("Set returned error: %v", err)
			}
			if !tc.check(next) {
				t.Fatalf("Set(%q, %q) produced %#v", tc.key, tc.value, next)
			}
		})
	}
}
