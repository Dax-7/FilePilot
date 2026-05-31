package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"filepilot/internal/session"
)

const FileName = "transfer-history.jsonl"

type Attempt struct {
	Timestamp time.Time

	Mode        string
	Status      string
	PayloadType string

	InputPath   string
	OutputPath  string
	PackagePath string
	ArchivePath string
	FileSize    int64
	Packed      bool
	Unpacked    bool

	Backend       string
	BackendSource string
	SessionID     string
	Duration      time.Duration

	ErrorCode    string
	ErrorMessage string

	BackendRawCommand string
	BackendCredential string
}

type Writer struct {
	path string
}

func NewWriter(path string) Writer {
	return Writer{path: path}
}

func DefaultPath(logDir string) string {
	return filepath.Join(logDir, FileName)
}

func (w Writer) Append(attempt Attempt) error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return err
	}

	if attempt.Timestamp.IsZero() {
		attempt.Timestamp = time.Now().UTC()
	}

	row := record{
		Timestamp:         attempt.Timestamp.UTC().Format(time.RFC3339),
		Mode:              attempt.Mode,
		Status:            attempt.Status,
		PayloadType:       attempt.PayloadType,
		InputPath:         attempt.InputPath,
		OutputPath:        attempt.OutputPath,
		PackagePath:       firstNonEmpty(attempt.PackagePath, attempt.ArchivePath),
		FileSize:          attempt.FileSize,
		Packed:            attempt.Packed,
		Unpacked:          attempt.Unpacked,
		Backend:           attempt.Backend,
		BackendSource:     attempt.BackendSource,
		SessionIDRedacted: RedactSessionID(attempt.SessionID),
		DurationMS:        attempt.Duration.Milliseconds(),
		ErrorCode:         attempt.ErrorCode,
		ErrorMessage:      attempt.ErrorMessage,
	}

	content, err := json.Marshal(row)
	if err != nil {
		return err
	}
	content = append(content, '\n')

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(content)
	return err
}

func RedactSessionID(sessionID string) string {
	return session.Redact(sessionID)
}

type record struct {
	Timestamp string `json:"timestamp"`

	Mode        string `json:"mode"`
	Status      string `json:"status"`
	PayloadType string `json:"payload_type"`

	InputPath   string `json:"input_path,omitempty"`
	OutputPath  string `json:"output_path,omitempty"`
	PackagePath string `json:"package_path,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
	Packed      bool   `json:"packed"`
	Unpacked    bool   `json:"unpacked"`

	Backend       string `json:"backend,omitempty"`
	BackendSource string `json:"backend_source,omitempty"`

	SessionIDRedacted string `json:"session_id_redacted,omitempty"`
	DurationMS        int64  `json:"duration_ms"`

	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
