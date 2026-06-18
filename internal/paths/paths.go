package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	ConfigPath  string
	CacheDir    string
	LogDir      string
	DownloadDir string
}

func Current() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return ResolveWithHome(runtime.GOOS, os.Getenv, home)
}

func Resolve(goos string, getenv func(string) string) (Paths, error) {
	return ResolveWithHome(goos, getenv, "")
}

func ResolveWithHome(goos string, getenv func(string) string, userHome string) (Paths, error) {
	paths, err := defaults(goos, getenv, userHome)
	if err != nil {
		return Paths{}, err
	}

	if override := getenv("FILEPILOT_CONFIG"); override != "" {
		paths.ConfigPath = filepath.Clean(override)
	}
	if override := getenv("FILEPILOT_CACHE_DIR"); override != "" {
		paths.CacheDir = filepath.Clean(override)
	}
	if override := getenv("FILEPILOT_LOG_DIR"); override != "" {
		paths.LogDir = filepath.Clean(override)
	}

	return paths, nil
}

func defaults(goos string, getenv func(string) string, userHome string) (Paths, error) {
	switch goos {
	case "windows":
		appData := first(getenv("APPDATA"), filepath.Join(getenv("USERPROFILE"), "AppData", "Roaming"))
		localAppData := first(getenv("LOCALAPPDATA"), filepath.Join(getenv("USERPROFILE"), "AppData", "Local"))
		userProfile := first(userHome, getenv("USERPROFILE"))
		if appData == "" || localAppData == "" || userProfile == "" {
			return Paths{}, errors.New("missing Windows profile environment")
		}
		return Paths{
			ConfigPath:  filepath.Join(appData, "FilePilot", "config.toml"),
			CacheDir:    filepath.Join(localAppData, "FilePilot", "Cache"),
			LogDir:      filepath.Join(localAppData, "FilePilot", "Logs"),
			DownloadDir: filepath.Join(userProfile, "Downloads", "FilePilot"),
		}, nil
	case "darwin":
		home := first(userHome, getenv("HOME"))
		if home == "" {
			return Paths{}, errors.New("missing HOME environment")
		}
		return Paths{
			ConfigPath:  filepath.Join(home, "Library", "Application Support", "FilePilot", "config.toml"),
			CacheDir:    filepath.Join(home, "Library", "Caches", "FilePilot"),
			LogDir:      filepath.Join(home, "Library", "Logs", "FilePilot"),
			DownloadDir: filepath.Join(home, "Downloads", "FilePilot"),
		}, nil
	default:
		home := first(userHome, getenv("HOME"))
		if home == "" {
			return Paths{}, errors.New("missing HOME environment")
		}
		return Paths{
			ConfigPath:  filepath.Join(home, ".config", "filepilot", "config.toml"),
			CacheDir:    filepath.Join(home, ".cache", "filepilot"),
			LogDir:      filepath.Join(home, ".local", "state", "filepilot", "logs"),
			DownloadDir: filepath.Join(home, "Downloads", "FilePilot"),
		}, nil
	}
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
