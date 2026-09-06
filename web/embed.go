// Package web embeds the server's static assets, htmx bundle, and HTML templates.
package web

import "embed"

//go:embed static templates htmx
var FS embed.FS
