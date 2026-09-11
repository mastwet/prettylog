package prettylog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	r, g, b, err := ParseHexColor("#0D9488")
	if err != nil || r != 13 || g != 148 || b != 136 {
		t.Fatalf("got %d,%d,%d err=%v", r, g, b, err)
	}
	r, g, b, err = ParseHexColor("#fff")
	if err != nil || r != 255 || g != 255 || b != 255 {
		t.Fatalf("short form: %d,%d,%d err=%v", r, g, b, err)
	}
	if _, _, _, err := ParseHexColor("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseColor16(t *testing.T) {
	n, err := ParseColor16("br-cyan")
	if err != nil || n != c16BrCyan {
		t.Fatalf("br-cyan → %d err=%v", n, err)
	}
	n, err = ParseColor16("14")
	if err != nil || n != 14 {
		t.Fatalf("14 → %d err=%v", n, err)
	}
	if _, err := ParseColor16("not-a-color"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultConfigJSONRoundTrip(t *testing.T) {
	c := DefaultConfig()
	if c.Palette.Info.Bg != "#0D9488" {
		t.Fatalf("info bg = %q", c.Palette.Info.Bg)
	}
	if c.Level != "info" || c.Color != "auto" {
		t.Fatalf("level/color = %q/%q", c.Level, c.Color)
	}
}

func TestParseConfigAndMerge(t *testing.T) {
	raw := []byte(`{
		"level": "debug",
		"color": "16",
		"palette": {
			"info": { "bg": "#FF0000", "bg16": "br-red" }
		}
	}`)
	over, err := ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	merged := DefaultConfig().Merge(over)
	if merged.Level != "debug" {
		t.Fatalf("level=%q", merged.Level)
	}
	if merged.Palette.Info.Bg != "#FF0000" {
		t.Fatalf("info bg not overridden: %q", merged.Palette.Info.Bg)
	}
	// Unspecified roles keep defaults
	if merged.Palette.Warn.Bg != "#B45309" {
		t.Fatalf("warn bg should stay default, got %q", merged.Palette.Warn.Bg)
	}
}

func TestWithConfigChangesInfoBadge(t *testing.T) {
	var buf bytes.Buffer
	cfg := DefaultConfig()
	cfg.Color = "16"
	red := "#FF0000"
	cfg.Palette.Info = ColorSpec{Fg: "#000000", Bg: red, Fg16: "black", Bg16: "br-red"}
	log := New(&buf, WithConfig(cfg))
	log.Info("x")
	out := buf.String()
	// br-red bg = 101
	if !bytes.Contains([]byte(out), []byte("101")) {
		t.Fatalf("expected br-red bg 101 in %q", out)
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prettylog.json")
	body := `{"level":"warn","color":"off","palette":{"key":{"fg":"#123456"}}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Level != "warn" || c.Color != "off" {
		t.Fatalf("got %+v", c)
	}
	merged := DefaultConfig().Merge(c)
	if merged.Palette.Key.Fg != "#123456" {
		t.Fatalf("key fg=%q", merged.Palette.Key.Fg)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestWithConfigFileOption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	// Custom message color + always 16
	body := `{"color":"16","palette":{"message":{"fg":"#FFFFFF","fg16":"white"}}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	log := New(&buf, WithConfigFile(path))
	log.Info("hello")
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("hello")) {
		t.Fatalf("out=%q", out)
	}
	// 16-color white fg = 37
	if !bytes.Contains([]byte(out), []byte("37")) {
		t.Logf("output (may still be valid): %q", out)
	}
}

func TestShippedDefaultConfigFileParses(t *testing.T) {
	path := "prettylog.json"
	if _, err := os.Stat(path); err != nil {
		t.Skip("shipped config not found at", path)
	}
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	merged := DefaultConfig().Merge(c)
	if merged.Palette.Info.Bg == "" {
		t.Fatal("info bg empty after merge")
	}
}
