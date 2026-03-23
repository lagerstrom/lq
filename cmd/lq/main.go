package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	colorReset   = "\033[0m"
	colorKey     = "\033[38;5;212m"
	colorString  = "\033[38;5;222m"
	colorNumber  = "\033[38;5;117m"
	colorBool    = "\033[38;5;117m"
	colorNull    = "\033[38;5;117m"
	colorBrace   = "\033[38;5;146m"
	colorTS      = "\033[38;5;120m"
	colorLevel   = "\033[38;5;214m"
	colorSource  = "\033[38;5;153m"
	colorLineNum = "\033[38;5;103m"
)

type styler struct {
	color bool
	loc   *time.Location
}

var version = "dev"

var bracketedLogPattern = regexp.MustCompile(`^\[([^\]]+):\s*([A-Z]+)/([^\]]+)\]\s*(.*)$`)

func main() {
	locationName := flag.String("timezone", "local", "Timezone to use for ts fields, e.g. local, UTC, Europe/Stockholm")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usageText())
	}
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version)
		return
	}

	location, err := parseLocation(*locationName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := run(os.Stdin, os.Stdout, location); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(stdin io.Reader, stdout io.Writer, location *time.Location) error {
	s := styler{
		color: shouldUseColor(stdout),
		loc:   location,
	}

	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	lineNumber := 1
	for scanner.Scan() {
		line := scanner.Text()
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			if formatted, ok := tryFormatJSON(trimmed, s); ok {
				for _, outLine := range strings.Split(formatted, "\n") {
					fmt.Fprintln(stdout, s.renderLineNumber(lineNumber)+outLine)
					lineNumber++
				}
				continue
			}

			if formatted, ok := tryFormatBracketedLog(trimmed, s); ok {
				fmt.Fprintln(stdout, s.renderLineNumber(lineNumber)+formatted)
				lineNumber++
				continue
			}
		}

		fmt.Fprintln(stdout, s.renderLineNumber(lineNumber)+line)
		lineNumber++
	}

	return scanner.Err()
}

func tryFormatJSON(line string, s styler) (string, bool) {
	dec := json.NewDecoder(strings.NewReader(line))
	dec.UseNumber()

	var value any
	if err := dec.Decode(&value); err != nil {
		return "", false
	}

	if dec.More() {
		return "", false
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return "", false
	}

	return renderJSON(value, 0, "", s), true
}

func tryFormatBracketedLog(line string, s styler) (string, bool) {
	match := bracketedLogPattern.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}

	ts, ok := parseBracketedTimestamp(match[1], s.loc)
	if !ok {
		return "", false
	}

	level := strings.TrimSpace(match[2])
	source := strings.TrimSpace(match[3])
	message := strings.TrimSpace(match[4])

	return s.paint(colorBrace, "[") +
		s.paint(colorTS, ts.Format("15:04:05.000 02/01/2006 MST")) +
		s.paint(colorBrace, "] [") +
		s.paint(colorLevel, level) +
		s.paint(colorBrace, "/") +
		s.paint(colorSource, source) +
		s.paint(colorBrace, "] ") +
		s.paint(colorString, message), true
}

func renderJSON(value any, indent int, keyName string, s styler) string {
	padding := strings.Repeat("  ", indent)
	nextPadding := strings.Repeat("  ", indent+1)

	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return s.paint(colorBrace, "{}")
		}

		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		lines := make([]string, 0, len(keys))
		for _, key := range keys {
			rendered := nextPadding + s.paint(colorKey, strconv.Quote(key)) +
				s.paint(colorBrace, ": ") +
				renderJSON(typed[key], indent+1, key, s)
			lines = append(lines, rendered)
		}

		return s.paint(colorBrace, "{") + "\n" +
			strings.Join(lines, s.paint(colorBrace, ",")+"\n") + "\n" +
			padding + s.paint(colorBrace, "}")

	case []any:
		if len(typed) == 0 {
			return s.paint(colorBrace, "[]")
		}

		lines := make([]string, 0, len(typed))
		for _, item := range typed {
			lines = append(lines, nextPadding+renderJSON(item, indent+1, "", s))
		}

		return s.paint(colorBrace, "[") + "\n" +
			strings.Join(lines, s.paint(colorBrace, ",")+"\n") + "\n" +
			padding + s.paint(colorBrace, "]")

	case string:
		return s.paint(colorString, strconv.Quote(typed))

	case json.Number:
		return renderNumber(typed, keyName, s)

	case float64:
		return renderFloat(typed, keyName, s)

	case bool:
		return s.paint(colorBool, strconv.FormatBool(typed))

	case nil:
		return s.paint(colorNull, "null")

	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(raw)
	}
}

func renderNumber(value json.Number, keyName string, s styler) string {
	raw := value.String()
	if keyName == "ts" {
		if human := formatTimestamp(raw, s.loc); human != "" {
			return s.paint(colorNumber, raw) + " " + s.paint(colorTS, "("+human+")")
		}
	}
	return s.paint(colorNumber, raw)
}

func renderFloat(value float64, keyName string, s styler) string {
	raw := strconv.FormatFloat(value, 'f', -1, 64)
	if keyName == "ts" {
		if human := formatTimestamp(raw, s.loc); human != "" {
			return s.paint(colorNumber, raw) + " " + s.paint(colorTS, "("+human+")")
		}
	}
	return s.paint(colorNumber, raw)
}

func formatTimestamp(raw string, location *time.Location) string {
	ts, ok := parseTimestamp(raw, location)
	if !ok {
		return ""
	}

	return ts.Format("15:04:05.000 02/01/2006 MST")
}

func parseTimestamp(raw string, location *time.Location) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}

	if !strings.ContainsAny(raw, ".eE") {
		millis, ok := parseInteger(raw)
		if ok && absCmp(millis, big.NewInt(1_000_000_000_000)) > 0 {
			nanosPerMillisecond := big.NewInt(int64(time.Millisecond))
			totalNanos := new(big.Int).Mul(millis, nanosPerMillisecond)
			return unixFromNanoseconds(totalNanos, location)
		}
	}

	value, _, err := big.ParseFloat(raw, 10, 256, big.ToZero)
	if err != nil {
		return time.Time{}, false
	}

	nanosPerSecond := big.NewFloat(float64(time.Second))
	totalNanosFloat := new(big.Float).Mul(value, nanosPerSecond)
	totalNanos, acc := totalNanosFloat.Int(nil)
	if acc != big.Exact && acc != big.Below && acc != big.Above {
		return time.Time{}, false
	}

	return unixFromNanoseconds(totalNanos, location)
}

func parseBracketedTimestamp(raw string, location *time.Location) (time.Time, bool) {
	ts, err := time.ParseInLocation("2006-01-02 15:04:05,000", strings.TrimSpace(raw), location)
	if err != nil {
		return time.Time{}, false
	}
	return ts.In(location), true
}

func parseInteger(raw string) (*big.Int, bool) {
	value := new(big.Int)
	if _, ok := value.SetString(raw, 10); !ok {
		return nil, false
	}
	return value, true
}

func absCmp(value, limit *big.Int) int {
	return new(big.Int).Abs(value).Cmp(limit)
}

func unixFromNanoseconds(totalNanos *big.Int, location *time.Location) (time.Time, bool) {
	if totalNanos == nil || !totalNanos.IsInt64() {
		return time.Time{}, false
	}
	return time.Unix(0, totalNanos.Int64()).In(location), true
}

func parseLocation(name string) (*time.Location, error) {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "", "local":
		return time.Local, nil
	case "utc":
		return time.UTC, nil
	default:
		location, err := time.LoadLocation(name)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q", name)
		}
		return location, nil
	}
}

func (s styler) renderLineNumber(n int) string {
	label := fmt.Sprintf("%03d  ", n)
	return s.paint(colorLineNum, label)
}

func (s styler) paint(color, text string) string {
	if !s.color || text == "" {
		return text
	}
	return color + text + colorReset
}

func shouldUseColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	statter, ok := w.(interface {
		Stat() (os.FileInfo, error)
	})
	if !ok {
		return false
	}

	info, err := statter.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func usageText() string {
	return `Usage:
  cat app.log | go run ./cmd/lq
  cat app.log | go run ./cmd/lq --timezone UTC
  journalctl -o cat | go run ./cmd/lq --timezone Europe/Stockholm

Behavior:
  - Pretty-prints valid JSON lines with ANSI colors.
  - Normalizes bracketed worker logs like [ts: LEVEL/source] message.
  - Passes through non-JSON input unchanged.
  - Expands numeric ts fields into the selected timezone.
  - Default timezone is local.

Flags:
  --timezone string
        Timezone to use for ts fields, e.g. local, UTC, Europe/Stockholm
  --version
        Print version and exit
`
}
