package fperrors

type Code string

const (
	InvalidArgument       Code = "INVALID_ARGUMENT"
	PathNotFound          Code = "PATH_NOT_FOUND"
	PermissionDenied      Code = "PERMISSION_DENIED"
	BackendNotFound       Code = "BACKEND_NOT_FOUND"
	BackendUnavailable    Code = "BACKEND_UNAVAILABLE"
	PackFailed            Code = "PACK_FAILED"
	TransferFailed        Code = "TRANSFER_FAILED"
	MissingSessionID      Code = "MISSING_SESSION_ID"
	InvalidSessionID      Code = "INVALID_SESSION_ID"
	LocalTimeout          Code = "LOCAL_TIMEOUT"
	Cancelled             Code = "CANCELLED"
	ConfigError           Code = "CONFIG_ERROR"
	DownloadDirUnwritable Code = "DOWNLOAD_DIR_UNWRITABLE"
	CacheDirUnwritable    Code = "CACHE_DIR_UNWRITABLE"
)

type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func New(code Code, message string, hint string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Hint:    hint,
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) ExitCode() int {
	if e == nil {
		return 0
	}
	return ExitCodeFor(e.Code)
}

func ExitCodeFor(code Code) int {
	switch code {
	case InvalidArgument:
		return 2
	case PathNotFound:
		return 3
	case BackendNotFound, BackendUnavailable:
		return 4
	case PackFailed:
		return 5
	case TransferFailed:
		return 6
	case MissingSessionID, InvalidSessionID:
		return 7
	case LocalTimeout, Cancelled:
		return 8
	case PermissionDenied:
		return 9
	case ConfigError, DownloadDirUnwritable, CacheDirUnwritable:
		return 10
	default:
		return 1
	}
}
