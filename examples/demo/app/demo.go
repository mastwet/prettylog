// Package app emits realistic sample logs so the pretty handler has
// something meaty to render: boot, HTTP, DB, cache, auth, errors, shutdown.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// RequestID is the classic middleware key stored in context.
type ctxKey string

const requestIDKey ctxKey = "request_id"

// WithRequestID returns a child context carrying the request id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extracts the request id, or empty.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// Server is a tiny fake HTTP service used only for log demos.
type Server struct {
	log  *slog.Logger
	port int
}

func NewServer(log *slog.Logger, port int) *Server {
	return &Server{log: log, port: port}
}

// BootSequence writes a rich startup narrative.
func (s *Server) BootSequence(ctx context.Context) {
	s.log.Info("starting vivy-demo service",
		slog.String("version", "0.4.2"),
		slog.String("commit", "a1b2c3d"),
		slog.String("env", "dev"),
		slog.Int("port", s.port),
		slog.Time("built_at", time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)),
	)

	// Nested config group
	cfg := s.log.WithGroup("config")
	cfg.Info("loaded configuration",
		slog.String("source", "config.yaml"),
		slog.Duration("elapsed", 12*time.Millisecond),
		slog.Group("storage",
			slog.String("backend", "sqlite"),
			slog.String("path", "data/demo.db"),
			slog.Int("max_open_conns", 8),
		),
		slog.Group("llm",
			slog.String("provider", "openai-compatible"),
			slog.String("model", "gpt-4o-mini"),
			slog.Float64("temperature", 0.3),
		),
	)

	// Module registration
	for _, m := range []struct {
		name    string
		version string
		ports   []string
	}{
		{"channel.telegram", "1.2.0", []string{"std/message", "std/channel"}},
		{"face.default", "0.9.1", []string{"std/face"}},
		{"skill.runner", "0.3.0", []string{"std/skill", "std/tool"}},
	} {
		s.log.Info("module registered",
			slog.String("module", m.name),
			slog.String("version", m.version),
			slog.Any("ports", m.ports),
		)
	}

	s.log.Warn("demo mode enabled — credentials are synthetic",
		slog.Bool("allow_network", false),
		slog.String("reason", "no real provider keys in this demo"),
	)

	s.log.Info("service ready",
		slog.String("addr", fmt.Sprintf("127.0.0.1:%d", s.port)),
		slog.Duration("boot_time", 842*time.Millisecond),
	)
}

// SimulateHTTP walks a few fake requests with middleware-style logging.
func (s *Server) SimulateHTTP(ctx context.Context) {
	routes := []struct {
		method string
		path   string
		status int
		ms     int
		bytes  int
	}{
		{"GET", "/health", 200, 2, 17},
		{"POST", "/rpc/session.create", 200, 41, 256},
		{"GET", "/rpc/run.list", 200, 18, 1840},
		{"POST", "/rpc/tool.execute", 200, 128, 512},
		{"GET", "/rpc/missing", 404, 1, 48},
		{"POST", "/rpc/run.create", 500, 903, 96},
	}

	for i, r := range routes {
		reqID := fmt.Sprintf("req_%04d", 1000+i)
		reqCtx := WithRequestID(ctx, reqID)

		reqLog := s.log.With(
			slog.String("request_id", reqID),
			slog.String("method", r.method),
			slog.String("path", r.path),
		)
		reqLog.Info("incoming request",
			slog.String("remote", "127.0.0.1"),
			slog.String("user_agent", "vivy-ui/0.4"),
		)

		// Simulate work
		time.Sleep(time.Duration(r.ms/10) * time.Millisecond)

		switch {
		case r.status >= 500:
			reqLog.Error("request failed",
				slog.Int("status", r.status),
				slog.Duration("latency", time.Duration(r.ms)*time.Millisecond),
				slog.Int("bytes", r.bytes),
				slog.String("error", "upstream provider timeout"),
				slog.Group("upstream",
					slog.String("provider", "openai"),
					slog.Duration("timeout", 900*time.Millisecond),
					slog.Int("attempt", 2),
				),
			)
		case r.status >= 400:
			reqLog.Warn("client error",
				slog.Int("status", r.status),
				slog.Duration("latency", time.Duration(r.ms)*time.Millisecond),
				slog.String("hint", "route not registered"),
			)
		default:
			reqLog.Info("request completed",
				slog.Int("status", r.status),
				slog.Duration("latency", time.Duration(r.ms)*time.Millisecond),
				slog.Int("bytes", r.bytes),
			)
		}
		_ = reqCtx
	}
}

// SimulateDB emits SQL / cache / journal activity.
func (s *Server) SimulateDB(ctx context.Context) {
	db := s.log.WithGroup("db")

	db.Info("opening database",
		slog.String("driver", "sqlite"),
		slog.String("dsn", "file:data/demo.db?_pragma=busy_timeout(5000)"),
	)

	queries := []struct {
		sql   string
		rows  int
		took  time.Duration
		err   error
	}{
		{"SELECT id, title FROM sessions WHERE archived=0", 3, 180 * time.Microsecond, nil},
		{"SELECT * FROM runs WHERE session_id=? ORDER BY created_at DESC LIMIT 20", 20, 2 * time.Millisecond, nil},
		{"INSERT INTO journal_entries (kind, payload) VALUES (?, ?)", 1, 900 * time.Microsecond, nil},
		{"UPDATE sessions SET last_active_at=? WHERE id=?", 1, 120 * time.Microsecond, nil},
		{"SELECT * FROM missing_table", 0, 40 * time.Microsecond, errors.New("no such table: missing_table")},
	}

	for _, q := range queries {
		if q.err != nil {
			db.Error("query failed",
				slog.String("sql", q.sql),
				slog.Duration("took", q.took),
				slog.Any("err", q.err),
			)
			continue
		}
		db.Debug("query ok",
			slog.String("sql", q.sql),
			slog.Int("rows", q.rows),
			slog.Duration("took", q.took),
		)
	}

	// Cache layer
	cache := s.log.WithGroup("cache")
	hits, misses := 17, 3
	cache.Info("cache stats",
		slog.Int("hits", hits),
		slog.Int("misses", misses),
		slog.Float64("hit_ratio", float64(hits)/float64(hits+misses)),
		slog.Int("entries", 42),
		slog.Duration("ttl_default", 5*time.Minute),
	)
	cache.Warn("cache entry evicted",
		slog.String("key", "skill:runner:manifest"),
		slog.String("reason", "ttl_expired"),
	)

	// Journal append
	s.log.Info("journal entry appended",
		slog.String("kind", "run.started"),
		slog.String("run_id", "run_9f3a"),
		slog.Int("payload_bytes", 384),
	)
}

// SimulateAuth emits login / token / permission lines.
func (s *Server) SimulateAuth() {
	auth := s.log.WithGroup("auth")

	auth.Info("user authenticated",
		slog.String("user", "alice@vivy.local"),
		slog.String("method", "password"),
		slog.Duration("took", 34*time.Millisecond),
		slog.Bool("mfa", true),
	)

	auth.Info("token issued",
		slog.String("user", "alice@vivy.local"),
		slog.String("token_type", "Bearer"),
		slog.Duration("ttl", 15*time.Minute),
		slog.String("jti", "tok_c0ffee"),
	)

	auth.Warn("permission denied",
		slog.String("user", "bob@vivy.local"),
		slog.String("action", "plugin.install"),
		slog.String("resource", "plugins/telegram-bridge"),
		slog.String("required_role", "admin"),
	)

	auth.Debug("session refreshed",
		slog.String("session_id", "sess_77aa"),
		slog.Int("refresh_count", 2),
	)
}

// SimulateErrors shows error chains, panics-as-errors, and recovery.
func (s *Server) SimulateErrors() {
	// Nested error chain
	base := errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")
	wrapped := fmt.Errorf("connect postgres primary: %w", base)
	s.log.Error("database unavailable",
		slog.Any("err", wrapped),
		slog.String("component", "storage.postgres"),
		slog.Int("retry_in_ms", 2000),
	)

	// Recovered panic
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("recovered from panic",
					slog.Any("panic", rec),
					slog.String("goroutine", "http-handler"),
					slog.Bool("service_still_alive", true),
				)
			}
		}()
		panic("demo: nil map write in plugin loader")
	}()

	// Structured domain error
	s.log.Error("run aborted",
		slog.String("run_id", "run_9f3a"),
		slog.String("reason", "policy_denied"),
		slog.Group("policy",
			slog.String("id", "net.egress"),
			slog.String("decision", "deny"),
			slog.String("detail", "outbound network disabled in demo mode"),
		),
	)
}

// SimulateParallel writes concurrent logs to prove the mutex works.
func (s *Server) SimulateParallel(ctx context.Context) {
	var wg sync.WaitGroup
	workers := 4
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			log := s.log.With(
				slog.Int("worker", id),
				slog.String("pool", "tool-exec"),
			)
			for i := 0; i < 3; i++ {
				took := time.Duration(20+rand.Intn(80)) * time.Millisecond
				log.Info("tool executed",
					slog.String("tool", []string{"read_file", "search_files", "write_file"}[i%3]),
					slog.Duration("took", took),
					slog.Bool("ok", true),
				)
				time.Sleep(15 * time.Millisecond)
			}
		}(w)
	}
	wg.Wait()
	s.log.Info("parallel batch finished",
		slog.Int("workers", workers),
		slog.Int("tasks", workers*3),
	)
}

// Shutdown writes a clean exit narrative.
func (s *Server) Shutdown() {
	s.log.Warn("shutdown signal received",
		slog.String("signal", "SIGINT"),
		slog.String("source", "user"),
	)
	s.log.Info("draining in-flight requests",
		slog.Int("inflight", 2),
		slog.Duration("grace", 5*time.Second),
	)
	s.log.Info("closing listeners",
		slog.String("addr", fmt.Sprintf("127.0.0.1:%d", s.port)),
	)
	s.log.Info("service stopped",
		slog.Duration("uptime", 3*time.Minute+42*time.Second),
		slog.Int("exit_code", 0),
	)
}
