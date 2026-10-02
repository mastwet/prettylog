package prettylog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWithGroupPreservesBoundAttributeScopes(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever)).
		With("run_id", "r1").
		WithGroup("model").
		With("provider", "deepseek").
		WithGroup("usage")
	log.Info("call", "tokens", 7)
	_, attrs, ok := strings.Cut(buf.String(), "│  ")
	want := "run_id=\"r1\"  model.provider=\"deepseek\"  model.usage.tokens=7\n"
	if !ok || attrs != want {
		t.Fatalf("attribute scopes: got %q, want %q", buf.String(), want)
	}
}

type logValue slog.Value

func (v logValue) LogValue() slog.Value { return slog.Value(v) }

func TestReplaceAttrReceivesResolvedLeavesAndFullGroups(t *testing.T) {
	var buf bytes.Buffer
	var seen []string
	log := New(&buf, WithColor(ColorNever), WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
		if a.Value.Kind() == slog.KindGroup || a.Value.Kind() == slog.KindLogValuer {
			t.Errorf("hook received unresolved or group attribute: %v", a)
		}
		seen = append(seen, strings.Join(append(append([]string(nil), groups...), a.Key), "."))
		if a.Key == "token" {
			if a.Value.Kind() != slog.KindString || a.Value.String() != "secret" {
				t.Errorf("token was not resolved before redaction: %v", a)
			}
			return slog.String(a.Key, "[redacted]")
		}
		if a.Key == "at" {
			return slog.String(a.Key, "hidden")
		}
		return a
	}))
	log.With("run_id", "r1").WithGroup("request").
		With(slog.Group("auth", slog.Any("token", logValue(slog.StringValue("secret"))))).
		Info("call", slog.Group("usage", slog.Int("tokens", 7), slog.Time("at", time.Now())))
	want := []string{"run_id", "request.auth.token", "request.usage.tokens", "request.usage.at"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("hook paths: got %v, want %v", seen, want)
	}
	out := buf.String()
	if strings.Contains(out, "secret") || !strings.Contains(out, `request.auth.token="[redacted]"`) ||
		!strings.Contains(out, `request.usage.at="hidden"`) {
		t.Fatalf("redaction was not applied to resolved leaves: %q", out)
	}
}

func TestReplaceAttrResolvesReturnedValues(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever), WithReplaceAttr(func(_ []string, a slog.Attr) slog.Attr {
		return slog.Any(a.Key, logValue(slog.StringValue("replacement")))
	}))
	log.Info("call", "key", "original")
	if !strings.Contains(buf.String(), `key="replacement"`) {
		t.Fatalf("replacement value not resolved: %q", buf.String())
	}
}

func TestAddSourceEmitsSourceAndAllowsReplacement(t *testing.T) {
	t.Run("source", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(&buf, WithColor(ColorNever), func(o *Options) { o.AddSource = true })
		log.WithGroup("call").Info("here", "id", 1)
		out := buf.String()
		if !strings.Contains(out, "source=") || !strings.Contains(out, "handler_contract_test.go:") ||
			strings.Contains(out, "call.source=") {
			t.Fatalf("missing ungrouped source: %q", out)
		}
	})
	t.Run("replace", func(t *testing.T) {
		var buf bytes.Buffer
		var sawSource bool
		log := New(&buf, WithColor(ColorNever), func(o *Options) { o.AddSource = true },
			WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					sawSource = true
					source, ok := a.Value.Any().(*slog.Source)
					if len(groups) != 0 || !ok || source.Line == 0 || source.File == "" {
						t.Errorf("invalid source hook input: groups=%v attr=%v", groups, a)
					}
					return slog.String(a.Key, "mapped.go:42")
				}
				return a
			}))
		log.WithGroup("call").Info("here", "id", 1)
		if !sawSource || !strings.Contains(buf.String(), `source="mapped.go:42"`) ||
			strings.Contains(buf.String(), "handler_contract_test.go:") {
			t.Fatalf("source replacement was not applied: %q", buf.String())
		}
	})
	t.Run("missing pc", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHandler(&buf, WithColor(ColorNever), func(o *Options) { o.AddSource = true })
		if err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "here", 0)); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(buf.String(), "source=") {
			t.Fatalf("source emitted without a program counter: %q", buf.String())
		}
	})
}

type textValue string

func (v textValue) String() string { return string(v) }

func TestControlCharactersDoNotSplitRecords(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode ColorMode
	}{{"plain", ColorNever}, {"color", Color16}} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, WithColor(tc.mode), WithTimeFormat("15:04:05\n"))
			log.WithGroup("model\n").Error("failed\nnext\r\t\x1b[31m",
				"err", errors.New("first\nsecond\r\t\x1b[31m"),
				"detail", textValue("first\nsecond"),
				"payload", []string{"first\nsecond"},
				"key\n", "value")
			out := buf.String()
			if strings.Count(out, "\n") != 1 || !strings.HasSuffix(out, "\n") || strings.ContainsAny(out, "\r\t") {
				t.Fatalf("record contains raw line controls: %q", out)
			}
			for _, escaped := range []string{`failed\nnext\r\t\x1b[31m`, `first\nsecond`, `model\n.key\n`} {
				if !strings.Contains(out, escaped) {
					t.Errorf("missing escaped text %q in %q", escaped, out)
				}
			}
			if tc.mode == ColorNever && strings.Contains(out, "\x1b") {
				t.Fatalf("plain output contains ANSI: %q", out)
			}
		})
	}
}

func TestStringAttributesEscapeQuotesAndBackslashes(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, WithColor(ColorNever)).Info("call", "payload", "say \"hi\" at C:\\tmp")
	if !strings.Contains(buf.String(), `payload="say \"hi\" at C:\\tmp"`) {
		t.Fatalf("string is not quoted unambiguously: %q", buf.String())
	}
}

func TestEmptyAttributesDoNotLeaveSeparator(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, WithColor(ColorNever), WithReplaceAttr(func(_ []string, a slog.Attr) slog.Attr {
		return slog.Attr{}
	}))
	log.With(slog.Attr{}).Info("call", "secret", "hidden", slog.Group("empty"))
	if strings.Contains(buf.String(), "│") || strings.Contains(buf.String(), "hidden") {
		t.Fatalf("empty attributes left output behind: %q", buf.String())
	}
}

func TestZeroTimeOmitsTimestamp(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, WithColor(ColorNever))
	if err := h.Handle(context.Background(), slog.NewRecord(time.Time{}, slog.LevelInfo, "call", 0)); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), " INFO ") {
		t.Fatalf("zero timestamp was rendered: %q", buf.String())
	}
}
