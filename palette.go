package prettylog

import "fmt"

// Teal-first palette with graceful degradation:
//   truecolor → 256-color → 16-color → plain
//
// Badge chips use solid backgrounds at every depth so the level block
// reads the same on modern terminals and legacy conhost.

const (
	sgrReset = "\x1b[0m"
	sgrBold  = "1"
	sgrDim   = "2"
	sgrItal  = "3"
)

// PaintReset and PaintDim are exported for callers that want matching decoration.
const (
	PaintReset = sgrReset
	PaintDim   = "\x1b[2m"
)

type sgr struct {
	fg16, bg16   int // 0–15, -1 = unused
	fg256, bg256 int // 0–255, -1 = unused
	r, g, b      int
	br, bgg, bb  int
	bgValid      bool
	bold, dim, ital bool
}

func style(bold, dim, ital bool) sgr {
	return sgr{fg16: -1, bg16: -1, fg256: -1, bg256: -1, bold: bold, dim: dim, ital: ital}
}

func (s sgr) withFg16(c int) sgr { s.fg16 = c; return s }
func (s sgr) withBg16(c int) sgr { s.bg16 = c; return s }
func (s sgr) withFg256(n int) sgr { s.fg256 = n; return s }
func (s sgr) withBg256(n int) sgr { s.bg256 = n; return s }
func (s sgr) withTrueFg(r, g, b int) sgr { s.r, s.g, s.b = r, g, b; return s }
func (s sgr) withTrueBg(r, g, b int) sgr {
	s.br, s.bgg, s.bb = r, g, b
	s.bgValid = true
	return s
}

func (s sgr) encode(depth ColorDepth) []string {
	var out []string
	if s.bold {
		out = append(out, sgrBold)
	}
	if s.dim {
		out = append(out, sgrDim)
	}
	if s.ital {
		out = append(out, sgrItal)
	}

	switch depth {
	case DepthTrueColor:
		out = append(out, fmt.Sprintf("38;2;%d;%d;%d", s.r, s.g, s.b))
		if s.bgValid {
			out = append(out, fmt.Sprintf("48;2;%d;%d;%d", s.br, s.bgg, s.bb))
		}
	case DepthANSI256:
		if s.fg256 >= 0 {
			out = append(out, fmt.Sprintf("38;5;%d", s.fg256))
		} else if s.fg16 >= 0 {
			out = append(out, fg16Code(s.fg16))
		}
		if s.bg256 >= 0 {
			out = append(out, fmt.Sprintf("48;5;%d", s.bg256))
		} else if s.bg16 >= 0 {
			out = append(out, bg16Code(s.bg16))
		}
	case DepthANSI16:
		if s.fg16 >= 0 {
			out = append(out, fg16Code(s.fg16))
		}
		if s.bg16 >= 0 {
			out = append(out, bg16Code(s.bg16))
		}
	}
	return out
}

func fg16Code(i int) string {
	if i >= 8 {
		return fmt.Sprintf("%d", 90+(i-8))
	}
	return fmt.Sprintf("%d", 30+i)
}

func bg16Code(i int) string {
	if i >= 8 {
		return fmt.Sprintf("%d", 100+(i-8))
	}
	return fmt.Sprintf("%d", 40+i)
}

// 16-color indexes.
const (
	c16Black = iota
	c16Red
	c16Green
	c16Yellow
	c16Blue
	c16Magenta
	c16Cyan
	c16White
	c16BrBlack
	c16BrRed
	c16BrGreen
	c16BrYellow
	c16BrBlue
	c16BrMagenta
	c16BrCyan
	c16BrWhite
)

// Semantic recipes.
var (
	recInfoBadge = style(true, false, false).
			withTrueBg(13, 148, 136).withTrueFg(4, 47, 46).
			withBg256(30).withFg256(16).
			withBg16(c16BrCyan).withFg16(c16Black)

	recWarnBadge = style(true, false, false).
			withTrueBg(180, 83, 9).withTrueFg(254, 243, 199).
			withBg256(130).withFg256(230).
			withBg16(c16BrYellow).withFg16(c16Black)

	recErrorBadge = style(true, false, false).
			withTrueBg(190, 24, 93).withTrueFg(255, 241, 242).
			withBg256(161).withFg256(231).
			withBg16(c16BrRed).withFg16(c16BrWhite)

	recDebugBadge = style(true, false, false).
			withTrueBg(51, 65, 85).withTrueFg(226, 232, 240).
			withBg256(238).withFg256(255).
			withBg16(c16BrBlack).withFg16(c16BrWhite)

	recFatalBadge = style(true, false, false).
			withTrueBg(126, 34, 206).withTrueFg(243, 232, 255).
			withBg256(99).withFg256(255).
			withBg16(c16BrMagenta).withFg16(c16BrWhite)

	recKey = style(true, false, false).
		withTrueFg(94, 234, 212).
		withFg256(122).
		withFg16(c16BrCyan)

	recString = style(false, false, false).
			withTrueFg(167, 243, 208).
			withFg256(158).
			withFg16(c16BrGreen)

	recNumber = style(false, false, false).
			withTrueFg(251, 191, 36).
			withFg256(220).
			withFg16(c16Yellow)

	recBool = style(false, false, false).
			withTrueFg(196, 181, 253).
			withFg256(183).
			withFg16(c16Magenta)

	recErrorText = style(true, false, false).
			withTrueFg(251, 113, 133).
			withFg256(211).
			withFg16(c16BrRed)

	recText = style(true, false, false).
			withTrueFg(226, 245, 240).
			withFg256(231).
			withFg16(c16BrWhite)

	recDim = style(false, false, true).
		withTrueFg(110, 138, 138).
		withFg256(245).
		withFg16(c16BrBlack)

	recMuted = style(false, false, false).
			withTrueFg(80, 104, 104).
			withFg256(240).
			withFg16(c16BrBlack)

	recTeal = style(false, false, false).
		withTrueFg(45, 212, 191).
		withFg256(43).
		withFg16(c16Cyan)

	recTealBright = style(true, false, false).
			withTrueFg(94, 234, 212).
			withFg256(122).
			withFg16(c16BrCyan)
)

type palette struct {
	on    bool
	depth ColorDepth
	rs    *recipeSet
}

func newPalette(on bool, depth ColorDepth) palette {
	return newPaletteWith(on, depth, defaultRecipes())
}

func newPaletteWith(on bool, depth ColorDepth, rs *recipeSet) palette {
	if rs == nil {
		rs = defaultRecipes()
	}
	if !on || depth == DepthOff {
		return palette{on: false, depth: DepthOff, rs: rs}
	}
	return palette{on: true, depth: depth, rs: rs}
}

func (p palette) wrap(rec sgr, s string) string {
	if !p.on {
		return s
	}
	params := rec.encode(p.depth)
	if len(params) == 0 {
		return s
	}
	return "\x1b[" + stringsJoin(params, ";") + "m" + s + sgrReset
}

func stringsJoin(parts []string, sep string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	n := len(sep) * (len(parts) - 1)
	for _, p := range parts {
		n += len(p)
	}
	b := make([]byte, 0, n)
	for i, p := range parts {
		if i > 0 {
			b = append(b, sep...)
		}
		b = append(b, p...)
	}
	return string(b)
}

func (p palette) levelBadge(level string) string {
	label := " " + level + " "
	rs := p.rs
	switch level {
	case "DEBUG":
		return p.wrap(rs.debug, label)
	case "INFO":
		return p.wrap(rs.info, label)
	case "WARN":
		return p.wrap(rs.warn, label)
	case "ERROR":
		return p.wrap(rs.errorBadge, label)
	case "FATAL":
		return p.wrap(rs.fatal, label)
	default:
		return p.wrap(rs.tealBright, label)
	}
}

func (p palette) key(k string) string         { return p.wrap(p.rs.key, k) }
func (p palette) stringValue(v string) string { return p.wrap(p.rs.str, `"`+v+`"`) }
func (p palette) quoted(s string) string      { return p.wrap(p.rs.str, s) }
func (p palette) number(v string) string      { return p.wrap(p.rs.num, v) }
func (p palette) boolean(v string) string     { return p.wrap(p.rs.boolean, v) }
func (p palette) errText(v string) string     { return p.wrap(p.rs.errText, v) }
func (p palette) timeText(t string) string    { return p.wrap(p.rs.dim, t) }
func (p palette) source(s string) string      { return p.wrap(p.rs.muted, s) }
func (p palette) message(m string) string     { return p.wrap(p.rs.message, m) }
func (p palette) sep(s string) string         { return p.wrap(p.rs.muted, s) }
func (p palette) teal(s string) string        { return p.wrap(p.rs.teal, s) }
func (p palette) tealBright(s string) string  { return p.wrap(p.rs.tealBright, s) }
func (p palette) dimText(s string) string     { return p.wrap(p.rs.dim, s) }

// Deco exposes banner/section helpers for apps that want matching chrome.
type Deco struct{ pal palette }

// NewDeco returns a decorator for the given color settings (built-in palette).
func NewDeco(mode ColorMode, w FileStat) Deco {
	on, depth := resolve(mode, w)
	return Deco{pal: newPalette(on, depth)}
}

// NewDecoFromConfig returns a decorator using a loaded Config's palette.
func NewDecoFromConfig(cfg Config, w FileStat) Deco {
	mode, _ := ParseColorMode(cfg.Color)
	on, depth := resolve(mode, w)
	rs := recipesFromConfig(cfg.Palette)
	return Deco{pal: newPaletteWith(on, depth, rs)}
}

func (d Deco) BannerLine(s string) string  { return d.pal.teal(s) }
func (d Deco) BannerTitle(s string) string { return d.pal.tealBright(s) }
func (d Deco) BannerMuted(s string) string { return d.pal.dimText(s) }
func (d Deco) Section(name string) string  { return d.pal.dimText("── " + name + " ──") }
func (d Deco) Hint(s string) string        { return d.pal.dimText(s) }
