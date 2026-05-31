package fperrors

import "testing"

func TestExitCodesMatchPRD(t *testing.T) {
	cases := map[Code]int{
		InvalidArgument:       2,
		PathNotFound:          3,
		PermissionDenied:      9,
		BackendNotFound:       4,
		BackendUnavailable:    4,
		PackFailed:            5,
		TransferFailed:        6,
		MissingSessionID:      7,
		InvalidSessionID:      7,
		LocalTimeout:          8,
		Cancelled:             8,
		ConfigError:           10,
		DownloadDirUnwritable: 10,
		CacheDirUnwritable:    10,
	}

	for code, want := range cases {
		got := ExitCodeFor(code)
		if got != want {
			t.Fatalf("ExitCodeFor(%s) = %d, want %d", code, got, want)
		}
	}
}

func TestNewErrorCarriesStableFields(t *testing.T) {
	err := New(InvalidArgument, "The command is not valid.", "Run filepilot --help.")

	if err.Code != InvalidArgument {
		t.Fatalf("Code = %s, want %s", err.Code, InvalidArgument)
	}
	if err.Message != "The command is not valid." {
		t.Fatalf("Message = %q", err.Message)
	}
	if err.Hint != "Run filepilot --help." {
		t.Fatalf("Hint = %q", err.Hint)
	}
	if err.ExitCode() != 2 {
		t.Fatalf("ExitCode = %d, want 2", err.ExitCode())
	}
}
