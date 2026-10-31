package core

import "errors"

var (
	// ErrUnknownFormat is returned when no converter is registered for a format or extension.
	ErrUnknownFormat = errors.New("proteus: unknown format")

	// ErrResourceDenied is returned when a resource URL is outside what the loader allows
	// (for example an http URL while network access is disabled).
	ErrResourceDenied = errors.New("proteus: resource access denied")

	// ErrResourceTooLarge is returned when a resource exceeds Config.MaxResourceSize.
	ErrResourceTooLarge = errors.New("proteus: resource too large")
)
