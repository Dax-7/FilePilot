package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Values struct {
	BackendPath  string
	DownloadDir  string
	AutoUnpack   bool
	KeepPackages bool
	JSONOutput   bool
}

type Effective struct {
	Values
	DefaultDownloadDir string
}

func Defaults(defaultDownloadDir string) Values {
	return Values{
		BackendPath:  "",
		DownloadDir:  defaultDownloadDir,
		AutoUnpack:   true,
		KeepPackages: false,
		JSONOutput:   false,
	}
}

func Load(path string, defaultDownloadDir string) (Effective, error) {
	values := Defaults(defaultDownloadDir)
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return Effective{Values: values, DefaultDownloadDir: defaultDownloadDir}, nil
	}
	if err != nil {
		return Effective{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return Effective{}, fmt.Errorf("invalid config line %d", lineNo)
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		next, err := apply(values, key, rawValue)
		if err != nil {
			return Effective{}, fmt.Errorf("invalid config line %d: %w", lineNo, err)
		}
		values = next
	}
	if err := scanner.Err(); err != nil {
		return Effective{}, err
	}
	if values.DownloadDir == "" {
		values.DownloadDir = defaultDownloadDir
	}
	return Effective{Values: values, DefaultDownloadDir: defaultDownloadDir}, nil
}

func Save(path string, values Values) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := strings.Join([]string{
		`backend_path = "` + escapeString(values.BackendPath) + `"`,
		`download_dir = "` + escapeString(values.DownloadDir) + `"`,
		"auto_unpack = " + strconv.FormatBool(values.AutoUnpack),
		"keep_packages = " + strconv.FormatBool(values.KeepPackages),
		"json_output = " + strconv.FormatBool(values.JSONOutput),
		"",
	}, "\n")
	return os.WriteFile(path, []byte(content), 0o644)
}

func Set(values Values, key string, value string) (Values, error) {
	switch key {
	case "backend_path":
		values.BackendPath = value
	case "download_dir":
		values.DownloadDir = value
	case "auto_unpack":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Values{}, fmt.Errorf("auto_unpack must be true or false")
		}
		values.AutoUnpack = parsed
	case "keep_packages":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Values{}, fmt.Errorf("keep_packages must be true or false")
		}
		values.KeepPackages = parsed
	case "json_output":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Values{}, fmt.Errorf("json_output must be true or false")
		}
		values.JSONOutput = parsed
	default:
		return Values{}, fmt.Errorf("unsupported config key %q", key)
	}
	return values, nil
}

func apply(values Values, key string, rawValue string) (Values, error) {
	switch key {
	case "backend_path", "download_dir":
		value, err := parseString(rawValue)
		if err != nil {
			return Values{}, err
		}
		return Set(values, key, value)
	case "auto_unpack", "keep_packages", "json_output":
		return Set(values, key, rawValue)
	default:
		return Values{}, fmt.Errorf("unsupported config key %q", key)
	}
}

func parseString(raw string) (string, error) {
	value, err := strconv.Unquote(raw)
	if err != nil {
		return "", fmt.Errorf("expected TOML string")
	}
	return value, nil
}

func escapeString(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`)
}
