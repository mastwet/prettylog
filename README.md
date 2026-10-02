# prettylog

[![Go Reference](https://pkg.go.dev/badge/github.com/mastwet/prettylog.svg)](https://pkg.go.dev/github.com/mastwet/prettylog)

Zero-dependency, teal-first, color-aware [`log/slog`](https://pkg.go.dev/log/slog) handler for Go.

Degrades cleanly: **truecolor → 256-color → 16-color → plain**.

```
20:13:37.506  INFO  service ready  │  addr="127.0.0.1:8787"  boot_time=842ms
```

## Install

```bash
go get github.com/mastwet/prettylog
```

## Quick start

```go
package main

import (
    "os"

    "github.com/mastwet/prettylog"
)

func main() {
    log := prettylog.New(os.Stdout)
    log.Info("service ready", "addr", "127.0.0.1:8787")
}
```

### Options

```go
log := prettylog.New(os.Stdout,
    prettylog.WithLevelString("debug"),
    prettylog.WithSource(true),
    prettylog.WithColor(prettylog.ColorAuto),
)
```

### Global slog default

```go
prettylog.SetDefault(prettylog.WithLevelString("info"))
```

### As a slog.Handler

```go
h := prettylog.NewHandler(os.Stderr, prettylog.WithColor(prettylog.Color16))
logger := slog.New(h)
```

### Attribute scopes and redaction

Attributes keep the group scope in which they were attached:

```go
logger.With("run_id", "r1").
    WithGroup("model").
    With("provider", "deepseek").
    WithGroup("usage").
    Info("call", "tokens", 7)
// ... call │ run_id="r1" model.provider="deepseek" model.usage.tokens=7
```

`WithReplaceAttr` receives resolved leaf attributes and their full group path.
It runs on user attributes, including time values, and the optional `source`
attribute. It does not run on the timestamp, level, message or `WithSource`
columns. Return `slog.Attr{}` to remove an attribute.

To rewrite or remove source information, enable `Options.AddSource`:

```go
log := prettylog.New(os.Stderr,
    func(o *prettylog.Options) { o.AddSource = true },
    prettylog.WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
        if len(groups) == 0 && a.Key == slog.SourceKey {
            return slog.Attr{}
        }
        return a
    }),
)
```

The source hook receives a `*slog.Source`; without a program counter, the
source attribute is omitted. `WithSource(true)` independently enables the
file:line column.

Messages, errors, keys and other values escape line breaks and control
characters so each record stays on one physical line. String values also
escape quotes and backslashes. Empty or removed attributes leave no separator.

## Color policy

| Mode | Behavior |
|---|---|
| `ColorAuto` | TTY + env detect; plain when piped |
| `ColorAlways` | Force color even when piped |
| `ColorNever` | Plain text |
| `ColorTrueColor` | Force 24-bit |
| `Color256` | Force xterm-256 |
| `Color16` | Classic 16-color with solid badge chips |

On Windows the handler enables `ENABLE_VIRTUAL_TERMINAL_PROCESSING` so
legacy conhost can render the 16-color fallback instead of escape garbage.

```go
mode, _ := prettylog.ParseColorMode("truecolor")
log := prettylog.New(os.Stdout, prettylog.WithColor(mode))
```

## Config file

Default colors and options live in JSON. Search order:

1. `$PRETTYLOG_CONFIG`
2. `./prettylog.json`
3. `./prettylog.config.json`
4. `./config/prettylog.json`
5. `$XDG_CONFIG_HOME/prettylog/config.json`
6. `~/.config/prettylog/config.json`
7. `~/.prettylog.json`

A complete teal-first default ships in this repo as [`prettylog.json`](prettylog.json).
Omitted fields keep built-in defaults; only the keys you set are overridden.

```json
{
  "level": "info",
  "color": "auto",
  "show_source": false,
  "time_format": "15:04:05.000",
  "palette": {
    "info":   { "fg": "#042F2E", "bg": "#0D9488", "fg16": "black",    "bg16": "br-cyan" },
    "key":    { "fg": "#5EEAD4", "fg16": "br-cyan" },
    "number": { "fg": "#FBBF24", "fg16": "yellow" }
  }
}
```

```go
// Auto-discover
log := prettylog.New(os.Stdout, prettylog.WithConfigSearch())

// Explicit file
log := prettylog.New(os.Stdout, prettylog.WithConfigFile("my-theme.json"))

// In-memory
cfg := prettylog.DefaultConfig()
cfg.Palette.Info.Bg = "#FF0000"
log := prettylog.New(os.Stdout, prettylog.WithConfig(cfg))
```

Color fields: `fg`/`bg` are `#RRGGBB`; `fg16`/`bg16` are names
(`black`…`white`, `br-cyan`…); `fg256`/`bg256` are xterm indexes.

## Features

- Single-line records, groups flattened as `parent.child=v`
- Level chips as solid color blocks at every depth
- Typed value colors (string / number / bool / error / duration)
- `ReplaceAttr` hook (drop or rewrite keys, e.g. secrets)
- Thread-safe writes
- Windows VT auto-enable for legacy conhost
- Zero third-party dependencies

## Demo

```bash
cd examples/demo
go run .                       # auto depth
go run . -color truecolor -source
go run . -color 16             # old-terminal chips
go run . -color off
go run . -level debug
```

## Tests

```bash
go test ./...
go test -race ./...
go vet ./...
```

GitHub Actions runs the suite and vet on Linux and Windows, plus the race
detector on Linux.

## License

[MIT](LICENSE)
