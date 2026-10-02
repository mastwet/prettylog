package prettylog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Handler is a slog.Handler that renders single-line, colorized records.
//
// Layout:
//
//	15:04:05.000  INFO  message  │  key="v" n=1 flag=true
//	              ^src  ^msg      ^attrs (groups flattened as a.b)
type Handler struct {
	opts Options
	pal  palette
	w    io.Writer
	mu   *sync.Mutex
	// pre-attached attrs/groups from WithAttrs / WithGroup
	attrs  []scopedAttrs
	groups []string
}

// Each batch retains the groups that were open when WithAttrs was called.
type scopedAttrs struct {
	attrs  []slog.Attr
	groups []string
}

// NewHandler creates a pretty console handler.
func NewHandler(w io.Writer, opts ...Option) *Handler {
	o := applyOptions(opts)
	on, depth := resolve(o.Color, asStat(w))
	rs := defaultRecipes()
	if o.Config != nil {
		// Palette from a full DefaultConfig merge so omitted fields keep
		// the teal built-ins.
		full := DefaultConfig().Merge(*o.Config)
		rs = recipesFromConfig(full.Palette)
	}
	return &Handler{
		opts: o,
		pal:  newPaletteWith(on, depth, rs),
		w:    w,
		mu:   &sync.Mutex{},
	}
}

// asStat adapts an io.Writer to FileStat when it can report TTY state.
func asStat(w io.Writer) FileStat {
	if f, ok := w.(FileStat); ok {
		return f
	}
	return nil
}

// Enabled reports whether records at level should be emitted.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle renders one record.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var b strings.Builder

	// Timestamp
	if !r.Time.IsZero() {
		b.WriteString(h.pal.timeText(singleLine(r.Time.Format(h.opts.TimeFormat))))
		b.WriteString("  ")
	}

	// Level badge
	b.WriteString(h.pal.levelBadge(r.Level.String()))
	b.WriteString("  ")

	var source *slog.Source
	if (h.opts.ShowSource || h.opts.AddSource) && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		if f, _ := fs.Next(); f.File != "" {
			source = &slog.Source{Function: f.Function, File: f.File, Line: f.Line}
		}
	}
	if h.opts.ShowSource && source != nil {
		b.WriteString(h.pal.source(singleLine(trimSource(source.File, source.Line))))
		b.WriteString("  ")
	}

	// Message
	if r.Message == "" {
		b.WriteString(h.pal.sep("(no message)"))
	} else {
		b.WriteString(h.pal.message(singleLine(r.Message)))
	}

	// Source is a record-level attribute, outside any WithGroup scope.
	var parts []string
	if h.opts.AddSource && source != nil {
		parts = append(parts, h.formatAttr(nil, slog.Any(slog.SourceKey, source))...)
	}
	for _, batch := range h.attrs {
		parts = append(parts, h.formatAttrs(batch.attrs, batch.groups)...)
	}
	r.Attrs(func(a slog.Attr) bool {
		parts = append(parts, h.formatAttr(h.groups, a)...)
		return true
	})

	if len(parts) > 0 {
		b.WriteString("  ")
		b.WriteString(h.pal.sep("│"))
		b.WriteString("  ")
		b.WriteString(stringsJoin(parts, "  "))
	}

	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

// WithAttrs returns a child handler with extra bound attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	nh := *h
	nh.attrs = make([]scopedAttrs, 0, len(h.attrs)+1)
	nh.attrs = append(nh.attrs, h.attrs...)
	nh.attrs = append(nh.attrs, scopedAttrs{attrs: attrs, groups: h.groups})
	return &nh
}

// WithGroup returns a child handler that prefixes attribute keys with name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	nh := *h
	nh.groups = append(append([]string{}, h.groups...), name)
	return &nh
}

func (h *Handler) formatAttrs(attrs []slog.Attr, groups []string) []string {
	parts := make([]string, 0, len(attrs))
	for _, a := range attrs {
		parts = append(parts, h.formatAttr(groups, a)...)
	}
	return parts
}

func (h *Handler) formatAttr(groups []string, a slog.Attr) []string {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return nil
	}
	if a.Value.Kind() != slog.KindGroup && h.opts.ReplaceAttr != nil {
		a = h.opts.ReplaceAttr(groups, a)
		a.Value = a.Value.Resolve()
		if a.Equal(slog.Attr{}) {
			return nil
		}
	}

	// Flatten groups as parent.child.
	if a.Value.Kind() == slog.KindGroup {
		inner := a.Value.Group()
		if a.Key != "" {
			groups = append(append([]string(nil), groups...), a.Key)
		}
		var out []string
		for _, ga := range inner {
			out = append(out, h.formatAttr(groups, ga)...)
		}
		return out
	}

	key := a.Key
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}
	return []string{h.pal.key(singleLine(key)) + "=" + h.formatValue(a.Value)}
}

func (h *Handler) formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return h.pal.quoted(strconv.Quote(v.String()))
	case slog.KindInt64:
		return h.pal.number(strconv.FormatInt(v.Int64(), 10))
	case slog.KindUint64:
		return h.pal.number(strconv.FormatUint(v.Uint64(), 10))
	case slog.KindFloat64:
		return h.pal.number(strconv.FormatFloat(v.Float64(), 'f', -1, 64))
	case slog.KindBool:
		return h.pal.boolean(strconv.FormatBool(v.Bool()))
	case slog.KindDuration:
		return h.pal.number(HumanDuration(v.Duration()))
	case slog.KindTime:
		return h.pal.timeText(v.Time().Format(time.RFC3339))
	case slog.KindAny:
		switch x := v.Any().(type) {
		case *slog.Source:
			if x == nil {
				return h.pal.sep("null")
			}
			return h.pal.source(singleLine(trimSource(x.File, x.Line)))
		case error:
			if x == nil {
				return h.pal.sep("null")
			}
			return h.pal.errText(singleLine(x.Error()))
		case fmt.Stringer:
			return h.pal.quoted(strconv.Quote(x.String()))
		default:
			return h.pal.quoted(singleLine(fmt.Sprint(x)))
		}
	default:
		return h.pal.quoted(singleLine(fmt.Sprint(v.Any())))
	}
}

// Escape user-supplied controls before adding the palette's ANSI sequences.
func singleLine(s string) string {
	isControl := func(r rune) bool {
		return unicode.IsControl(r) || r == '\u2028' || r == '\u2029'
	}
	if strings.IndexFunc(s, isControl) < 0 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if isControl(r) {
			quoted := strconv.QuoteRune(r)
			b.WriteString(quoted[1 : len(quoted)-1])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// HumanDuration formats d with a readable precision.
func HumanDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return d.Round(time.Millisecond).String()
	case d >= time.Millisecond:
		return d.Round(100 * time.Microsecond).String()
	default:
		return d.Round(time.Microsecond).String()
	}
}

func trimSource(file string, line int) string {
	file = strings.ReplaceAll(file, "\\", "/")
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return fmt.Sprintf("%s:%d", parts[len(parts)-1], line)
}
