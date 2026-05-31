package cli

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"filepilot/internal/backend"
	"filepilot/internal/cacheclean"
	"filepilot/internal/config"
	"filepilot/internal/doctor"
	"filepilot/internal/fperrors"
	"filepilot/internal/history"
	"filepilot/internal/output"
	"filepilot/internal/packaging"
	"filepilot/internal/paths"
	"filepilot/internal/session"
)

const (
	exitOK       = 0
	exitArgument = 2
)

type globalOptions struct {
	json    bool
	verbose bool
	quiet   bool
}

var newBaseContext = func() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

type command struct {
	name        string
	usage       string
	description string
	run         func(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int
	children    map[string]command
}

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, rest, err := parseGlobalFlags(args)
	if err != nil {
		fpErr := fperrors.New(fperrors.InvalidArgument, err.Error(), "Run filepilot --help.")
		return writeError(opts, stdout, stderr, "cli", fpErr)
	}

	commands := rootCommands()
	if len(rest) == 0 {
		printRootHelp(stdout)
		return exitOK
	}

	if isHelp(rest[0]) {
		printRootHelp(stdout)
		return exitOK
	}

	name := rest[0]
	cmd, ok := commands[name]
	if !ok {
		fpErr := fperrors.New(fperrors.InvalidArgument, fmt.Sprintf("Unknown command %q.", name), "Run filepilot --help.")
		return writeError(opts, stdout, stderr, "cli", fpErr)
	}

	return dispatch(cmd, rest[1:], opts, stdout, stderr)
}

func dispatch(cmd command, args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	opts, rest, err := parseGlobalFlagsWith(opts, args)
	if err != nil {
		fpErr := fperrors.New(fperrors.InvalidArgument, err.Error(), fmt.Sprintf("Run %s --help.", cmd.usage))
		return writeError(opts, stdout, stderr, cmd.name, fpErr)
	}

	if len(rest) > 0 && isHelp(rest[0]) {
		printCommandHelp(stdout, cmd)
		return exitOK
	}

	if len(rest) > 0 && len(cmd.children) > 0 {
		child, ok := cmd.children[rest[0]]
		if !ok {
			fpErr := fperrors.New(fperrors.InvalidArgument, fmt.Sprintf("Unknown command %q for %q.", rest[0], cmd.name), fmt.Sprintf("Run %s --help.", cmd.usage))
			return writeError(opts, stdout, stderr, cmd.name, fpErr)
		}
		return dispatch(child, rest[1:], opts, stdout, stderr)
	}

	if cmd.run == nil {
		printCommandHelp(stdout, cmd)
		return exitOK
	}
	return cmd.run(rest, opts, stdout, stderr)
}

func rootCommands() map[string]command {
	return map[string]command{
		"send": {
			name:        "send",
			usage:       "filepilot send <path>",
			description: "Start a FilePilot send attempt.",
			run:         runSend,
		},
		"receive": {
			name:        "receive",
			usage:       "filepilot receive [session-id]",
			description: "Receive a payload by FilePilot Session ID.",
			run:         runReceive,
		},
		"pack": {
			name:        "pack",
			usage:       "filepilot pack <dir>",
			description: "Create a FilePilot Directory Package.",
			run:         runPack,
		},
		"doctor": {
			name:        "doctor",
			usage:       "filepilot doctor",
			description: "Run local FilePilot diagnostics.",
			run:         runDoctor,
		},
		"clean": {
			name:        "clean",
			usage:       "filepilot clean",
			description: "Clean FilePilot-owned cache files.",
			run:         runClean,
		},
		"config": {
			name:        "config",
			usage:       "filepilot config <show|set>",
			description: "Show or update FilePilot configuration.",
			children: map[string]command{
				"show": {
					name:        "show",
					usage:       "filepilot config show",
					description: "Show effective FilePilot configuration.",
					run:         runConfigShow,
				},
				"set": {
					name:        "set",
					usage:       "filepilot config set <key> <value>",
					description: "Set a FilePilot configuration value.",
					run:         runConfigSet,
				},
			},
		},
	}
}

func placeholder(mode string) func(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	return func(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
		for _, arg := range args {
			if strings.HasPrefix(arg, "--") {
				fpErr := fperrors.New(fperrors.InvalidArgument, fmt.Sprintf("Unknown option %q.", arg), fmt.Sprintf("Run filepilot %s --help.", mode))
				return writeError(opts, stdout, stderr, mode, fpErr)
			}
		}
		if opts.json {
			_ = output.WriteJSON(stdout, output.Success(mode, "not_implemented", nil))
			return exitOK
		}
		if !opts.quiet {
			fmt.Fprintf(stdout, "FilePilot %s is registered but not implemented yet.\n", mode)
		}
		return exitOK
	}
}

func runSend(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	state, fpErr := loadConfigState()
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "send", fpErr)
	}
	opts.json = opts.json || state.Config.Values.JSONOutput

	sourcePath, transferOpts, fpErr := parseSendArgs(args)
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "send", fpErr)
	}
	payload, fpErr := prepareSendPayload(sourcePath, state)
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "send", fpErr)
	}
	defer cleanupSendPayload(payload, state.Config.Values.KeepPackages)

	executable, _ := os.Executable()
	resolved, fpErr := backend.Resolve(backend.ResolveRequest{
		ConfiguredPath: state.Config.Values.BackendPath,
		BundledDir:     backend.DefaultBundledDir(executable),
		PathDirs:       backend.PathDirsFromEnv(os.Getenv("PATH")),
	})
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "send", fpErr)
	}

	sessionID, err := session.Generate()
	if err != nil {
		fpErr := fperrors.New(fperrors.ConfigError, err.Error(), "Retry the command.")
		return writeError(opts, stdout, stderr, "send", fpErr)
	}

	transferBackend := backend.NewCrocBackend(resolved.Path)
	ctx, cancel := transferContext(transferOpts.Timeout)
	defer cancel()
	started := time.Now()
	if !opts.quiet && !opts.json {
		fmt.Fprintln(stdout, "FilePilot send is ready.")
		fmt.Fprintf(stdout, "Receiver command: filepilot receive %s\n", sessionID)
		fmt.Fprintln(stdout, "Waiting for receiver...")
	}

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
		fpErr = transferErrorFor("send", err)
		attempt.Status = historyStatusFor(fpErr)
		attempt.ErrorCode = string(fpErr.Code)
		attempt.ErrorMessage = fpErr.Message
		_ = history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt)
		return writeError(opts, stdout, stderr, "send", fpErr)
	}
	if historyErr := history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt); historyErr != nil {
		fpErr = fperrors.New(fperrors.ConfigError, historyErr.Error(), "Check that the FilePilot log directory is writable.")
		return writeError(opts, stdout, stderr, "send", fpErr)
	}

	if opts.json {
		return writeSendJSON(stdout, sessionID, attempt, resolved)
	}
	if !opts.quiet {
		fmt.Fprintln(stdout, "FilePilot send completed.")
	}
	return exitOK
}

type transferOptions struct {
	Timeout time.Duration
}

func parseSendArgs(args []string) (string, transferOptions, *fperrors.Error) {
	var sourcePath string
	var opts transferOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--timeout" {
			if i+1 >= len(args) {
				return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, "--timeout requires a duration.", "Run filepilot send <path> --timeout 5m.")
			}
			duration, err := time.ParseDuration(args[i+1])
			if err != nil || duration <= 0 {
				return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, "--timeout must be a positive duration such as 5m.", "Run filepilot send <path> --timeout 5m.")
			}
			opts.Timeout = duration
			i++
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, fmt.Sprintf("Unknown option %q.", arg), "Run filepilot send --help.")
		}
		if sourcePath != "" {
			return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, "send accepts exactly one source path.", "Run filepilot send <path>.")
		}
		sourcePath = arg
	}
	if sourcePath == "" {
		return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, "send requires a source path.", "Run filepilot send <path>.")
	}
	return filepath.Clean(sourcePath), opts, nil
}

type sendPayload struct {
	PayloadType string
	SendPath    string
	PackagePath string
	FileSize    int64
	Packed      bool
}

func prepareSendPayload(sourcePath string, state configState) (sendPayload, *fperrors.Error) {
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

func writeSendJSON(stdout io.Writer, sessionID string, attempt history.Attempt, resolved backend.Resolved) int {
	fields := map[string]any{
		"session_id":          sessionID,
		"session_id_redacted": history.RedactSessionID(sessionID),
		"payload_type":        attempt.PayloadType,
		"input_path":          attempt.InputPath,
		"package_path":        attempt.PackagePath,
		"file_size":           attempt.FileSize,
		"backend": map[string]any{
			"name":   attempt.Backend,
			"source": resolved.Source,
		},
	}
	if err := output.WriteJSON(stdout, output.Success("send", "completed", fields)); err != nil {
		return 1
	}
	return exitOK
}

func transferContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	base, stop := newBaseContext()
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(base, timeout)
		return ctx, func() {
			cancel()
			stop()
		}
	}
	return base, stop
}

func transferErrorFor(mode string, err error) *fperrors.Error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fperrors.New(fperrors.LocalTimeout, fmt.Sprintf("FilePilot %s reached the local timeout.", mode), "Increase --timeout or retry when the receiver is ready.")
	}
	if errors.Is(err, context.Canceled) {
		return fperrors.New(fperrors.Cancelled, fmt.Sprintf("FilePilot %s was cancelled.", mode), "Retry when you are ready to transfer again.")
	}
	return fperrors.New(fperrors.TransferFailed, fmt.Sprintf("FilePilot %s failed before the transfer completed.", mode), "Check the session ID, backend availability, and network access.")
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

func runReceive(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	state, fpErr := loadConfigState()
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}
	opts.json = opts.json || state.Config.Values.JSONOutput

	sessionID, transferOpts, fpErr := parseReceiveArgs(args, opts, stderr)
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}
	if err := session.Validate(sessionID); err != nil {
		fpErr := fperrors.New(fperrors.InvalidSessionID, err.Error(), "Use the FilePilot Session ID shown by the sender.")
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}
	if err := os.MkdirAll(state.Paths.DownloadDir, 0o755); err != nil {
		fpErr := fperrors.New(fperrors.DownloadDirUnwritable, err.Error(), "Choose a writable download_dir in FilePilot config.")
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}
	before, err := snapshotDownloadDir(state.Paths.DownloadDir)
	if err != nil {
		fpErr := fperrors.New(fperrors.DownloadDirUnwritable, err.Error(), "Check that the FilePilot download directory is readable.")
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}

	executable, _ := os.Executable()
	resolved, fpErr := backend.Resolve(backend.ResolveRequest{
		ConfiguredPath: state.Config.Values.BackendPath,
		BundledDir:     backend.DefaultBundledDir(executable),
		PathDirs:       backend.PathDirsFromEnv(os.Getenv("PATH")),
	})
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}

	transferBackend := backend.NewCrocBackend(resolved.Path)
	ctx, cancel := transferContext(transferOpts.Timeout)
	defer cancel()
	started := time.Now()
	if !opts.quiet && !opts.json {
		fmt.Fprintln(stdout, "FilePilot receive is ready.")
	}
	err = transferBackend.Receive(ctx, backend.ReceiveRequest{
		SessionID: sessionID,
		OutputDir: state.Paths.DownloadDir,
	})
	duration := time.Since(started)
	if err != nil {
		fpErr = transferErrorFor("receive", err)
		attempt := history.Attempt{
			Mode:          "receive",
			Status:        historyStatusFor(fpErr),
			PayloadType:   "unknown",
			OutputPath:    state.Paths.DownloadDir,
			Backend:       transferBackend.Name(),
			BackendSource: string(resolved.Source),
			SessionID:     sessionID,
			Duration:      duration,
			ErrorCode:     string(fpErr.Code),
			ErrorMessage:  fpErr.Message,
		}
		_ = history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt)
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}

	received, fpErr := summarizeReceivedPayload(state.Paths.DownloadDir, before, state.Config.Values.AutoUnpack)
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "receive", fpErr)
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
		SessionID:     sessionID,
		Duration:      duration,
	}
	if historyErr := history.NewWriter(history.DefaultPath(state.Paths.LogDir)).Append(attempt); historyErr != nil {
		fpErr = fperrors.New(fperrors.ConfigError, historyErr.Error(), "Check that the FilePilot log directory is writable.")
		return writeError(opts, stdout, stderr, "receive", fpErr)
	}

	if opts.json {
		return writeReceiveJSON(stdout, sessionID, attempt, resolved)
	}
	if !opts.quiet {
		fmt.Fprintln(stdout, "FilePilot receive completed.")
		fmt.Fprintf(stdout, "Saved: %s\n", received.OutputPath)
	}
	return exitOK
}

func parseReceiveArgs(args []string, opts globalOptions, stderr io.Writer) (string, transferOptions, *fperrors.Error) {
	var sessionID string
	var transferOpts transferOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--timeout" {
			if i+1 >= len(args) {
				return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, "--timeout requires a duration.", "Run filepilot receive <session-id> --timeout 5m.")
			}
			duration, err := time.ParseDuration(args[i+1])
			if err != nil || duration <= 0 {
				return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, "--timeout must be a positive duration such as 5m.", "Run filepilot receive <session-id> --timeout 5m.")
			}
			transferOpts.Timeout = duration
			i++
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, fmt.Sprintf("Unknown option %q.", arg), "Run filepilot receive --help.")
		}
		if sessionID != "" {
			return "", transferOptions{}, fperrors.New(fperrors.InvalidArgument, "receive accepts at most one FilePilot Session ID.", "Run filepilot receive <session-id>.")
		}
		sessionID = arg
	}
	if sessionID == "" {
		if opts.json {
			return "", transferOptions{}, fperrors.New(fperrors.MissingSessionID, "receive requires a FilePilot Session ID in JSON mode.", "Run filepilot receive <session-id> --json.")
		}
		fmt.Fprint(stderr, "FilePilot Session ID: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", transferOptions{}, fperrors.New(fperrors.MissingSessionID, "receive requires a FilePilot Session ID.", "Run filepilot receive <session-id>.")
		}
		sessionID = strings.TrimSpace(line)
		if sessionID == "" {
			return "", transferOptions{}, fperrors.New(fperrors.MissingSessionID, "receive requires a FilePilot Session ID.", "Run filepilot receive <session-id>.")
		}
	}
	return sessionID, transferOpts, nil
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
		rel := strings.TrimPrefix(name, sourcePrefix)
		if rel == "" || rel == "." {
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
	return err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func writeReceiveJSON(stdout io.Writer, sessionID string, attempt history.Attempt, resolved backend.Resolved) int {
	fields := map[string]any{
		"session_id":          sessionID,
		"session_id_redacted": history.RedactSessionID(sessionID),
		"payload_type":        attempt.PayloadType,
		"output_path":         attempt.OutputPath,
		"package_path":        attempt.PackagePath,
		"file_size":           attempt.FileSize,
		"unpacked":            attempt.Unpacked,
		"backend": map[string]any{
			"name":   attempt.Backend,
			"source": resolved.Source,
		},
	}
	if err := output.WriteJSON(stdout, output.Success("receive", "completed", fields)); err != nil {
		return 1
	}
	return exitOK
}

func runPack(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	state, fpErr := loadConfigState()
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "pack", fpErr)
	}
	opts.json = opts.json || state.Config.Values.JSONOutput

	sourceDir, outputPath, fpErr := parsePackArgs(args, state.Paths.CacheDir)
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "pack", fpErr)
	}

	result, err := packaging.CreateDirectoryPackage(packaging.Request{
		SourceDir:  sourceDir,
		OutputPath: outputPath,
	})
	if err != nil {
		fpErr := fperrors.New(fperrors.PackFailed, err.Error(), "Check that the source directory is readable and the output path is writable.")
		return writeError(opts, stdout, stderr, "pack", fpErr)
	}

	if opts.json {
		return writePackJSON(stdout, result)
	}
	if !opts.quiet {
		fmt.Fprintf(stdout, "Created FilePilot Directory Package: %s\n", result.PackagePath)
	}
	return exitOK
}

func parsePackArgs(args []string, cacheDir string) (string, string, *fperrors.Error) {
	var sourceDir string
	var outputPath string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--output":
			if i+1 >= len(args) {
				return "", "", fperrors.New(fperrors.InvalidArgument, "--output requires a path.", "Run filepilot pack <dir> --output <path>.")
			}
			outputPath = args[i+1]
			i++
		default:
			if strings.HasPrefix(arg, "--") {
				return "", "", fperrors.New(fperrors.InvalidArgument, fmt.Sprintf("Unknown option %q.", arg), "Run filepilot pack --help.")
			}
			if sourceDir != "" {
				return "", "", fperrors.New(fperrors.InvalidArgument, "pack accepts exactly one source directory.", "Run filepilot pack <dir>.")
			}
			sourceDir = arg
		}
	}
	if sourceDir == "" {
		return "", "", fperrors.New(fperrors.InvalidArgument, "pack requires a source directory.", "Run filepilot pack <dir>.")
	}
	if outputPath == "" {
		outputPath = defaultPackagePath(cacheDir, sourceDir)
	}
	return sourceDir, outputPath, nil
}

func defaultPackagePath(cacheDir string, sourceDir string) string {
	name := filepath.Base(filepath.Clean(sourceDir))
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	return filepath.Join(cacheDir, fmt.Sprintf("%s-%s.tar.gz", name, stamp))
}

func writePackJSON(stdout io.Writer, result packaging.Result) int {
	fields := map[string]any{
		"package_path": result.PackagePath,
		"payload_type": result.Manifest.PayloadType,
		"manifest": map[string]any{
			"schema_version": result.Manifest.SchemaVersion,
			"payload_type":   result.Manifest.PayloadType,
			"source_name":    result.Manifest.SourceName,
			"created_by":     result.Manifest.CreatedBy,
			"created_at":     result.Manifest.CreatedAt,
		},
	}
	if err := output.WriteJSON(stdout, output.Success("pack", "completed", fields)); err != nil {
		return 1
	}
	return exitOK
}

func runDoctor(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		fpErr := fperrors.New(fperrors.InvalidArgument, "doctor does not accept arguments.", "Run filepilot doctor.")
		return writeError(opts, stdout, stderr, "doctor", fpErr)
	}
	state, fpErr := loadConfigState()
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "doctor", fpErr)
	}
	opts.json = opts.json || state.Config.Values.JSONOutput

	executable, _ := os.Executable()
	report := doctor.Run(doctor.Request{
		Paths: state.Paths,
		Backend: backend.ResolveRequest{
			ConfiguredPath: state.Config.Values.BackendPath,
			BundledDir:     backend.DefaultBundledDir(executable),
			PathDirs:       backend.PathDirsFromEnv(os.Getenv("PATH")),
		},
		Getenv: os.Getenv,
	})

	if opts.json {
		return writeDoctorJSON(stdout, report)
	}
	writeDoctorHuman(stdout, report)
	if report.Fatal != nil {
		return report.Fatal.ExitCode()
	}
	return exitOK
}

func writeDoctorJSON(stdout io.Writer, report doctor.Report) int {
	fields := doctorFields(report)
	if report.Fatal != nil {
		if err := output.WriteJSON(stdout, output.Result{
			OK:     false,
			Status: "failed",
			Mode:   "doctor",
			Error:  report.Fatal,
			Fields: fields,
		}); err != nil {
			return 1
		}
		return report.Fatal.ExitCode()
	}
	if err := output.WriteJSON(stdout, output.Success("doctor", "completed", fields)); err != nil {
		return 1
	}
	return exitOK
}

func doctorFields(report doctor.Report) map[string]any {
	return map[string]any{
		"version":          report.Version,
		"os":               report.OS,
		"arch":             report.Arch,
		"config_path":      report.ConfigPath,
		"backend":          report.Backend,
		"directory_checks": report.DirectoryChecks,
		"warnings":         report.Warnings,
		"fatal":            report.Fatal,
	}
}

func writeDoctorHuman(stdout io.Writer, report doctor.Report) {
	fmt.Fprintf(stdout, "FilePilot version: %s\n", report.Version)
	fmt.Fprintf(stdout, "Platform: %s/%s\n", report.OS, report.Arch)
	fmt.Fprintf(stdout, "Config path: %s\n", report.ConfigPath)
	if report.Backend.Path != "" {
		fmt.Fprintf(stdout, "Backend source: %s\n", report.Backend.Source)
		fmt.Fprintf(stdout, "Backend path: %s\n", report.Backend.Path)
		fmt.Fprintf(stdout, "Backend version: %s\n", report.Backend.Version)
	}
	for _, check := range report.DirectoryChecks {
		status := "not writable"
		if check.Writable {
			status = "writable"
		}
		fmt.Fprintf(stdout, "Directory %s: %s (%s)\n", check.Name, status, check.Path)
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "Warning %s: %s\n", warning.Code, warning.Message)
	}
	if report.Fatal != nil {
		fmt.Fprintf(stdout, "Fatal %s: %s\n", report.Fatal.Code, report.Fatal.Message)
	}
}

func runClean(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	state, fpErr := loadConfigState()
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "clean", fpErr)
	}
	opts.json = opts.json || state.Config.Values.JSONOutput

	cleanOpts, fpErr := parseCleanArgs(args)
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "clean", fpErr)
	}
	result, err := cacheclean.Clean(state.Paths.CacheDir, cleanOpts)
	if err != nil {
		fpErr := fperrors.New(fperrors.CacheDirUnwritable, err.Error(), "Check that FILEPILOT_CACHE_DIR points to a writable FilePilot cache directory.")
		return writeError(opts, stdout, stderr, "clean", fpErr)
	}
	if opts.json {
		return writeCleanJSON(stdout, result)
	}
	if !opts.quiet {
		writeCleanHuman(stdout, result)
	}
	return exitOK
}

func parseCleanArgs(args []string) (cacheclean.Options, *fperrors.Error) {
	var opts cacheclean.Options
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			opts.DryRun = true
		case "--older-than":
			if i+1 >= len(args) {
				return cacheclean.Options{}, fperrors.New(fperrors.InvalidArgument, "--older-than requires a duration.", "Run filepilot clean --older-than 24h.")
			}
			duration, err := time.ParseDuration(args[i+1])
			if err != nil || duration < 0 {
				return cacheclean.Options{}, fperrors.New(fperrors.InvalidArgument, "--older-than must be a non-negative duration such as 24h.", "Run filepilot clean --older-than 24h.")
			}
			opts.OlderThan = duration
			i++
		default:
			if strings.HasPrefix(arg, "--") {
				return cacheclean.Options{}, fperrors.New(fperrors.InvalidArgument, fmt.Sprintf("Unknown option %q.", arg), "Run filepilot clean --help.")
			}
			return cacheclean.Options{}, fperrors.New(fperrors.InvalidArgument, "clean does not accept path arguments.", "Run filepilot clean or filepilot clean --dry-run.")
		}
	}
	return opts, nil
}

func writeCleanHuman(stdout io.Writer, result cacheclean.Result) {
	items := result.Removed
	verb := "Removed"
	if result.DryRun {
		items = result.Planned
		verb = "Would remove"
	}
	noun := "files"
	if len(items) == 1 {
		noun = "file"
	}
	fmt.Fprintf(stdout, "%s %d FilePilot cache %s.\n", verb, len(items), noun)
	for _, item := range items {
		fmt.Fprintf(stdout, "%s\n", item.Path)
	}
}

func writeCleanJSON(stdout io.Writer, result cacheclean.Result) int {
	fields := map[string]any{
		"cache_dir": result.CacheDir,
		"dry_run":   result.DryRun,
		"planned":   result.Planned,
		"removed":   result.Removed,
		"skipped":   result.Skipped,
	}
	if err := output.WriteJSON(stdout, output.Success("clean", "completed", fields)); err != nil {
		return 1
	}
	return exitOK
}

func runConfigShow(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		fpErr := fperrors.New(fperrors.InvalidArgument, "config show does not accept arguments.", "Run filepilot config show.")
		return writeError(opts, stdout, stderr, "config show", fpErr)
	}

	state, fpErr := loadConfigState()
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "config show", fpErr)
	}
	opts.json = opts.json || state.Config.Values.JSONOutput
	if opts.json {
		return writeConfigJSON(stdout, state)
	}
	writeConfigHuman(stdout, state)
	return exitOK
}

func runConfigSet(args []string, opts globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 2 {
		fpErr := fperrors.New(fperrors.InvalidArgument, "config set requires a key and value.", "Run filepilot config set <key> <value>.")
		return writeError(opts, stdout, stderr, "config set", fpErr)
	}

	state, fpErr := loadConfigState()
	if fpErr != nil {
		return writeError(opts, stdout, stderr, "config set", fpErr)
	}
	next, err := config.Set(state.Config.Values, args[0], args[1])
	if err != nil {
		fpErr := fperrors.New(fperrors.InvalidArgument, err.Error(), "Use a supported FilePilot config key.")
		return writeError(opts, stdout, stderr, "config set", fpErr)
	}
	if err := config.Save(state.Paths.ConfigPath, next); err != nil {
		fpErr := fperrors.New(fperrors.ConfigError, err.Error(), "Check that the FilePilot config directory is writable.")
		return writeError(opts, stdout, stderr, "config set", fpErr)
	}
	state.Config.Values = next
	opts.json = opts.json || next.JSONOutput
	if opts.json {
		return writeConfigJSON(stdout, state)
	}
	if !opts.quiet {
		fmt.Fprintf(stdout, "Updated %s in FilePilot config.\n", args[0])
	}
	return exitOK
}

type configState struct {
	Paths  paths.Paths
	Config config.Effective
}

func loadConfigState() (configState, *fperrors.Error) {
	resolved, err := paths.Current()
	if err != nil {
		return configState{}, fperrors.New(fperrors.ConfigError, err.Error(), "Check your platform profile environment.")
	}
	effective, err := config.Load(resolved.ConfigPath, resolved.DownloadDir)
	if err != nil {
		return configState{}, fperrors.New(fperrors.ConfigError, err.Error(), "Fix config.toml or choose another FILEPILOT_CONFIG path.")
	}
	resolved.DownloadDir = effective.Values.DownloadDir
	return configState{Paths: resolved, Config: effective}, nil
}

func writeConfigJSON(stdout io.Writer, state configState) int {
	payload := map[string]any{
		"backend_path":  state.Config.Values.BackendPath,
		"download_dir":  state.Config.Values.DownloadDir,
		"auto_unpack":   state.Config.Values.AutoUnpack,
		"keep_packages": state.Config.Values.KeepPackages,
		"json_output":   state.Config.Values.JSONOutput,
		"config_path":   state.Paths.ConfigPath,
		"cache_dir":     state.Paths.CacheDir,
		"log_dir":       state.Paths.LogDir,
	}
	if err := output.WriteJSON(stdout, output.Success("config show", "completed", payload)); err != nil {
		return 1
	}
	return exitOK
}

func writeConfigHuman(stdout io.Writer, state configState) {
	fmt.Fprintf(stdout, "backend_path: %s\n", state.Config.Values.BackendPath)
	fmt.Fprintf(stdout, "download_dir: %s\n", state.Config.Values.DownloadDir)
	fmt.Fprintf(stdout, "auto_unpack: %t\n", state.Config.Values.AutoUnpack)
	fmt.Fprintf(stdout, "keep_packages: %t\n", state.Config.Values.KeepPackages)
	fmt.Fprintf(stdout, "json_output: %t\n", state.Config.Values.JSONOutput)
	fmt.Fprintf(stdout, "config_path: %s\n", state.Paths.ConfigPath)
	fmt.Fprintf(stdout, "cache_dir: %s\n", state.Paths.CacheDir)
	fmt.Fprintf(stdout, "log_dir: %s\n", state.Paths.LogDir)
}

func writeError(opts globalOptions, stdout io.Writer, stderr io.Writer, mode string, err *fperrors.Error) int {
	if opts.json {
		if writeErr := output.WriteJSON(stdout, output.Failure(mode, err)); writeErr != nil {
			return 1
		}
		return err.ExitCode()
	}
	fmt.Fprintln(stderr, err.Message)
	if err.Hint != "" {
		fmt.Fprintln(stderr, "Hint:", err.Hint)
	}
	return err.ExitCode()
}

func parseGlobalFlags(args []string) (globalOptions, []string, error) {
	return parseGlobalFlagsWith(globalOptions{}, args)
}

func parseGlobalFlagsWith(opts globalOptions, args []string) (globalOptions, []string, error) {
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--json":
			opts.json = true
		case "--verbose":
			opts.verbose = true
		case "--quiet":
			opts.quiet = true
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

func isHelp(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "help"
}

func printRootHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "FilePilot moves files and directories through a replaceable transfer backend.")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Usage: filepilot <command> [options]")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintln(stdout, "  send       Start a FilePilot send attempt")
	fmt.Fprintln(stdout, "  receive    Receive a payload by FilePilot Session ID")
	fmt.Fprintln(stdout, "  pack       Create a FilePilot Directory Package")
	fmt.Fprintln(stdout, "  doctor     Run local FilePilot diagnostics")
	fmt.Fprintln(stdout, "  clean      Clean FilePilot-owned cache files")
	fmt.Fprintln(stdout, "  config     Show or update FilePilot configuration")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Global Flags:")
	fmt.Fprintln(stdout, "  --json       Write machine-readable JSON")
	fmt.Fprintln(stdout, "  --verbose    Write additional human diagnostics")
	fmt.Fprintln(stdout, "  --quiet      Suppress non-essential human output")
}

func printCommandHelp(stdout io.Writer, cmd command) {
	fmt.Fprintf(stdout, "Usage: %s\n", cmd.usage)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, cmd.description)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Global Flags:")
	fmt.Fprintln(stdout, "  --json")
	fmt.Fprintln(stdout, "  --verbose")
	fmt.Fprintln(stdout, "  --quiet")
	if len(cmd.children) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Subcommands:")
		for _, child := range cmd.children {
			fmt.Fprintf(stdout, "  %-8s %s\n", child.name, child.description)
		}
	}
}
