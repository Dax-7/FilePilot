package output

import (
	"encoding/json"
	"io"

	"filepilot/internal/fperrors"
)

type Result struct {
	OK     bool
	Status string
	Mode   string
	Error  *fperrors.Error
	Fields map[string]any
}

func Success(mode string, status string, fields map[string]any) Result {
	return Result{
		OK:     true,
		Status: status,
		Mode:   mode,
		Error:  nil,
		Fields: fields,
	}
}

func Failure(mode string, err *fperrors.Error) Result {
	return Result{
		OK:     false,
		Status: "failed",
		Mode:   mode,
		Error:  err,
	}
}

func WriteJSON(writer io.Writer, result Result) error {
	payload := map[string]any{
		"ok":     result.OK,
		"status": result.Status,
		"mode":   result.Mode,
		"error":  result.Error,
	}
	for key, value := range result.Fields {
		payload[key] = value
	}

	encoder := json.NewEncoder(writer)
	return encoder.Encode(payload)
}
