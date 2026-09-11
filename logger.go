package prettylog

import (
	"io"
	"log/slog"
	"os"
)

// Logger wraps *slog.Logger with a few prettylog conveniences.
// It is safe for concurrent use.
type Logger struct {
	*slog.Logger
	handler *Handler
}

// New returns a Logger writing pretty records to w.
//
//	log := prettylog.New(os.Stdout, prettylog.WithLevel(slog.LevelDebug))
func New(w io.Writer, opts ...Option) *Logger {
	h := NewHandler(w, opts...)
	return &Logger{Logger: slog.New(h), handler: h}
}

// NewStdout is New(os.Stdout, opts...).
func NewStdout(opts ...Option) *Logger {
	return New(os.Stdout, opts...)
}

// Handler returns the underlying slog.Handler.
func (l *Logger) Handler() slog.Handler { return l.handler }

// Depth reports the resolved color depth.
func (l *Logger) Depth() ColorDepth { return l.handler.pal.depth }

// Colored reports whether ANSI output is enabled.
func (l *Logger) Colored() bool { return l.handler.pal.on }

// Deco returns banner helpers matching this logger's palette.
func (l *Logger) Deco() Deco {
	return Deco{pal: l.handler.pal}
}

// SetDefault installs a pretty handler as the global slog default
// and returns the Logger. Convenient for main().
//
//	log := prettylog.SetDefault(prettylog.WithLevel(slog.LevelInfo))
func SetDefault(opts ...Option) *Logger {
	l := New(os.Stdout, opts...)
	slog.SetDefault(l.Logger)
	return l
}
