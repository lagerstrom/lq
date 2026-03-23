package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunFormatsJSONAndPassesThroughText(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	var out bytes.Buffer
	input := strings.NewReader("{\"ts\":1712345678.25,\"b\":1,\"a\":true}\nnot json\n")

	if err := run(input, &out, time.UTC); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	want := "" +
		"001  {\n" +
		"002    \"a\": true,\n" +
		"003    \"b\": 1,\n" +
		"004    \"ts\": 1712345678.25 (19:34:38.250 05/04/2024 UTC)\n" +
		"005  }\n" +
		"006  not json\n"

	if got := out.String(); got != want {
		t.Fatalf("unexpected output:\n%s", got)
	}
}

func TestRunFormatsBracketedLogs(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	var out bytes.Buffer
	input := strings.NewReader("[2026-03-23 08:14:41,898: INFO/ForkPoolWorker-1] Task seal6.tasks.delete_old_catalog_index_by_cat_id[60558595-a80f-413d-8982-91844d101dde] succeeded in 39.72505997799999s: None\n")

	if err := run(input, &out, time.UTC); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	want := "001  [08:14:41.898 23/03/2026 UTC] [INFO/ForkPoolWorker-1] Task seal6.tasks.delete_old_catalog_index_by_cat_id[60558595-a80f-413d-8982-91844d101dde] succeeded in 39.72505997799999s: None\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected output:\n%s", got)
	}
}

func TestShouldUseColorRespectsWriterAndNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if shouldUseColor(&bytes.Buffer{}) {
		t.Fatal("expected bytes.Buffer to disable color")
	}

	t.Setenv("NO_COLOR", "1")
	if shouldUseColor(os.Stdout) {
		t.Fatal("expected NO_COLOR to disable color")
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "seconds with fraction",
			raw:  "1712345678.25",
			want: "19:34:38.250 05/04/2024 UTC",
		},
		{
			name: "milliseconds integer",
			raw:  "1712345678250",
			want: "19:34:38.250 05/04/2024 UTC",
		},
		{
			name: "negative fractional seconds",
			raw:  "-0.25",
			want: "23:59:59.750 31/12/1969 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTimestamp(tt.raw, time.UTC); got != tt.want {
				t.Fatalf("formatTimestamp(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestFormatTimestampRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"",
		"not-a-number",
		"1e999999",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if got := formatTimestamp(raw, time.UTC); got != "" {
				t.Fatalf("formatTimestamp(%q) = %q, want empty string", raw, got)
			}
		})
	}
}

func TestParseBracketedTimestamp(t *testing.T) {
	got, ok := parseBracketedTimestamp("2026-03-23 08:14:41,898", time.UTC)
	if !ok {
		t.Fatal("expected bracketed timestamp to parse")
	}

	if want := "08:14:41.898 23/03/2026 UTC"; got.Format("15:04:05.000 02/01/2006 MST") != want {
		t.Fatalf("unexpected parsed timestamp: %s", got.Format("15:04:05.000 02/01/2006 MST"))
	}
}

func TestUsageTextExamplesMatchCommandPath(t *testing.T) {
	usage := usageText()
	if !strings.Contains(usage, "go run ./cmd/lq") {
		t.Fatalf("usage text does not mention correct command path:\n%s", usage)
	}
	if !strings.Contains(usage, "--version") {
		t.Fatalf("usage text does not mention version flag:\n%s", usage)
	}
}

func TestVersionDefault(t *testing.T) {
	if version == "" {
		t.Fatal("version should not be empty")
	}
}
