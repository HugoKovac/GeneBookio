package errorwrapper

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

type IErrorWrapper interface {
	FormatTrace() error
}

type ErrorWrapper struct {
	error
	trace string
}

func (te *ErrorWrapper) FormatTrace() error {
	return fmt.Errorf("%v - %w", te.trace, te.error)
}

func Wrap(v any) error {
	var trace string
	if v == nil {
		return nil
	}
	// _, file, line, ok := runtime.Caller(1)
	pcs := make([]uintptr, 10)
	n := runtime.Callers(2, pcs)

	frames := runtime.CallersFrames(pcs[:n])
	first := true
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.Function, "hkorpo/book") {
			break
		}
		if first {
			trace = fmt.Sprintf("%s:%d", frame.File, frame.Line)
		} else {
			trace = fmt.Sprintf("%s <- %s:%d", trace, frame.File, frame.Line)
		}
		if !more {
			break
		}
		first = false
	}
	var err error
	switch e := v.(type) {
	case string:
		err = errors.New(e)
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}
	return &ErrorWrapper{error: err, trace: trace}
}

func (te *ErrorWrapper) Unwrap() error {
	return te.error
}

// StatusError attaches an explicit HTTP status to an error so the shared
// access-log error handler (pkg/fiber/logger) can respond with it, instead
// of falling back to a generic 500 — see WithStatus.
type StatusError struct {
	error
	status int
}

// WithStatus tags err with the HTTP status the access-log error handler
// should respond with. Wrap the result (or wrap this around a Wrap call) so
// the trace is still captured — Unwrap() lets StatusOf find the status
// regardless of which one is outermost.
func WithStatus(status int, err error) error {
	if err == nil {
		return nil
	}
	return &StatusError{error: err, status: status}
}

func (se *StatusError) Unwrap() error {
	return se.error
}
