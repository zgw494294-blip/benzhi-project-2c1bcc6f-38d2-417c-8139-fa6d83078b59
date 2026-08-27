// Package web exposes the browser workbench as assets embedded in the Go binary.
package web

import "embed"

// IndexHTML is the workbench document served at the root route.
//
//go:embed index.html
var IndexHTML []byte

// Files contains the JavaScript and stylesheet used by IndexHTML.
//
//go:embed static/*
var Files embed.FS
