package paths

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathsUsePlatformConventions(t *testing.T) {
	cases := []struct {
		name       string
		goos       string
		env        map[string]string
		configPath string
		cacheDir   string
		logDir     string
		download   string
	}{
		{
			name:       "linux",
			goos:       "linux",
			env:        map[string]string{"HOME": "/home/alice"},
			configPath: filepath.FromSlash("/home/alice/.config/filepilot/config.toml"),
			cacheDir:   filepath.FromSlash("/home/alice/.cache/filepilot"),
			logDir:     filepath.FromSlash("/home/alice/.local/state/filepilot/logs"),
			download:   filepath.FromSlash("/home/alice/Downloads/FilePilot"),
		},
		{
			name:       "darwin",
			goos:       "darwin",
			env:        map[string]string{"HOME": "/Users/alice"},
			configPath: filepath.FromSlash("/Users/alice/Library/Application Support/FilePilot/config.toml"),
			cacheDir:   filepath.FromSlash("/Users/alice/Library/Caches/FilePilot"),
			logDir:     filepath.FromSlash("/Users/alice/Library/Logs/FilePilot"),
			download:   filepath.FromSlash("/Users/alice/Downloads/FilePilot"),
		},
		{
			name:       "windows",
			goos:       "windows",
			env:        map[string]string{"APPDATA": `C:\Users\alice\AppData\Roaming`, "LOCALAPPDATA": `C:\Users\alice\AppData\Local`, "USERPROFILE": `C:\Users\alice`},
			configPath: filepath.Join(`C:\Users\alice\AppData\Roaming`, "FilePilot", "config.toml"),
			cacheDir:   filepath.Join(`C:\Users\alice\AppData\Local`, "FilePilot", "Cache"),
			logDir:     filepath.Join(`C:\Users\alice\AppData\Local`, "FilePilot", "Logs"),
			download:   filepath.Join(`C:\Users\alice`, "Downloads", "FilePilot"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(tc.goos, mapLookup(tc.env))
			if err != nil {
				t.Fatalf("Resolve returned error: %v", err)
			}
			if got.ConfigPath != tc.configPath {
				t.Fatalf("config path mismatch: got %q want %q", got.ConfigPath, tc.configPath)
			}
			if got.CacheDir != tc.cacheDir {
				t.Fatalf("cache dir mismatch: got %q want %q", got.CacheDir, tc.cacheDir)
			}
			if got.LogDir != tc.logDir {
				t.Fatalf("log dir mismatch: got %q want %q", got.LogDir, tc.logDir)
			}
			if got.DownloadDir != tc.download {
				t.Fatalf("download dir mismatch: got %q want %q", got.DownloadDir, tc.download)
			}
		})
	}
}

func TestEnvironmentOverridesConfiguredFilePilotPaths(t *testing.T) {
	got, err := Resolve("linux", mapLookup(map[string]string{
		"HOME":                "/home/alice",
		"FILEPILOT_CONFIG":    "/tmp/filepilot/config.toml",
		"FILEPILOT_CACHE_DIR": "/tmp/filepilot/cache",
		"FILEPILOT_LOG_DIR":   "/tmp/filepilot/logs",
	}))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.ConfigPath != filepath.FromSlash("/tmp/filepilot/config.toml") {
		t.Fatalf("config override mismatch: %q", got.ConfigPath)
	}
	if got.CacheDir != filepath.FromSlash("/tmp/filepilot/cache") {
		t.Fatalf("cache override mismatch: %q", got.CacheDir)
	}
	if got.LogDir != filepath.FromSlash("/tmp/filepilot/logs") {
		t.Fatalf("log override mismatch: %q", got.LogDir)
	}
}

func TestResolveWithHomeUsesUserHomeForDefaultDownloadDir(t *testing.T) {
	got, err := ResolveWithHome("linux", mapLookup(map[string]string{
		"HOME": "/env-home/alice",
	}), filepath.FromSlash("/home/alice"))
	if err != nil {
		t.Fatalf("ResolveWithHome returned error: %v", err)
	}

	want := filepath.FromSlash("/home/alice/Downloads/FilePilot")
	if got.DownloadDir != want {
		t.Fatalf("download dir mismatch: got %q want %q", got.DownloadDir, want)
	}
}

func mapLookup(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
