// Package web embeds the app's HTML templates and static assets into the
// binary so it ships as a single self-contained executable.
package web

import "embed"

//go:embed templates
var TemplatesFS embed.FS

//go:embed static
var StaticFS embed.FS
