package charts

import (
	"fmt"
	"strings"
	"time"

	"html/template"

	"rhythms/internal/domain"
)

const (
	heatmapCell    = 11
	heatmapGap     = 2
	heatmapStep    = heatmapCell + heatmapGap
	heatmapMarginX = 15
	heatmapMarginY = 15

	// HeatmapDefaultWeeks is the recommended window for CalendarHeatmap — 26
	// weeks (~6 months) keeps the SVG a reasonable size while still reading
	// as a real contribution-style history.
	HeatmapDefaultWeeks = 26
)

// CalendarHeatmap renders a GitHub-contribution-style grid: one column per
// week, one row per weekday, colored when that day is completed. dense must
// be ascending (oldest first) with no gaps, as produced by domain.DenseRange
// — its last element is treated as "today" (the grid's right edge).
func CalendarHeatmap(h domain.Habit, dense []domain.Entry, weeks int, colorHex string) template.HTML {
	width := weeks*heatmapStep + 2*heatmapMarginX
	height := 7*heatmapStep + 2*heatmapMarginY

	if len(dense) == 0 {
		return emptyChart("chart chart-heatmap", width, height, "No data yet")
	}

	byDate := make(map[string]domain.Entry, len(dense))
	for _, e := range dense {
		byDate[e.Date.String()] = e
	}

	to := dense[len(dense)-1].Date
	from := to.AddDays(-(weeks*7 - 1))
	for from.Weekday() != time.Sunday {
		from = from.AddDays(-1)
	}

	var cells strings.Builder
	col := 0
	for d := from; !d.After(to); d = d.AddDays(1) {
		row := int(d.Weekday())
		x := heatmapMarginX + col*heatmapStep
		y := heatmapMarginY + row*heatmapStep

		fill := "var(--border)"
		if e, ok := byDate[d.String()]; ok && domain.IsCompleted(h, e) {
			fill = colorHex
		}

		fmt.Fprintf(&cells,
			`<rect x="%d" y="%d" width="%d" height="%d" rx="2" fill="%s"><title>%s</title></rect>`,
			x, y, heatmapCell, heatmapCell, fill, d.String(),
		)

		if row == 6 {
			col++
		}
	}

	return svg(width, height, "chart chart-heatmap", cells.String())
}
