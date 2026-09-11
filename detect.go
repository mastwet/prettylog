package prettylog

import (
	"os"
	"runtime"
	"strings"
)

// DetectDepth picks the best supported depth from env and platform signals.
func DetectDepth() ColorDepth {
	if os.Getenv("NO_COLOR") != "" {
		return DepthOff
	}

	term := strings.ToLower(os.Getenv("TERM"))
	colorterm := strings.ToLower(os.Getenv("COLORTERM"))
	prog := os.Getenv("TERM_PROGRAM")

	switch {
	case colorterm == "truecolor" || colorterm == "24bit":
		return DepthTrueColor
	case os.Getenv("WT_SESSION") != "": // Windows Terminal
		return DepthTrueColor
	case prog == "vscode" || prog == "WezTerm" || prog == "iTerm.app":
		return DepthTrueColor
	case prog == "Apple_Terminal":
		return DepthANSI256
	case strings.Contains(term, "256color"):
		return DepthANSI256
	case term == "" || term == "dumb":
		// Windows conhost without WT still benefits from 16-color after VT.
		if runtime.GOOS == "windows" {
			return DepthANSI16
		}
		return DepthANSI16
	case strings.HasPrefix(term, "xterm"),
		strings.HasPrefix(term, "screen"),
		strings.HasPrefix(term, "tmux"),
		strings.HasPrefix(term, "linux"),
		strings.HasPrefix(term, "ansi"),
		strings.HasPrefix(term, "vt"):
		return DepthANSI16
	default:
		return DepthANSI16
	}
}

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
