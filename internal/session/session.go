package session

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)

const (
	Prefix             = "FP"
	randomSuffixLength = 13
	minRandomLength    = 10
)

var (
	wordPattern   = regexp.MustCompile(`^[a-z][a-z0-9]{2,15}$`)
	randomPattern = regexp.MustCompile(`^[A-Z2-9]+$`)
	words         = []string{
		"river", "copper", "lamp", "forest", "maple", "stone", "harbor", "silver",
		"orbit", "signal", "meadow", "summit", "ember", "anchor", "violet", "cedar",
		"quartz", "lantern", "bridge", "canvas", "rocket", "pencil", "velvet", "garden",
		"cloud", "delta", "frost", "matrix", "pixel", "solar", "tunnel", "winter",
	}
	randomAlphabet = []byte("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")
)

func Generate() (string, error) {
	first, err := randomWord()
	if err != nil {
		return "", err
	}
	second, err := randomWord()
	if err != nil {
		return "", err
	}
	third, err := randomWord()
	if err != nil {
		return "", err
	}
	suffix, err := randomSuffix()
	if err != nil {
		return "", err
	}
	return strings.Join([]string{Prefix, first, second, third, suffix}, "-"), nil
}

func Validate(id string) error {
	if id == "" {
		return fmt.Errorf("session ID is required")
	}
	if strings.ContainsAny(id, " \t\r\n") {
		return fmt.Errorf("session ID contains whitespace")
	}
	parts := strings.Split(id, "-")
	if len(parts) != 5 || parts[0] != Prefix {
		return fmt.Errorf("session ID must use FP-word-word-word-random shape")
	}
	for _, word := range parts[1:4] {
		if !wordPattern.MatchString(word) {
			return fmt.Errorf("session ID contains an invalid word segment")
		}
	}
	suffix := parts[4]
	if len(suffix) < minRandomLength || !randomPattern.MatchString(suffix) {
		return fmt.Errorf("session ID random suffix is invalid or too short")
	}
	if !hasDiversity(suffix) {
		return fmt.Errorf("session ID random suffix has insufficient diversity")
	}
	return nil
}

func Redact(id string) string {
	if id == "" {
		return ""
	}
	parts := strings.Split(id, "-")
	if len(parts) >= 4 && parts[0] == Prefix {
		return parts[0] + "-" + parts[1] + "-****-" + parts[len(parts)-1]
	}
	if strings.HasPrefix(id, Prefix+"-") {
		return Prefix + "-****"
	}
	return "****"
}

func randomWord() (string, error) {
	index, err := secureIndex(len(words))
	if err != nil {
		return "", err
	}
	return words[index], nil
}

func randomSuffix() (string, error) {
	encoded := make([]byte, randomSuffixLength)
	for i := range encoded {
		index, err := secureIndex(len(randomAlphabet))
		if err != nil {
			return "", err
		}
		encoded[i] = randomAlphabet[index]
	}
	return string(encoded), nil
}

func secureIndex(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("empty word list")
	}
	var value [1]byte
	limit := 256 - (256 % max)
	for {
		if _, err := rand.Read(value[:]); err != nil {
			return 0, err
		}
		if int(value[0]) < limit {
			return int(value[0]) % max, nil
		}
	}
}

func hasDiversity(value string) bool {
	if value == "" {
		return false
	}
	first := value[0]
	for i := 1; i < len(value); i++ {
		if value[i] != first {
			return true
		}
	}
	return false
}
