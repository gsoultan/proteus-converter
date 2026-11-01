// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package core

import (
	"errors"
	"fmt"
	"runtime/debug"
)

var (
	// ErrUnknownFormat is returned when no converter is registered for a format or extension.
	ErrUnknownFormat = errors.New("proteus: unknown format")

	// ErrResourceDenied is returned when a resource URL is outside what the loader allows
	// (for example an http URL while network access is disabled).
	ErrResourceDenied = errors.New("proteus: resource access denied")

	// ErrResourceTooLarge is returned when a resource exceeds Config.MaxResourceSize.
	ErrResourceTooLarge = errors.New("proteus: resource too large")
)

// ErrTooManyPages is returned when a document would need more pages than
// Config.MaxPages allows (see WithMaxPages).
var ErrTooManyPages = errors.New("proteus: too many pages")

// ErrInternal marks a bug inside proteus: a conversion that panicked. The
// panic is turned into an *InternalError wrapping it, so a malformed document
// can never crash the calling program.
var ErrInternal = errors.New("proteus: internal error")

// InternalError carries the panic value and stack of a failed conversion.
type InternalError struct {
	Value any
	Stack []byte
}

func (e *InternalError) Error() string { return fmt.Sprintf("%v: %v", ErrInternal, e.Value) }

// Unwrap lets errors.Is(err, ErrInternal) match.
func (e *InternalError) Unwrap() error { return ErrInternal }

// Recover turns a panic in the calling function into an *InternalError stored
// in *err. Use it as `defer core.Recover(&err)` at conversion entry points.
func Recover(err *error) {
	if r := recover(); r != nil {
		*err = &InternalError{Value: r, Stack: debug.Stack()}
	}
}
