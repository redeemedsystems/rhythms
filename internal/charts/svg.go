// Package charts renders small inline SVG visualizations (a completion
// strip / heatmap) for the dashboard, avoiding any client-side JS charting
// library.
package charts

import (
	"fmt"
	"html/template"
	"strings"

	"rhythms/internal/store"
)

const (
	cellSize = 14
	cellGap  = 3
)

// CompletionStrip renders one square per date in order, colored by whether
// the habit was completed that day. Intended for date ranges from a single
// day up to a month; wraps onto multiple rows of 7 for anything longer than
// a week so it reads like a small calendar heatmap.
func CompletionStrip(habit *store.Habit, dates []string, completion map[string]int) template.HTML {
	const perRow = 7

	rows := (len(dates) + perRow - 1) / perRow
	width := perRow*(cellSize+cellGap) - cellGap
	height := rows*(cellSize+cellGap) - cellGap
	if height < cellSize {
		height = cellSize
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg viewBox="0 0 %d %d" width="100%%" role="img" aria-label="%s completion history">`,
		width, height, template.HTMLEscapeString(habit.Name))

	for i, d := range dates {
		row := i / perRow
		col := i % perRow
		x := col * (cellSize + cellGap)
		y := row * (cellSize + cellGap)

		complete := store.IsComplete(habit, completion[d])
		color := "var(--border)"
		if complete {
			color = "var(--accent)"
		}

		fmt.Fprintf(&sb,
			`<rect x="%d" y="%d" width="%d" height="%d" rx="3" fill="%s"><title>%s</title></rect>`,
			x, y, cellSize, cellSize, color, template.HTMLEscapeString(d),
		)
	}

	sb.WriteString(`</svg>`)
	return template.HTML(sb.String())
}
