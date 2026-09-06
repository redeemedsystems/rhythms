// Package charts renders habit statistics as server-side SVG — no client
// chart library, so it works identically with JS disabled and adds no
// client-side bundle (see the rewrite plan's deviation #4). Every function
// here only ever embeds numeric coordinates, dates in a fixed YYYY-MM-DD
// format, and colors from domain.Palette — never arbitrary user text — so
// building raw SVG strings without html/template's escaping is safe.
package charts

import (
	"fmt"
	"html/template"
)

func svg(width, height int, class, inner string) template.HTML {
	return template.HTML(fmt.Sprintf(
		`<svg class="%s" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg" role="img" preserveAspectRatio="xMinYMid meet">%s</svg>`,
		class, width, height, inner,
	))
}

func emptyChart(class string, width, height int, message string) template.HTML {
	inner := fmt.Sprintf(
		`<text x="10" y="%d" font-size="12" fill="currentColor" opacity="0.6">%s</text>`,
		height/2, message,
	)
	return svg(width, height, class, inner)
}
