package prettylog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the on-disk (or in-memory) default for prettylog.
// Zero values are filled from DefaultConfig / built-in palette.
type Config struct {
	// Level is "debug"|"info"|"warn"|"error".
	Level string `json:"level,omitempty"`
	// Color is "auto"|"always"|"never"|"truecolor"|"256"|"16".
	Color string `json:"color,omitempty"`
	// ShowSource toggles the file:line column.
	ShowSource bool `json:"show_source,omitempty"`
	// TimeFormat is the timestamp layout.
	TimeFormat string `json:"time_format,omitempty"`
	// Palette overrides semantic colors. Omitted fields keep defaults.
	Palette PaletteConfig `json:"palette,omitempty"`
}

// PaletteConfig holds per-role color overrides.
// Hex fields use #RRGGBB; *16 fields use names like "black", "br-cyan".
type PaletteConfig struct {
	Info       ColorSpec `json:"info,omitempty"`
	Warn       ColorSpec `json:"warn,omitempty"`
	Error      ColorSpec `json:"error,omitempty"`
	Debug      ColorSpec `json:"debug,omitempty"`
	Fatal      ColorSpec `json:"fatal,omitempty"`
	Key        ColorSpec `json:"key,omitempty"`
	String     ColorSpec `json:"string,omitempty"`
	Number     ColorSpec `json:"number,omitempty"`
	Bool       ColorSpec `json:"bool,omitempty"`
	ErrorText  ColorSpec `json:"error_text,omitempty"`
	Message    ColorSpec `json:"message,omitempty"`
	Dim        ColorSpec `json:"dim,omitempty"`
	Muted      ColorSpec `json:"muted,omitempty"`
	Teal       ColorSpec `json:"teal,omitempty"`
	TealBright ColorSpec `json:"teal_bright,omitempty"`
}

// ColorSpec is one semantic color. Empty fields keep the built-in default.
type ColorSpec struct {
	// Fg / Bg are #RRGGBB truecolor values.
	Fg string `json:"fg,omitempty"`
	Bg string `json:"bg,omitempty"`
	// Fg16 / Bg16 are classic 16-color names.
	Fg16 string `json:"fg16,omitempty"`
	Bg16 string `json:"bg16,omitempty"`
	// Fg256 / Bg256 are xterm-256 indexes (0–255).
	Fg256 *int `json:"fg256,omitempty"`
	Bg256 *int `json:"bg256,omitempty"`
	// Bold forces the bold attribute on this role.
	Bold *bool `json:"bold,omitempty"`
}

// DefaultConfig returns the built-in teal-first defaults.
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Color:      "auto",
		ShowSource: false,
		TimeFormat: "15:04:05.000",
		Palette:    defaultPaletteConfig(),
	}
}

func defaultPaletteConfig() PaletteConfig {
	return PaletteConfig{
		Info:       ColorSpec{Fg: "#042F2E", Bg: "#0D9488", Fg16: "black", Bg16: "br-cyan"},
		Warn:       ColorSpec{Fg: "#FEF3C7", Bg: "#B45309", Fg16: "black", Bg16: "br-yellow"},
		Error:      ColorSpec{Fg: "#FFF1F2", Bg: "#BE185D", Fg16: "br-white", Bg16: "br-red"},
		Debug:      ColorSpec{Fg: "#E2E8F0", Bg: "#334155", Fg16: "br-white", Bg16: "br-black"},
		Fatal:      ColorSpec{Fg: "#F3E8FF", Bg: "#7E22CE", Fg16: "br-white", Bg16: "br-magenta"},
		Key:        ColorSpec{Fg: "#5EEAD4", Fg16: "br-cyan"},
		String:     ColorSpec{Fg: "#A7F3D0", Fg16: "br-green"},
		Number:     ColorSpec{Fg: "#FBBF24", Fg16: "yellow"},
		Bool:       ColorSpec{Fg: "#C4B5FD", Fg16: "magenta"},
		ErrorText:  ColorSpec{Fg: "#FB7185", Fg16: "br-red"},
		Message:    ColorSpec{Fg: "#E2F5F0", Fg16: "br-white"},
		Dim:        ColorSpec{Fg: "#6E8A8A", Fg16: "br-black"},
		Muted:      ColorSpec{Fg: "#506868", Fg16: "br-black"},
		Teal:       ColorSpec{Fg: "#2DD4BF", Fg16: "cyan"},
		TealBright: ColorSpec{Fg: "#5EEAD4", Fg16: "br-cyan"},
	}
}

// DefaultConfigPaths lists locations searched by LoadConfig (first hit wins).
func DefaultConfigPaths() []string {
	paths := []string{
		"prettylog.json",
		"prettylog.config.json",
		filepath.Join("config", "prettylog.json"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "prettylog", "config.json"),
			filepath.Join(home, ".prettylog.json"),
		)
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		paths = append([]string{filepath.Join(x, "prettylog", "config.json")}, paths...)
	}
	if env := os.Getenv("PRETTYLOG_CONFIG"); env != "" {
		paths = append([]string{env}, paths...)
	}
	return paths
}

// ErrConfigNotFound is returned when no config file exists on the search path.
var ErrConfigNotFound = errors.New("prettylog: config file not found")

// LoadConfig reads a JSON config from path. Missing file → ErrConfigNotFound.
// Unknown fields are ignored; zero fields keep built-in defaults after Merge.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return Config{}, fmt.Errorf("prettylog: read config %s: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig decodes JSON bytes into a Config (no defaults applied).
func ParseConfig(data []byte) (Config, error) {
	var c Config
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields() // strict enough to catch typos, still flexible for known fields
	// Actually DisallowUnknownFields is harsh for forward-compat; use lenient decode.
	dec = json.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("prettylog: parse config: %w", err)
	}
	return c, nil
}

// FindAndLoadConfig walks DefaultConfigPaths and returns the first file that loads.
// If none exist, returns DefaultConfig() and the path "".
func FindAndLoadConfig() (Config, string, error) {
	for _, p := range DefaultConfigPaths() {
		c, err := LoadConfig(p)
		if err == nil {
			return c, p, nil
		}
		if !errors.Is(err, ErrConfigNotFound) {
			return Config{}, p, err
		}
	}
	return DefaultConfig(), "", nil
}

// Merge overlays non-zero fields from src onto dst.
func (dst Config) Merge(src Config) Config {
	if src.Level != "" {
		dst.Level = src.Level
	}
	if src.Color != "" {
		dst.Color = src.Color
	}
	if src.TimeFormat != "" {
		dst.TimeFormat = src.TimeFormat
	}
	// ShowSource: only override when src came from a file that set it true,
	// or when we want explicit false. JSON false is zero-value so we cannot
	// distinguish "unset" from "false". Callers that need force-false should
	// set it in DefaultConfig and merge carefully. For file load, we treat
	// true as explicit; false keeps previous unless entire palette merge.
	if src.ShowSource {
		dst.ShowSource = true
	}
	dst.Palette = mergePalette(dst.Palette, src.Palette)
	return dst
}

func mergePalette(base, over PaletteConfig) PaletteConfig {
	base.Info = mergeSpec(base.Info, over.Info)
	base.Warn = mergeSpec(base.Warn, over.Warn)
	base.Error = mergeSpec(base.Error, over.Error)
	base.Debug = mergeSpec(base.Debug, over.Debug)
	base.Fatal = mergeSpec(base.Fatal, over.Fatal)
	base.Key = mergeSpec(base.Key, over.Key)
	base.String = mergeSpec(base.String, over.String)
	base.Number = mergeSpec(base.Number, over.Number)
	base.Bool = mergeSpec(base.Bool, over.Bool)
	base.ErrorText = mergeSpec(base.ErrorText, over.ErrorText)
	base.Message = mergeSpec(base.Message, over.Message)
	base.Dim = mergeSpec(base.Dim, over.Dim)
	base.Muted = mergeSpec(base.Muted, over.Muted)
	base.Teal = mergeSpec(base.Teal, over.Teal)
	base.TealBright = mergeSpec(base.TealBright, over.TealBright)
	return base
}

func mergeSpec(base, over ColorSpec) ColorSpec {
	if over.Fg != "" {
		base.Fg = over.Fg
	}
	if over.Bg != "" {
		base.Bg = over.Bg
	}
	if over.Fg16 != "" {
		base.Fg16 = over.Fg16
	}
	if over.Bg16 != "" {
		base.Bg16 = over.Bg16
	}
	if over.Fg256 != nil {
		base.Fg256 = over.Fg256
	}
	if over.Bg256 != nil {
		base.Bg256 = over.Bg256
	}
	if over.Bold != nil {
		base.Bold = over.Bold
	}
	return base
}

// Options applies Level/Color/ShowSource/TimeFormat onto functional options.
func (c Config) Options() []Option {
	var opts []Option
	if c.Level != "" {
		opts = append(opts, WithLevelString(c.Level))
	}
	if c.TimeFormat != "" {
		opts = append(opts, WithTimeFormat(c.TimeFormat))
	}
	opts = append(opts, WithSource(c.ShowSource))
	if c.Color != "" {
		if mode, ok := ParseColorMode(c.Color); ok {
			opts = append(opts, WithColor(mode))
		}
	}
	return opts
}

// LoadDefaultConfigFile is the convenience entrypoint for main():
// find file (if any), merge onto defaults, return config + path used.
func LoadDefaultConfigFile() (Config, string) {
	c, path, err := FindAndLoadConfig()
	if err != nil {
		// Malformed file: fall back to defaults rather than crash the host app.
		return DefaultConfig(), ""
	}
	merged := DefaultConfig().Merge(c)
	return merged, path
}
