// Package prettylog is a zero-dependency, color-aware slog.Handler.
//
// It renders human-friendly single-line records on a teal-first palette
// and degrades cleanly across truecolor → 256-color → 16-color → plain.
//
// Quick start:
//
//	log := prettylog.New(os.Stdout)
//	log.Info("service ready", "addr", "127.0.0.1:8787")
//
// Custom options:
//
//	log := prettylog.New(os.Stdout,
//		prettylog.WithLevel(slog.LevelDebug),
//		prettylog.WithSource(true),
//		prettylog.WithColor(prettylog.ColorAuto),
//	)
//
// As a slog.Handler:
//
//	h := prettylog.NewHandler(os.Stdout, prettylog.WithLevel(slog.LevelInfo))
//	logger := slog.New(h)
//
// The package is safe for concurrent use.
package prettylog
