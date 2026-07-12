// Package webui embeds the built Vue UI (webui/dist). The Vite build outputs to
// webui/dist; this file exposes it as an embed.FS for the server to serve.
package webui

import "embed"

// Dist holds the embedded built frontend.
//
//go:embed all:dist
var Dist embed.FS
