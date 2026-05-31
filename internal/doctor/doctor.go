package doctor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"filepilot/internal/backend"
	"filepilot/internal/fperrors"
	"filepilot/internal/paths"
)

type Request struct {
	Paths   paths.Paths
	Backend backend.ResolveRequest
	Getenv  func(string) string
}

type Report struct {
	Version         string           `json:"version"`
	OS              string           `json:"os"`
	Arch            string           `json:"arch"`
	ConfigPath      string           `json:"config_path"`
	Backend         backend.Resolved `json:"backend"`
	DirectoryChecks []DirectoryCheck `json:"directory_checks"`
	Warnings        []Warning        `json:"warnings"`
	Fatal           *fperrors.Error  `json:"fatal"`
}

type DirectoryCheck struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Writable bool   `json:"writable"`
	Error    string `json:"error,omitempty"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Run(req Request) Report {
	if req.Getenv == nil {
		req.Getenv = os.Getenv
	}

	report := Report{
		Version:    "dev",
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		ConfigPath: req.Paths.ConfigPath,
		Warnings:   proxyWarnings(req.Getenv),
	}

	resolved, fpErr := backend.Resolve(req.Backend)
	if fpErr != nil {
		report.Fatal = fpErr
	} else {
		resolved.Version = backend.DetectVersion(context.Background(), resolved.Path)
		report.Backend = resolved
	}

	report.DirectoryChecks = []DirectoryCheck{
		checkWritable("cache", req.Paths.CacheDir),
		checkWritable("log", req.Paths.LogDir),
		checkWritable("download", req.Paths.DownloadDir),
	}
	for _, check := range report.DirectoryChecks {
		if !check.Writable && report.Fatal == nil {
			report.Fatal = fperrors.New(fperrors.ConfigError, "A required FilePilot directory is not writable.", "Check config, cache, log, and download directory permissions.")
		}
	}

	return report
}

func checkWritable(name string, dir string) DirectoryCheck {
	check := DirectoryCheck{Name: name, Path: dir, Writable: true}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		check.Writable = false
		check.Error = err.Error()
		return check
	}
	probe, err := os.CreateTemp(dir, ".filepilot-doctor-*")
	if err != nil {
		check.Writable = false
		check.Error = err.Error()
		return check
	}
	probePath := probe.Name()
	_ = probe.Close()
	_ = os.Remove(probePath)
	check.Path = filepath.Clean(dir)
	return check
}

func proxyWarnings(getenv func(string) string) []Warning {
	var warnings []Warning
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY"} {
		if getenv(key) != "" {
			warnings = append(warnings, Warning{
				Code:    "PROXY_ENV",
				Message: key + " is set; backend connectivity may depend on proxy behavior.",
			})
		}
	}
	return warnings
}
