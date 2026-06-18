package transfer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filepilot/internal/backend"
	"filepilot/internal/config"
	"filepilot/internal/fperrors"
	"filepilot/internal/history"
	"filepilot/internal/packaging"
	"filepilot/internal/paths"
	"filepilot/internal/session"
)

type EventType string

const (
	EventStarted      EventType = "started"
	EventWaiting      EventType = "waiting"
	EventTransferring EventType = "transferring"
	EventCompleted    EventType = "completed"
	EventError        EventType = "error"
)

type Event struct {
	Type      EventType `json:"type"`
	Message   string    `json:"message,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	Path      string    `json:"path,omitempty"`
	ErrorCode string    `json:"error_code,omitempty"`
}

type EventHandler func(Event)

type SendOptions struct {
	SourcePath string
	Timeout    time.Duration
	OnEvent    EventHandler
}

type ReceiveOptions struct {
	SessionID string
	OutputDir string
	Timeout   time.Duration
	OnEvent   EventHandler
}

type SendResult struct {
	SessionID string
	Attempt   history.Attempt
	Backend   backend.Resolved
}

type ReceiveResult struct {
	SessionID string
	Attempt   history.Attempt
	Backend   backend.Resolved
}

func Send(ctx context.Context, opts SendOptions) (SendResult, *fperrors.Error) {
	emit(opts.OnEvent, Event{Type: EventStarted, Message: "Preparing send."})
	state, fpErr := loadState()
	if fpErr != nil {
		emitError(opts.OnEvent, fpErr)
		return SendResult{}, fpErr
	}
	sourcePath := filepath.Clean(opts.SourcePath)
	payload, fpErr := prepareSendPayload(sourcePath, state)
	if fpErr != nil {
		emitError(opts.OnEvent, fpErr)
		return SendResult{}, fpErr
	}
	defer cleanupSendPayload(payload, state.Config.Values.KeepPackages)

	resolved, fpErr := resolveBackend(state)
	if fpErr != nil {
		emitError(opts.OnEvent, fpErr)
		return SendResult{}, fpErr
	}
	sessionID, err := session.Generate()
	if err != nil {
		fpErr := fperrors.New(fperrors.ConfigError, err.Error(), "Retry the command.")
		emitError(opts.OnEvent, fpErr)
		return SendResult{}, fpErr
	}

	transferBackend := backend.NewCrocBackend(resolved.Path)
	ctx, cancel := withTimeout(ctx, opts.Timeout)
	defer cancel()
	started := time.Now()
	emit(opts.OnEvent, Event{Type: EventWaiting, Message: "Waiting for receiver.", SessionID: sessionID, Path: sourcePath})
	err = transferBackend.Send(ctx, backend.SendRequest{
		InputPath: payload.SendPath,
		SessionID: sessionID,
	})
	duration := time.Since(started)
	attempt := history.Attempt{
		Mode:          "send",
		Status:        "completed",
		PayloadType:   payload.PayloadType,
		InputPath:     sourcePath,
		PackagePath:   payload.PackagePath,
		FileSize:      payload.FileSize,
		Packed:        payload.Packed,
		Unpacked:      false,
		Backend:       transferBackend.Name(),
		BackendSource: string(resolved.Source),
		SessionID:     sessionID,
		Duration:      duration,
	}
	if err != nil {
		fpErr = errorFor("send", err)
		attempt.Status = historyStatusFor(fpErr)
		attempt.ErrorCode = string(fpErr.Code)
		attempt.ErrorMessage = fpErr.Message
		_ = history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt)
		emitError(opts.OnEvent, fpErr)
		return SendResult{SessionID: sessionID, Attempt: attempt, Backend: resolved}, fpErr
	}
	if historyErr := history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt); historyErr != nil {
		fpErr = fperrors.New(fperrors.ConfigError, historyErr.Error(), "Check that the FilePilot log directory is writable.")
		emitError(opts.OnEvent, fpErr)
		return SendResult{SessionID: sessionID, Attempt: attempt, Backend: resolved}, fpErr
	}
	emit(opts.OnEvent, Event{Type: EventCompleted, Message: "Send completed.", SessionID: sessionID, Path: sourcePath})
	return SendResult{SessionID: sessionID, Attempt: attempt, Backend: resolved}, nil
}

func Receive(ctx context.Context, opts ReceiveOptions) (ReceiveResult, *fperrors.Error) {
	emit(opts.OnEvent, Event{Type: EventStarted, Message: "Preparing receive.", SessionID: opts.SessionID})
	state, fpErr := loadState()
	if fpErr != nil {
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{}, fpErr
	}
	if err := session.Validate(opts.SessionID); err != nil {
		fpErr := fperrors.New(fperrors.InvalidSessionID, err.Error(), "Use the FilePilot Session ID shown by the sender.")
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{}, fpErr
	}
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = state.Paths.DownloadDir
	}
	state.Paths.DownloadDir = filepath.Clean(outputDir)
	if err := os.MkdirAll(state.Paths.DownloadDir, 0o755); err != nil {
		fpErr := fperrors.New(fperrors.DownloadDirUnwritable, err.Error(), "Choose a writable download directory.")
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{}, fpErr
	}
	before, err := snapshotDownloadDir(state.Paths.DownloadDir)
	if err != nil {
		fpErr := fperrors.New(fperrors.DownloadDirUnwritable, err.Error(), "Check that the FilePilot download directory is readable.")
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{}, fpErr
	}

	resolved, fpErr := resolveBackend(state)
	if fpErr != nil {
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{}, fpErr
	}
	transferBackend := backend.NewCrocBackend(resolved.Path)
	ctx, cancel := withTimeout(ctx, opts.Timeout)
	defer cancel()
	started := time.Now()
	emit(opts.OnEvent, Event{Type: EventTransferring, Message: "Receiving payload.", SessionID: opts.SessionID, Path: state.Paths.DownloadDir})
	err = transferBackend.Receive(ctx, backend.ReceiveRequest{
		SessionID: opts.SessionID,
		OutputDir: state.Paths.DownloadDir,
	})
	duration := time.Since(started)
	if err != nil {
		fpErr = errorFor("receive", err)
		attempt := history.Attempt{
			Mode:          "receive",
			Status:        historyStatusFor(fpErr),
			PayloadType:   "unknown",
			OutputPath:    state.Paths.DownloadDir,
			Backend:       transferBackend.Name(),
			BackendSource: string(resolved.Source),
			SessionID:     opts.SessionID,
			Duration:      duration,
			ErrorCode:     string(fpErr.Code),
			ErrorMessage:  fpErr.Message,
		}
		_ = history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt)
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{SessionID: opts.SessionID, Attempt: attempt, Backend: resolved}, fpErr
	}

	received, fpErr := summarizeReceivedPayload(state.Paths.DownloadDir, before, state.Config.Values.AutoUnpack)
	if fpErr != nil {
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{SessionID: opts.SessionID, Backend: resolved}, fpErr
	}
	attempt := history.Attempt{
		Mode:          "receive",
		Status:        "completed",
		PayloadType:   received.PayloadType,
		OutputPath:    received.OutputPath,
		PackagePath:   received.PackagePath,
		FileSize:      received.FileSize,
		Packed:        false,
		Unpacked:      received.Unpacked,
		Backend:       transferBackend.Name(),
		BackendSource: string(resolved.Source),
		SessionID:     opts.SessionID,
		Duration:      duration,
	}
	if historyErr := history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt); historyErr != nil {
		fpErr = fperrors.New(fperrors.ConfigError, historyErr.Error(), "Check that the FilePilot log directory is writable.")
		emitError(opts.OnEvent, fpErr)
		return ReceiveResult{SessionID: opts.SessionID, Attempt: attempt, Backend: resolved}, fpErr
	}
	emit(opts.OnEvent, Event{Type: EventCompleted, Message: "Receive completed.", SessionID: opts.SessionID, Path: received.OutputPath})
	return ReceiveResult{SessionID: opts.SessionID, Attempt: attempt, Backend: resolved}, nil
}

type state struct {
	Paths  paths.Paths
	Config config.Effective
}

func loadState() (state, *fperrors.Error) {
	resolved, err := paths.Current()
	if err != nil {
		return state{}, fperrors.New(fperrors.ConfigError, err.Error(), "Check your platform profile environment.")
	}
	effective, err := config.Load(resolved.ConfigPath, resolved.DownloadDir)
	if err != nil {
		return state{}, fperrors.New(fperrors.ConfigError, err.Error(), "Fix config.toml or choose another FILEPILOT_CONFIG path.")
	}
	resolved.DownloadDir = effective.Values.DownloadDir
	return state{Paths: resolved, Config: effective}, nil
}

func resolveBackend(state state) (backend.Resolved, *fperrors.Error) {
	executable, _ := os.Executable()
	return backend.Resolve(backend.ResolveRequest{
		ConfiguredPath: state.Config.Values.BackendPath,
		BundledDir:     backend.DefaultBundledDir(executable),
		PathDirs:       backend.PathDirsFromEnv(os.Getenv("PATH")),
	})
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

type sendPayload struct {
	PayloadType string
	SendPath    string
	PackagePath string
	FileSize    int64
	Packed      bool
}

func prepareSendPayload(sourcePath string, state state) (sendPayload, *fperrors.Error) {
	info, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		return sendPayload{}, fperrors.New(fperrors.PathNotFound, "Source path does not exist.", "Check the path and run filepilot send <path> again.")
	}
	if err != nil {
		return sendPayload{}, fperrors.New(fperrors.PermissionDenied, "Source path is not accessible.", "Check file permissions.")
	}
	if info.IsDir() {
		packagePath := defaultPackagePath(state.Paths.CacheDir, sourcePath)
		result, err := packaging.CreateDirectoryPackage(packaging.Request{
			SourceDir:  sourcePath,
			OutputPath: packagePath,
		})
		if err != nil {
			return sendPayload{}, fperrors.New(fperrors.PackFailed, err.Error(), "Check that the source directory is readable and the FilePilot cache directory is writable.")
		}
		packageInfo, err := os.Stat(result.PackagePath)
		if err != nil {
			return sendPayload{}, fperrors.New(fperrors.PackFailed, err.Error(), "Check that the FilePilot package was created in the cache directory.")
		}
		return sendPayload{
			PayloadType: packaging.PayloadTypeDirectoryPackage,
			SendPath:    result.PackagePath,
			PackagePath: result.PackagePath,
			FileSize:    packageInfo.Size(),
			Packed:      true,
		}, nil
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return sendPayload{}, fperrors.New(fperrors.PermissionDenied, "Source file is not readable.", "Check file permissions.")
	}
	_ = file.Close()
	return sendPayload{
		PayloadType: "file",
		SendPath:    sourcePath,
		FileSize:    info.Size(),
		Packed:      false,
	}, nil
}

func cleanupSendPayload(payload sendPayload, keepPackages bool) {
	if !payload.Packed || keepPackages || payload.PackagePath == "" {
		return
	}
	_ = os.Remove(payload.PackagePath)
}

func defaultPackagePath(cacheDir string, sourceDir string) string {
	name := filepath.Base(filepath.Clean(sourceDir))
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	return filepath.Join(cacheDir, fmt.Sprintf("%s-%s.tar.gz", name, stamp))
}

type receivedPayload struct {
	PayloadType string
	OutputPath  string
	PackagePath string
	FileSize    int64
	Unpacked    bool
}

func snapshotDownloadDir(downloadDir string) (map[string]bool, error) {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	snapshot := make(map[string]bool, len(entries))
	for _, entry := range entries {
		snapshot[entry.Name()] = true
	}
	return snapshot, nil
}

func summarizeReceivedPayload(downloadDir string, before map[string]bool, autoUnpack bool) (receivedPayload, *fperrors.Error) {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return receivedPayload{}, fperrors.New(fperrors.DownloadDirUnwritable, err.Error(), "Check that the FilePilot download directory is readable.")
	}
	var newPaths []string
	for _, entry := range entries {
		if before[entry.Name()] {
			continue
		}
		newPaths = append(newPaths, filepath.Join(downloadDir, entry.Name()))
	}
	if len(newPaths) == 0 {
		return receivedPayload{}, fperrors.New(fperrors.TransferFailed, "Transfer backend completed but no payload was found.", "Check the backend output directory.")
	}
	if len(newPaths) > 1 {
		return receivedPayload{}, fperrors.New(fperrors.TransferFailed, "Transfer backend produced multiple payloads.", "Receive one FilePilot payload at a time.")
	}
	path := newPaths[0]
	info, err := os.Stat(path)
	if err != nil {
		return receivedPayload{}, fperrors.New(fperrors.TransferFailed, err.Error(), "Check the received payload path.")
	}
	if autoUnpack {
		unpackedPath, ok, err := unpackFilePilotDirectoryPackage(path, downloadDir)
		if err != nil {
			return receivedPayload{}, fperrors.New(fperrors.TransferFailed, err.Error(), "Check that the received FilePilot Directory Package is valid.")
		}
		if ok {
			return receivedPayload{
				PayloadType: packaging.PayloadTypeDirectoryPackage,
				OutputPath:  unpackedPath,
				PackagePath: path,
				FileSize:    info.Size(),
				Unpacked:    true,
			}, nil
		}
	}
	return receivedPayload{
		PayloadType: "file",
		OutputPath:  path,
		FileSize:    info.Size(),
		Unpacked:    false,
	}, nil
}

func unpackFilePilotDirectoryPackage(packagePath string, downloadDir string) (string, bool, error) {
	manifest, ok, err := readDirectoryPackageManifest(packagePath)
	if err != nil || !ok {
		return "", ok, err
	}
	if manifest.PayloadType != packaging.PayloadTypeDirectoryPackage || manifest.CreatedBy != "filepilot" || manifest.SourceName == "" {
		return "", false, nil
	}
	destination := nonConflictingPath(downloadDir, manifest.SourceName)
	if err := extractDirectoryPackage(packagePath, destination, manifest.SourceName); err != nil {
		return "", true, err
	}
	return destination, true, nil
}

func readDirectoryPackageManifest(packagePath string) (packaging.Manifest, bool, error) {
	file, err := os.Open(packagePath)
	if err != nil {
		return packaging.Manifest{}, false, err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return packaging.Manifest{}, false, nil
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return packaging.Manifest{}, false, nil
		}
		if err != nil {
			return packaging.Manifest{}, false, err
		}
		if header.Name != ".filepilot/manifest.json" || header.Typeflag != tar.TypeReg {
			continue
		}
		var manifest packaging.Manifest
		if err := json.NewDecoder(tarReader).Decode(&manifest); err != nil {
			return packaging.Manifest{}, false, err
		}
		return manifest, true, nil
	}
}

func extractDirectoryPackage(packagePath string, destination string, sourceName string) error {
	file, err := os.Open(packagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destinationAbs, 0o755); err != nil {
		return err
	}

	sourcePrefix := filepath.ToSlash(filepath.Clean(sourceName)) + "/"
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Clean(header.Name))
		if name == ".filepilot/manifest.json" || strings.HasPrefix(name, ".filepilot/") {
			continue
		}
		if !strings.HasPrefix(name, sourcePrefix) {
			continue
		}
		rel, ok := cutPrefix(name, sourcePrefix)
		if !ok || rel == "" || rel == "." {
			continue
		}
		target := filepath.Join(destinationAbs, filepath.FromSlash(rel))
		if !isWithinDir(destinationAbs, target) {
			return fmt.Errorf("directory package contains an unsafe path")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				_ = file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
}

func nonConflictingPath(parentDir string, name string) string {
	candidate := filepath.Join(parentDir, filepath.Base(filepath.Clean(name)))
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	for index := 1; ; index++ {
		candidate = filepath.Join(parentDir, fmt.Sprintf("%s-%d", filepath.Base(filepath.Clean(name)), index))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func isWithinDir(parent string, child string) bool {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(parentAbs, childAbs)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func cutPrefix(value string, prefix string) (string, bool) {
	if len(value) < len(prefix) || value[:len(prefix)] != prefix {
		return "", false
	}
	return value[len(prefix):], true
}

func errorFor(mode string, err error) *fperrors.Error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fperrors.New(fperrors.LocalTimeout, fmt.Sprintf("FilePilot %s reached the local timeout.", mode), "Increase --timeout or retry when the receiver is ready.")
	}
	if errors.Is(err, context.Canceled) {
		return fperrors.New(fperrors.Cancelled, fmt.Sprintf("FilePilot %s was cancelled.", mode), "Retry when you are ready to transfer again.")
	}
	message := fmt.Sprintf("FilePilot %s failed before the transfer completed.", mode)
	if err != nil && err.Error() != "" {
		message = fmt.Sprintf("%s Backend reported: %s", message, err.Error())
	}
	return fperrors.New(fperrors.TransferFailed, message, "Check the session ID, backend availability, and network access.")
}

func historyStatusFor(err *fperrors.Error) string {
	if err == nil {
		return "completed"
	}
	switch err.Code {
	case fperrors.LocalTimeout:
		return "timeout"
	case fperrors.Cancelled:
		return "cancelled"
	default:
		return "failed"
	}
}

func emit(handler EventHandler, event Event) {
	if handler != nil {
		handler(event)
	}
}

func emitError(handler EventHandler, err *fperrors.Error) {
	if err != nil {
		emit(handler, Event{Type: EventError, Message: err.Message, ErrorCode: string(err.Code)})
	}
}
