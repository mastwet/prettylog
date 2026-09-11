// Command demo showcases the prettylog library.
//
//	cd examples/demo
//	go run . -color auto -level info -source
//	go run . -color 16
//	go run . -color truecolor -level debug
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/mastwet/prettylog"
	"github.com/mastwet/prettylog/examples/demo/app"
)

func main() {
	var (
		levelFlag = flag.String("level", "info", "minimum level: debug|info|warn|error")
		colorFlag = flag.String("color", "auto", "color: auto|truecolor|256|16|off")
		showSrc   = flag.Bool("source", false, "show file:line")
		port      = flag.Int("port", 8787, "demo listen port")
	)
	flag.Parse()

	mode, ok := prettylog.ParseColorMode(*colorFlag)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown -color %q, using auto\n", *colorFlag)
		mode = prettylog.ColorAuto
	}

	// Load prettylog.json (cwd or default search path) and merge onto defaults.
	// CLI flags still win over the file.
	cfg, cfgPath := prettylog.LoadDefaultConfigFile()
	opts := []prettylog.Option{prettylog.WithConfig(cfg)}
	if cfgPath != "" {
		fmt.Fprintf(os.Stderr, "prettylog config: %s\n", cfgPath)
	}
	opts = append(opts,
		prettylog.WithLevelString(*levelFlag),
		prettylog.WithColor(mode),
		prettylog.WithSource(*showSrc),
	)

	log := prettylog.New(os.Stdout, opts...)

	ctx := context.Background()
	banner(log, *levelFlag)

	srv := app.NewServer(log.Logger, *port)

	chapters := []struct {
		name string
		fn   func()
	}{
		{"BOOT", func() { srv.BootSequence(ctx) }},
		{"HTTP", func() { srv.SimulateHTTP(ctx) }},
		{"STORAGE / CACHE", func() { srv.SimulateDB(ctx) }},
		{"AUTH", srv.SimulateAuth},
		{"ERRORS", srv.SimulateErrors},
		{"CONCURRENCY", func() { srv.SimulateParallel(ctx) }},
		{"SHUTDOWN", srv.Shutdown},
	}

	d := log.Deco()
	for _, ch := range chapters {
		fmt.Println()
		fmt.Println(d.Section(ch.name))
		ch.fn()
	}

	fmt.Println()
	fmt.Println(d.Hint("demo complete — try: go run . -level debug -color 16"))
}

func banner(log *prettylog.Logger, level string) {
	d := log.Deco()
	line := strings.Repeat("─", 56)
	meta := fmt.Sprintf("level=%s  go=%s  color=%s  depth=%s",
		level, runtime.Version(), onOff(log.Colored()), log.Depth())

	fmt.Println()
	fmt.Println(d.BannerLine(line))
	fmt.Println("  " + d.BannerTitle("prettylog"))
	fmt.Println("  " + d.BannerMuted("teal-first slog handler · stdlib only · zero deps"))
	fmt.Println("  " + d.BannerMuted(meta))
	fmt.Println(d.BannerLine(line))
	fmt.Println()

	log.Info("logger initialized",
		"handler", "prettylog",
		"level", level,
		"color", log.Colored(),
		"depth", log.Depth().String(),
		"now", time.Now(),
	)
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
