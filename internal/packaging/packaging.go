package packaging

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const PayloadTypeDirectoryPackage = "directory_package"

type Manifest struct {
	SchemaVersion int    `json:"schema_version"`
	PayloadType   string `json:"payload_type"`
	SourceName    string `json:"source_name"`
	CreatedBy     string `json:"created_by"`
	CreatedAt     string `json:"created_at"`
}

type Request struct {
	SourceDir  string
	OutputPath string
}

type Result struct {
	PackagePath string
	Manifest    Manifest
}

func CreateDirectoryPackage(req Request) (Result, error) {
	if req.SourceDir == "" {
		return Result{}, fmt.Errorf("source directory is required")
	}
	if req.OutputPath == "" {
		return Result{}, fmt.Errorf("output path is required")
	}

	sourceInfo, err := os.Stat(req.SourceDir)
	if err != nil {
		return Result{}, err
	}
	if !sourceInfo.IsDir() {
		return Result{}, fmt.Errorf("%s is not a directory", req.SourceDir)
	}
	sourceDir, err := filepath.Abs(req.SourceDir)
	if err != nil {
		return Result{}, err
	}
	outputPath, err := filepath.Abs(req.OutputPath)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return Result{}, err
	}

	manifest := Manifest{
		SchemaVersion: 1,
		PayloadType:   PayloadTypeDirectoryPackage,
		SourceName:    filepath.Base(sourceDir),
		CreatedBy:     "filepilot",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	if err := writeManifest(tarWriter, manifest); err != nil {
		return Result{}, err
	}
	if err := writeDirectory(tarWriter, sourceDir, manifest.SourceName, outputPath); err != nil {
		return Result{}, err
	}

	return Result{
		PackagePath: outputPath,
		Manifest:    manifest,
	}, nil
}

func writeManifest(tarWriter *tar.Writer, manifest Manifest) error {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	header := &tar.Header{
		Name: ".filepilot/manifest.json",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err = tarWriter.Write(content)
	return err
}

func writeDirectory(tarWriter *tar.Writer, sourceDir string, sourceName string, outputPath string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceDir {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if samePath(absPath, outputPath) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		archiveName := filepath.ToSlash(filepath.Join(sourceName, rel))
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = archiveName
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tarWriter, file)
		return err
	})
}

func samePath(a string, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
