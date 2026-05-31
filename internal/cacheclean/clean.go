package cacheclean

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

type Options struct {
	DryRun    bool
	OlderThan time.Duration
	Now       time.Time
}

type Result struct {
	CacheDir string
	DryRun   bool
	Removed  []Item
	Planned  []Item
	Skipped  []Item
}

type Item struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	MTime string `json:"mtime"`
}

func Clean(cacheDir string, opts Options) (Result, error) {
	if cacheDir == "" {
		return Result{}, fmt.Errorf("cache directory is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	root, err := filepath.Abs(cacheDir)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Result{}, err
	}

	result := Result{CacheDir: root, DryRun: opts.DryRun}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		item := Item{
			Path:  filepath.Clean(path),
			Size:  info.Size(),
			MTime: info.ModTime().UTC().Format(time.RFC3339),
		}
		if opts.OlderThan > 0 && opts.Now.Sub(info.ModTime()) < opts.OlderThan {
			result.Skipped = append(result.Skipped, item)
			return nil
		}
		owned, err := isFilePilotPackage(path)
		if err != nil {
			result.Skipped = append(result.Skipped, item)
			return nil
		}
		if !owned {
			result.Skipped = append(result.Skipped, item)
			return nil
		}
		if !isWithinRoot(root, path) {
			return fmt.Errorf("refusing to delete path outside FilePilot cache root: %s", path)
		}
		if opts.DryRun {
			result.Planned = append(result.Planned, item)
			return nil
		}
		if err := deleteItem(root, item); err != nil {
			return err
		}
		result.Removed = append(result.Removed, item)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func deleteItem(root string, item Item) error {
	if !isWithinRoot(root, item.Path) {
		return fmt.Errorf("refusing to delete path outside FilePilot cache root: %s", item.Path)
	}
	return os.Remove(item.Path)
}

func isFilePilotPackage(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return false, nil
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if header.Name != ".filepilot/manifest.json" || header.Typeflag != tar.TypeReg {
			continue
		}
		var manifest struct {
			PayloadType string `json:"payload_type"`
			CreatedBy   string `json:"created_by"`
		}
		if err := json.NewDecoder(tarReader).Decode(&manifest); err != nil {
			return false, err
		}
		return manifest.PayloadType == PayloadTypeDirectoryPackage && manifest.CreatedBy == "filepilot", nil
	}
}

func isWithinRoot(root string, path string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
