// Package logger builds the logger the cli reports through.
package logger

import (
	"context"
	"io"
	"log/slog"
	"time"
)

// New returns a logger writing to w, which is stderr for the cli: what a
// command prints for a script to read goes to stdout, and everything about
// the run goes here, so redirecting one doesn't collect the other.
func New(w io.Writer) *slog.Logger {
	return slog.New(untimed{slog.NewTextHandler(w, nil)})
}

// untimed is a handler which drops the time a record was made. A command
// prints as it runs, in front of the person who started it, and the time
// each line was written says less than the line does.
//
// It's a handler rather than a ReplaceAttr because a replacement can't tell
// the record's own timestamp from an attribute which shares its name, and a
// command reporting a build time passes one of those.
type untimed struct {
	slog.Handler
}

// Handle writes the record without the time it was made. A handler leaves
// out the timestamp of a record which doesn't carry one.
func (h untimed) Handle(ctx context.Context, record slog.Record) error {
	record.Time = time.Time{}
	return h.Handler.Handle(ctx, record)
}

// WithAttrs keeps the wrapping, which the embedded handler would drop.
func (h untimed) WithAttrs(attrs []slog.Attr) slog.Handler {
	return untimed{h.Handler.WithAttrs(attrs)}
}

// WithGroup keeps the wrapping, which the embedded handler would drop.
func (h untimed) WithGroup(name string) slog.Handler {
	return untimed{h.Handler.WithGroup(name)}
}
