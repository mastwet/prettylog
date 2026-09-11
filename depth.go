package prettylog

// ColorDepth selects how ANSI colors are encoded.
type ColorDepth int

const (
	// DepthOff emits no ANSI codes.
	DepthOff ColorDepth = iota
	// DepthANSI16 uses classic 4-bit SGR (30–37 / 90–97 / 100–107).
	DepthANSI16
	// DepthANSI256 uses 38;5;N / 48;5;N.
	DepthANSI256
	// DepthTrueColor uses 38;2;R;G;B.
	DepthTrueColor
)

func (d ColorDepth) String() string {
	switch d {
	case DepthOff:
		return "off"
	case DepthANSI16:
		return "16"
	case DepthANSI256:
		return "256"
	case DepthTrueColor:
		return "truecolor"
	default:
		return "unknown"
	}
}

// ColorMode is the user-facing color policy passed to WithColor.
type ColorMode int

const (
	// ColorAuto detects depth from the environment and TTY state.
	ColorAuto ColorMode = iota
	// ColorAlways forces colors and auto-detected depth (even when piped).
	ColorAlways
	// ColorNever disables all ANSI output.
	ColorNever
	// ColorTrueColor forces 24-bit color.
	ColorTrueColor
	// Color256 forces xterm-256 color.
	Color256
	// Color16 forces classic 16-color.
	Color16
)

// ParseColorMode maps a CLI-style string to a ColorMode.
func ParseColorMode(s string) (ColorMode, bool) {
	switch lower(s) {
	case "auto":
		return ColorAuto, true
	case "always", "on", "force":
		return ColorAlways, true
	case "off", "never", "none", "plain":
		return ColorNever, true
	case "truecolor", "true", "24", "24bit":
		return ColorTrueColor, true
	case "256":
		return Color256, true
	case "16", "ansi", "basic":
		return Color16, true
	default:
		return ColorAuto, false
	}
}
