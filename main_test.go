package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

type fakeCloser struct {
	err    error
	closed int
}

func (f *fakeCloser) Close() error {
	f.closed++
	return f.err
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	origOut := Log.Logger.Out
	origFmt := Log.Logger.Formatter
	Log.Logger.Out = &buf
	Log.Logger.Formatter = &logrus.TextFormatter{DisableColors: true, DisableTimestamp: true}
	t.Cleanup(func() {
		Log.Logger.Out = origOut
		Log.Logger.Formatter = origFmt
	})
	return &buf
}

func TestCloseAndLogSuccess(t *testing.T) {
	buf := captureLog(t)
	c := &fakeCloser{}

	closeAndLog(c, "thing")

	if c.closed != 1 {
		t.Fatalf("Close called %d times, want 1", c.closed)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no log output on success, got %q", buf.String())
	}
}

func TestCloseAndLogError(t *testing.T) {
	buf := captureLog(t)
	c := &fakeCloser{err: errors.New("boom")}

	closeAndLog(c, "rows")

	if c.closed != 1 {
		t.Fatalf("Close called %d times, want 1", c.closed)
	}
	out := buf.String()
	if !strings.Contains(out, "level=error") {
		t.Errorf("expected error-level log, got %q", out)
	}
	if !strings.Contains(out, "rows") {
		t.Errorf("expected log to include label %q, got %q", "rows", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("expected log to include underlying error %q, got %q", "boom", out)
	}
}
