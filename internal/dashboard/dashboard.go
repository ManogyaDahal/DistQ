package dashboard

import "embed"

// StaticFS embeds all files from the static subdirectory.
//
//go:embed static/*
var StaticFS embed.FS
