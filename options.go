package prettylog

import (
	"log/slog"
	"os"
	"time"
)

// FileStat is the small surface resolve() needs to test for a TTY.
// *os.File satisfies it.
type FileStat interface {
	Stat() (os.FileInfo, error)
}

// Options configures a Handler or Logger.
type Options struct {
	// Level is the minimum level to emit. Defaults to Info.
	Level slog.Leveler
	// TimeFormat is used for the timestamp column. Defaults to "15:04:05.000".
	TimeFormat string
	// ShowSource prints file:line after the level badge.
	ShowSource bool
	// Color is the color policy. Defaults to ColorAuto.
	Color ColorMode
	// AddSource adds a record-level source attribute when a source is available.
	// ReplaceAttr receives its value as *slog.Source. Prefer ShowSource for the
	// separate file:line column.
	AddSource bool
	// ReplaceAttr, when set, rewrites resolved non-group attributes, including
	// time-valued attributes and AddSource. The groups argument contains the
	// attribute's full group path and must not be retained or modified.
	// Return a zero Attr to drop it. The time, level, message and ShowSource
	// columns are not passed to this hook.
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
	// Config, when non-nil, supplies palette overrides and default
	// level/color/time when those options are unset.
	Config *Config
}

// Option mutates Options.
type Option func(*Options)

// WithLevel sets the minimum level.
func WithLevel(level slog.Leveler) Option {
	return func(o *Options) { o.Level = level }
}

// WithLevelString parses "debug"|"info"|"warn"|"error".
func WithLevelString(s string) Option {
	return func(o *Options) { o.Level = ParseLevel(s) }
}

// WithTimeFormat sets the timestamp layout.
func WithTimeFormat(layout string) Option {
	return func(o *Options) { o.TimeFormat = layout }
}

// WithSource enables the file:line column.
func WithSource(on bool) Option {
	return func(o *Options) { o.ShowSource = on }
}

// WithColor sets the color policy.
func WithColor(mode ColorMode) Option {
	return func(o *Options) { o.Color = mode }
}

// WithReplaceAttr installs a slog-style attribute hook.
func WithReplaceAttr(fn func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(o *Options) { o.ReplaceAttr = fn }
}

// WithConfig merges cfg into the options (level/color/time/source + palette).
func WithConfig(cfg Config) Option {
	return func(o *Options) {
		c := cfg
		o.Config = &c
		if c.Level != "" {
			o.Level = ParseLevel(c.Level)
		}
		if c.TimeFormat != "" {
			o.TimeFormat = c.TimeFormat
		}
		if c.ShowSource {
			o.ShowSource = true
		}
		if c.Color != "" {
			if mode, ok := ParseColorMode(c.Color); ok {
				o.Color = mode
			}
		}
	}
}

// WithConfigFile loads a JSON config from path and applies it.
// Missing file is a no-op; malformed file is also a no-op (defaults stay).
func WithConfigFile(path string) Option {
	return func(o *Options) {
		cfg, err := LoadConfig(path)
		if err != nil {
			return
		}
		WithConfig(cfg)(o)
	}
}

// WithConfigSearch walks DefaultConfigPaths and applies the first hit,
// merged onto DefaultConfig().
func WithConfigSearch() Option {
	return func(o *Options) {
		cfg, _, err := FindAndLoadConfig()
		if err != nil {
			return
		}
		merged := DefaultConfig().Merge(cfg)
		WithConfig(merged)(o)
	}
}

func applyOptions(opts []Option) Options {
	o := Options{
		Level:      slog.LevelInfo,
		TimeFormat: "15:04:05.000",
		Color:      ColorAuto,
	}
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	if o.Level == nil {
		o.Level = slog.LevelInfo
	}
	if o.TimeFormat == "" {
		o.TimeFormat = "15:04:05.000"
	}
	return o
}

// ParseLevel maps a string to slog.Level. Unknown values become Info.
func ParseLevel(s string) slog.Level {
	switch lower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// resolve turns a ColorMode + writer into (enabled, depth).
func resolve(mode ColorMode, w FileStat) (bool, ColorDepth) {
	if os.Getenv("NO_COLOR") != "" && mode != ColorTrueColor &&
		mode != Color256 && mode != Color16 {
		// Explicit forced depths still win over NO_COLOR (user asked).
		// Auto/Always/Never respect NO_COLOR.
		if mode == ColorNever || mode == ColorAuto || mode == ColorAlways {
			return false, DepthOff
		}
	}

	switch mode {
	case ColorNever:
		return false, DepthOff
	case ColorTrueColor:
		return true, DepthTrueColor
	case Color256:
		return true, DepthANSI256
	case Color16:
		return true, DepthANSI16
	case ColorAlways:
		EnableVirtualTerminal()
		d := DetectDepth()
		if d == DepthOff {
			return true, DepthANSI16
		}
		return true, d
	default: // ColorAuto
		if !isTerminal(w) {
			return false, DepthOff
		}
		EnableVirtualTerminal()
		d := DetectDepth()
		return d != DepthOff, d
	}
}

func isTerminal(w FileStat) bool {
	if w == nil {
		return false
	}
	info, err := w.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// defaultNow is swappable in tests.
var defaultNow = time.Now
