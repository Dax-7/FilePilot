package backend

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"filepilot/internal/fperrors"
)

type Source string

const (
	SourceConfigured Source = "configured"
	SourceBundled    Source = "bundled"
	SourcePath       Source = "path"
)

var DefaultCandidateNames = DefaultCandidateNamesFor(runtime.GOOS)

type ResolveRequest struct {
	ConfiguredPath string
	BundledDir     string
	PathDirs       []string
	CandidateNames []string
	Platform       string
	Arch           string
}

type Resolved struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Source  Source `json:"source"`
	Version string `json:"version,omitempty"`
}

func Resolve(req ResolveRequest) (Resolved, *fperrors.Error) {
	candidateNames := req.CandidateNames
	platform := firstNonEmpty(req.Platform, runtime.GOOS)
	if len(candidateNames) == 0 {
		candidateNames = DefaultCandidateNamesFor(platform)
	}

	if req.ConfiguredPath != "" {
		if isUsableFile(req.ConfiguredPath) {
			return Resolved{Name: "transfer-engine", Path: filepath.Clean(req.ConfiguredPath), Source: SourceConfigured}, nil
		}
		return Resolved{}, fperrors.New(fperrors.BackendUnavailable, "Configured backend is not usable.", "Check backend_path in FilePilot config.")
	}

	if req.BundledDir != "" {
		arch := firstNonEmpty(req.Arch, runtime.GOARCH)
		if path := findInDir(BundledPlatformDir(req.BundledDir, platform, arch), candidateNames); path != "" {
			return Resolved{Name: "transfer-engine", Path: path, Source: SourceBundled}, nil
		}
		if path := findInDir(req.BundledDir, candidateNames); path != "" {
			return Resolved{Name: "transfer-engine", Path: path, Source: SourceBundled}, nil
		}
	}

	for _, dir := range req.PathDirs {
		if path := findInDir(dir, candidateNames); path != "" {
			return Resolved{Name: "transfer-engine", Path: path, Source: SourcePath}, nil
		}
	}

	return Resolved{}, fperrors.New(fperrors.BackendNotFound, "No usable transfer backend was found.", "Configure backend_path or use a FilePilot bundle with a backend.")
}

func DefaultCandidateNamesFor(platform string) []string {
	if platform == "windows" {
		return []string{"filepilot.exe", "croc.exe"}
	}
	return []string{"filepilot", "croc"}
}

func DefaultBundledDir(executablePath string) string {
	return ReleaseBundledDir(executablePath)
}

func ReleaseBundledDir(executablePath string) string {
	return filepath.Join(filepath.Dir(executablePath), "backend")
}

func BundledPlatformDir(bundledDir string, platform string, arch string) string {
	return filepath.Join(bundledDir, platform+"-"+arch)
}

func PathDirsFromEnv(pathEnv string) []string {
	if pathEnv == "" {
		return nil
	}
	return filepath.SplitList(pathEnv)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func DetectVersion(ctx context.Context, path string) string {
	if path == "" {
		return ""
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
	}
	output, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		output, err = exec.CommandContext(ctx, path, "version").CombinedOutput()
		if err != nil {
			return "unknown"
		}
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "unknown"
	}
	return version
}

func findInDir(dir string, names []string) string {
	for _, name := range expandNames(names) {
		path := filepath.Join(dir, name)
		if isUsableFile(path) {
			return filepath.Clean(path)
		}
	}
	return ""
}

func isUsableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

func expandNames(names []string) []string {
	expanded := make([]string, 0, len(names)*3)
	for _, name := range names {
		expanded = append(expanded, name)
		if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
			expanded = append(expanded, name+".exe", name+".cmd", name+".bat")
		}
	}
	return expanded
}
