package clilog

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"testing/slogtest"
)

type handlerSpy struct {
	slog.Handler
	timed *bool
}

func (h *handlerSpy) Handle(ctx context.Context, r slog.Record) error {
	*h.timed = !r.Time.IsZero()
	return h.Handler.Handle(ctx, r)
}

func (h *handlerSpy) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handlerSpy{Handler: h.Handler.WithAttrs(attrs), timed: h.timed}
}

func (h *handlerSpy) WithGroup(name string) slog.Handler {
	return &handlerSpy{Handler: h.Handler.WithGroup(name), timed: h.timed}
}

func TestLogHandler(t *testing.T) {
	var buf bytes.Buffer
	var timed bool
	slogtest.Run(t,
		func(*testing.T) slog.Handler {
			buf.Reset()
			timed = false
			return &handlerSpy{Handler: NewHandler(&buf, slog.LevelDebug), timed: &timed}
		},
		func(*testing.T) map[string]any {
			return parse(t, buf.Bytes(), timed)
		})
}

func parse(t *testing.T, lines []byte, timed bool) map[string]any {
	t.Helper()
	m := make(map[string]any)
	first, rest, _ := bytes.Cut(lines, []byte("\n"))
	var level slog.Level
	var message string
	if after, found := bytes.CutPrefix(first, []byte("DEBUG: ")); found {
		message = string(after)
		level = slog.LevelDebug
	} else if after, found := bytes.CutPrefix(first, []byte("WARN: ")); found {
		message = string(after)
		level = slog.LevelWarn
	} else if after, found := bytes.CutPrefix(first, []byte("ERROR: ")); found {
		message = string(after)
		level = slog.LevelError
	} else {
		message = string(first)
		level = slog.LevelInfo
	}
	m[slog.MessageKey] = message
	m[slog.LevelKey] = level
	for line := range bytes.Lines(rest) {
		line = bytes.TrimPrefix(bytes.TrimSuffix(line, []byte("\n")), []byte("\t"))
		if before, after, found := bytes.Cut(line, []byte(" = ")); found {
			ks := bytes.Split(before, []byte("."))
			g := m
			for _, gkb := range ks[:len(ks)-1] {
				gk := string(gkb)
				if g2, ok := g[gk].(map[string]any); ok {
					g = g2
				} else {
					g2 = make(map[string]any)
					g[gk] = g2
					g = g2
				}
			}
			k := string(ks[len(ks)-1])
			g[k] = string(after)
		}
	}
	if timed {
		m[slog.TimeKey] = nil
	}
	return m
}
