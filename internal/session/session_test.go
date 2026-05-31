package session

import (
	"strings"
	"testing"
)

func TestGenerateReturnsValidHighEntropyFilePilotSessionID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		id, err := Generate()
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}
		if err := Validate(id); err != nil {
			t.Fatalf("generated id did not validate: %s err=%v", id, err)
		}
		parts := strings.Split(id, "-")
		if len(parts) != 5 {
			t.Fatalf("generated id should have FP plus 3 words plus random suffix: %s", id)
		}
		if len(parts[4]) < 10 {
			t.Fatalf("random suffix too short for MVP entropy: %s", id)
		}
		if seen[id] {
			t.Fatalf("Generate returned duplicate id: %s", id)
		}
		seen[id] = true
	}
}

func TestValidateAcceptsExpectedShape(t *testing.T) {
	valid := "FP-river-copper-lamp-7K2Q9M4XP8"
	if err := Validate(valid); err != nil {
		t.Fatalf("Validate rejected valid session id: %v", err)
	}
}

func TestValidateRejectsMalformedOrUnsafeSessionIDs(t *testing.T) {
	invalid := []string{
		"",
		"FP-84K2",
		"river-copper-lamp-7K2Q9M4XP8",
		"FP-river-copper-lamp-7K2Q",
		"FP-river-copper-lamp-7K2Q9M4XP!",
		"FP-river-copper-lamp-7K2Q9M4XP8 extra",
		"FP-river-copper--7K2Q9M4XP8",
		"FP-river-copper-lamp-0000000000",
		"FP-river-copper-lamp-AAAAAAAAAA",
	}

	for _, id := range invalid {
		if err := Validate(id); err == nil {
			t.Fatalf("Validate accepted invalid session id: %q", id)
		}
	}
}

func TestRedactHidesSensitiveMiddle(t *testing.T) {
	cases := map[string]string{
		"":                                "",
		"FP-river-copper-lamp-7K2Q9M4XP8": "FP-river-****-7K2Q9M4XP8",
		"FP-84K2":                         "FP-****",
		"not-filepilot":                   "****",
	}

	for input, want := range cases {
		got := Redact(input)
		if got != want {
			t.Fatalf("Redact(%q) = %q, want %q", input, got, want)
		}
	}
}
