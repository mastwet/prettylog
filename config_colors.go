package prettylog

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseHexColor parses #RGB, #RRGGBB into r,g,b (0–255).
func ParseHexColor(s string) (r, g, b int, err error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3:
		// #RGB → #RRGGBB
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6:
		// ok
	default:
		return 0, 0, 0, fmt.Errorf("prettylog: bad hex color %q", s)
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("prettylog: bad hex color %q: %w", s, err)
	}
	return int(n >> 16 & 0xff), int(n >> 8 & 0xff), int(n & 0xff), nil
}

// 16-color name → index.
var color16Names = map[string]int{
	"black":        c16Black,
	"red":          c16Red,
	"green":        c16Green,
	"yellow":       c16Yellow,
	"blue":         c16Blue,
	"magenta":      c16Magenta,
	"cyan":         c16Cyan,
	"white":        c16White,
	"br-black":     c16BrBlack,
	"br-red":       c16BrRed,
	"br-green":     c16BrGreen,
	"br-yellow":    c16BrYellow,
	"br-blue":      c16BrBlue,
	"br-magenta":   c16BrMagenta,
	"br-cyan":      c16BrCyan,
	"br-white":     c16BrWhite,
	// aliases
	"bright-black":   c16BrBlack,
	"bright-red":     c16BrRed,
	"bright-green":   c16BrGreen,
	"bright-yellow":  c16BrYellow,
	"bright-blue":    c16BrBlue,
	"bright-magenta": c16BrMagenta,
	"bright-cyan":    c16BrCyan,
	"bright-white":   c16BrWhite,
	"gray":           c16BrBlack,
	"grey":           c16BrBlack,
}

// ParseColor16 maps a name to 0–15. Numeric strings also accepted.
func ParseColor16(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return -1, fmt.Errorf("prettylog: empty 16-color name")
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n >= 0 && n <= 15 {
			return n, nil
		}
		return -1, fmt.Errorf("prettylog: 16-color index %d out of range", n)
	}
	if n, ok := color16Names[s]; ok {
		return n, nil
	}
	return -1, fmt.Errorf("prettylog: unknown 16-color name %q", s)
}

// specToRecipe builds an sgr recipe from a ColorSpec, falling back to base.
func specToRecipe(base sgr, spec ColorSpec) sgr {
	rec := base

	if spec.Fg != "" {
		if r, g, b, err := ParseHexColor(spec.Fg); err == nil {
			rec = rec.withTrueFg(r, g, b)
		}
	}
	if spec.Bg != "" {
		if r, g, b, err := ParseHexColor(spec.Bg); err == nil {
			rec = rec.withTrueBg(r, g, b)
		}
	}
	if spec.Fg16 != "" {
		if n, err := ParseColor16(spec.Fg16); err == nil {
			rec = rec.withFg16(n)
		}
	}
	if spec.Bg16 != "" {
		if n, err := ParseColor16(spec.Bg16); err == nil {
			rec = rec.withBg16(n)
		}
	}
	if spec.Fg256 != nil && *spec.Fg256 >= 0 && *spec.Fg256 <= 255 {
		rec = rec.withFg256(*spec.Fg256)
	}
	if spec.Bg256 != nil && *spec.Bg256 >= 0 && *spec.Bg256 <= 255 {
		rec = rec.withBg256(*spec.Bg256)
	}
	if spec.Bold != nil {
		rec.bold = *spec.Bold
	}
	return rec
}

// recipesFromConfig replaces built-in recipes with config overrides.
// Called once at handler construction; mutates package-level maps is avoided
// by attaching recipes to the handler via palette fields.
func recipesFromConfig(p PaletteConfig) *recipeSet {
	return &recipeSet{
		info:       specToRecipe(recInfoBadge, p.Info),
		warn:       specToRecipe(recWarnBadge, p.Warn),
		errorBadge: specToRecipe(recErrorBadge, p.Error),
		debug:      specToRecipe(recDebugBadge, p.Debug),
		fatal:      specToRecipe(recFatalBadge, p.Fatal),
		key:        specToRecipe(recKey, p.Key),
		str:        specToRecipe(recString, p.String),
		num:        specToRecipe(recNumber, p.Number),
		boolean:    specToRecipe(recBool, p.Bool),
		errText:    specToRecipe(recErrorText, p.ErrorText),
		message:    specToRecipe(recText, p.Message),
		dim:        specToRecipe(recDim, p.Dim),
		muted:      specToRecipe(recMuted, p.Muted),
		teal:       specToRecipe(recTeal, p.Teal),
		tealBright: specToRecipe(recTealBright, p.TealBright),
	}
}

// recipeSet is the per-handler palette after config merge.
type recipeSet struct {
	info, warn, errorBadge, debug, fatal sgr
	key, str, num, boolean, errText      sgr
	message, dim, muted                  sgr
	teal, tealBright                     sgr
}

func defaultRecipes() *recipeSet {
	return recipesFromConfig(defaultPaletteConfig())
}
