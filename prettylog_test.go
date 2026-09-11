package prettylog

import (
	"bytes"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPlainOutput(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever), WithLevel(slog.LevelDebug))
	log.Info("hello", "k", "v")
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected no ANSI, got %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("missing message: %q", out)
	}
	if !strings.Contains(out, `k="v"`) {
		t.Fatalf("missing attr: %q", out)
	}
}

func TestForced16ColorHasBadge(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(Color16))
	log.Info("boot")
	out := buf.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI: %q", out)
	}
	if !strings.Contains(out, "INFO") {
		t.Fatalf("missing badge: %q", out)
	}
	// bright cyan bg for INFO chip
	if !strings.Contains(out, "106") {
		t.Fatalf("expected bright-cyan bg 106: %q", out)
	}
}

func TestForcedTrueColor(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorTrueColor))
	log.Info("boot")
	out := buf.String()
	if !strings.Contains(out, "38;2;") {
		t.Fatalf("expected truecolor codes: %q", out)
	}
}

func TestGroupsFlatten(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever))
	log.WithGroup("db").Info("query", "sql", "SELECT 1")
	out := buf.String()
	if !strings.Contains(out, "db.sql=") {
		t.Fatalf("expected flattened group key, got %q", out)
	}
}

func TestErrorValue(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever))
	log.Error("boom", "err", errors.New("dial refused"))
	out := buf.String()
	if !strings.Contains(out, "dial refused") {
		t.Fatalf("missing error text: %q", out)
	}
}

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever), WithLevel(slog.LevelWarn))
	log.Info("skip-me")
	log.Warn("keep-me")
	out := buf.String()
	if strings.Contains(out, "skip-me") {
		t.Fatalf("Info should be filtered: %q", out)
	}
	if !strings.Contains(out, "keep-me") {
		t.Fatalf("Warn should pass: %q", out)
	}
}

func TestReplaceAttrDrops(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever), WithReplaceAttr(func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == "secret" {
			return slog.Attr{}
		}
		return a
	}))
	log.Info("x", "secret", "password", "ok", "1")
	out := buf.String()
	if strings.Contains(out, "password") {
		t.Fatalf("secret leaked: %q", out)
	}
	if !strings.Contains(out, "ok=") {
		t.Fatalf("ok missing: %q", out)
	}
}

func TestConcurrentLinesAreAtomic(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever))
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			log.Info("line", "n", n)
		}(i)
	}
	wg.Wait()
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 50 {
		t.Fatalf("want 50 lines, got %d", len(lines))
	}
	re := regexp.MustCompile(`line.*n=\d+`)
	for i, ln := range lines {
		if !re.MatchString(ln) {
			t.Fatalf("line %d torn: %q", i, ln)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{1500 * time.Millisecond, "1.5s"},
		{2 * time.Millisecond, "2ms"},
		{250 * time.Microsecond, "250µs"},
	}
	for _, c := range cases {
		if got := HumanDuration(c.d); got != c.want {
			t.Fatalf("HumanDuration(%v)=%q want %q", c.d, got, c.want)
		}
	}
}

func TestParseColorMode(t *testing.T) {
	for s, want := range map[string]ColorMode{
		"auto":      ColorAuto,
		"truecolor": ColorTrueColor,
		"256":       Color256,
		"16":        Color16,
		"off":       ColorNever,
		"never":     ColorNever,
	} {
		got, ok := ParseColorMode(s)
		if !ok || got != want {
			t.Fatalf("ParseColorMode(%q)=%v,%v want %v,true", s, got, ok, want)
		}
	}
	if _, ok := ParseColorMode("nope"); ok {
		t.Fatal("expected unknown mode to fail")
	}
}

func TestParseLevel(t *testing.T) {
	if ParseLevel("debug") != slog.LevelDebug {
		t.Fatal("debug")
	}
	if ParseLevel("warn") != slog.LevelWarn {
		t.Fatal("warn")
	}
	if ParseLevel("error") != slog.LevelError {
		t.Fatal("error")
	}
	if ParseLevel("nope") != slog.LevelInfo {
		t.Fatal("default info")
	}
}

func TestDepthString(t *testing.T) {
	if DepthTrueColor.String() != "truecolor" {
		t.Fatal(DepthTrueColor.String())
	}
	if DepthANSI16.String() != "16" {
		t.Fatal(DepthANSI16.String())
	}
}

func TestLoggerDepthHelpers(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, WithColor(Color256))
	if !l.Colored() {
		t.Fatal("expected colored")
	}
	if l.Depth() != DepthANSI256 {
		t.Fatalf("depth=%v", l.Depth())
	}
	off := New(&buf, WithColor(ColorNever))
	if off.Colored() || off.Depth() != DepthOff {
		t.Fatal("expected off")
	}
}

func TestSourceColumn(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever), WithSource(true))
	log.Info("here")
	out := buf.String()
	if !strings.Contains(out, "prettylog_test.go:") {
		t.Fatalf("expected source column: %q", out)
	}
}
