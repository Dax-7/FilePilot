package desktop

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	"filepilot/internal/config"
	"filepilot/internal/fperrors"
	"filepilot/internal/paths"
	"filepilot/internal/transfer"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	mu  sync.Mutex
}

type Result struct {
	OK        bool   `json:"ok"`
	Status    string `json:"status"`
	SessionID string `json:"session_id,omitempty"`
	Path      string `json:"path,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Error     string `json:"error,omitempty"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) ChooseSendFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select file to send",
	})
}

func (a *App) ChooseSendFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select folder to send",
	})
}

func (a *App) ChooseReceiveFolder() (string, error) {
	defaultDir, _ := a.defaultReceiveFolder()
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select save folder",
		DefaultDirectory: defaultDir,
	})
}

func (a *App) DefaultReceiveFolder() Result {
	folder, err := a.defaultReceiveFolder()
	if err != nil {
		return failure(fperrors.ConfigError, err.Error())
	}
	return Result{
		OK:     true,
		Status: "completed",
		Path:   folder,
	}
}

func (a *App) StartSend(path string) Result {
	if path == "" {
		return failure(fperrors.InvalidArgument, "Choose a file or folder before sending.")
	}
	result, fpErr := transfer.Send(a.ctx, transfer.SendOptions{
		SourcePath: path,
		OnEvent:    a.emitTransferEvent,
	})
	if fpErr != nil {
		return failure(fpErr.Code, fpErr.Message)
	}
	return Result{
		OK:        true,
		Status:    "completed",
		SessionID: result.SessionID,
		Path:      result.Attempt.InputPath,
	}
}

func (a *App) StartReceive(sessionID string, outputDir string) Result {
	if sessionID == "" {
		return failure(fperrors.MissingSessionID, "Enter the FilePilot Session ID shown by the sender.")
	}
	result, fpErr := transfer.Receive(a.ctx, transfer.ReceiveOptions{
		SessionID: sessionID,
		OutputDir: outputDir,
		OnEvent:   a.emitTransferEvent,
	})
	if fpErr != nil {
		return failure(fpErr.Code, fpErr.Message)
	}
	return Result{
		OK:        true,
		Status:    "completed",
		SessionID: sessionID,
		Path:      result.Attempt.OutputPath,
	}
}

func (a *App) OpenFolder(path string) error {
	if path == "" {
		return fmt.Errorf("folder path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	runtime.BrowserOpenURL(a.ctx, (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String())
	return nil
}

func (a *App) emitTransferEvent(event transfer.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "transfer:event", event)
	}
}

func (a *App) defaultReceiveFolder() (string, error) {
	resolved, err := paths.Current()
	if err != nil {
		return "", err
	}
	effective, err := config.Load(resolved.ConfigPath, resolved.DownloadDir)
	if err != nil {
		return "", err
	}
	return filepath.Clean(effective.Values.DownloadDir), nil
}

func failure(code fperrors.Code, message string) Result {
	return Result{
		OK:        false,
		Status:    "failed",
		ErrorCode: string(code),
		Error:     message,
	}
}
