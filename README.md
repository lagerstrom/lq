# lq

`lq` is a small command-line tool for prettifying line-oriented logs.
It pretty-prints valid JSON lines, formats bracketed worker logs, leaves other lines unchanged, and expands numeric `ts` fields into a human-readable timestamp.

## Requirements

- Go 1.23 or newer

## Install Locally With Go

From the repository root, run:

```bash
go install ./cmd/lq
```

This builds the binary and installs it into Go's bin directory, usually:

```bash
$(go env GOPATH)/bin
```

If `GOBIN` is set, it will be installed there instead.

Make sure that directory is on your `PATH`. For example:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

If you use `GOBIN`, add that directory to your `PATH` instead.

## Install Directly From GitHub

You can also install `lq` without cloning the repository first:

```bash
go install github.com/lagerstrom/lq/cmd/lq@latest
```

This installs the latest tagged or published version from GitHub into the same Go bin directory described above.

## Verify The Install

```bash
lq --help
```

## Usage

```bash
cat app.log | lq
cat app.log | lq --timezone UTC
cat app.log | lq --theme dracula
journalctl -o cat | lq --timezone Europe/Stockholm
```

It also recognizes log lines in formats like:

```text
[2026-03-23 08:14:41,898: INFO/ForkPoolWorker-1] Task ... succeeded in 39.72s: None
```

## Development Run

If you want to run it without installing:

```bash
go run ./cmd/lq --help
```

## Behavior

- Pretty-prints valid JSON lines with ANSI colors when writing to a terminal
- Formats bracketed worker logs like `[timestamp: LEVEL/source] message`
- Passes through non-JSON input unchanged
- Expands numeric `ts` fields into the selected timezone
- Supports built-in color themes including `default` and `dracula`
- Uses the local timezone by default
