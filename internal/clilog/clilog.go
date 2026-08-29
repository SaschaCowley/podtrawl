package clilog

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

type Handler struct {
	level        slog.Leveler
	out          io.Writer
	mu           *sync.Mutex
	group        []byte
	preformatted []byte
}

func NewHandler(out io.Writer, level slog.Leveler) *Handler {
	h := &Handler{out: out, level: level, mu: &sync.Mutex{}}
	if h.level == nil {
		h.level = slog.LevelInfo
	}
	return h
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	if r.Level != slog.LevelInfo {
		buf = append(buf, r.Level.String()...)
		buf = append(buf, ": "...)
	}
	buf = append(buf, r.Message...)
	buf = append(buf, '\n')
	buf = append(buf, h.preformatted...)
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, h.group, a)
		return true
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *Handler) appendAttr(buf, group []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}
	if kind := a.Value.Kind(); kind == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}
		if a.Key != "" {
			group = h.appendGroup(h.group, a.Key)
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, group, ga)
		}
	} else {
		buf = append(buf, '\t')
		buf = append(buf, group...)
		buf = append(buf, a.Key...)
		buf = append(buf, " = "...)
		buf = append(buf, a.Value.String()...)
		buf = append(buf, '\n')
	}
	return buf
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	buf := make([]byte, len(h.preformatted), len(h.preformatted)+1024)
	copy(buf, h.preformatted)
	for _, a := range attrs {
		buf = h2.appendAttr(buf, h2.group, a)
	}
	h2.preformatted = append([]byte(nil), buf...)
	return &h2
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.group = h2.appendGroup(h2.group, name)
	return &h2
}

func (h *Handler) appendGroup(existing []byte, new string) []byte {
	if new == "" {
		return existing
	}
	group := make([]byte, 0, len(existing)+len(new)+1)
	group = append(group, existing...)
	group = append(group, new...)
	group = append(group, '.')
	return group
}
