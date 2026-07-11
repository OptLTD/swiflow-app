// Package web embeds the built Vue UI (web/dist). The Vite build outputs to
// web/dist; this file exposes it as an embed.FS for the server to serve.
package web

import "embed"

// Dist holds the embedded built frontend.
//
//go:embed all:dist
var Dist embed.FS
