package charts

import (
	"fmt"
	"sort"
	"strings"

	"html/template"

	"rhythms/internal/domain"
)

const (
	streakChartWidth = 600
	streakRowHeight  = 22
	streakLabelX     = 90
	streakBarStartX  = 96
	maxStreaksShown  = 10
)

// StreakBarChart shows the longest streaks as horizontal bars, most recent
// first — mirrors uHabits' "best streaks" list (longest N, re-sorted by
// recency for display).
func StreakBarChart(streaks []domain.Streak, colorHex string) template.HTML {
	if len(streaks) == 0 {
		return emptyChart("chart chart-bars", streakChartWidth, 40, "No streaks yet")
	}

	sorted := append([]domain.Streak(nil), streaks...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Length > sorted[j].Length })
	if len(sorted) > maxStreaksShown {
		sorted = sorted[:maxStreaksShown]
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].End.After(sorted[j].End) })

	maxLen := 1
	for _, s := range sorted {
		if s.Length > maxLen {
			maxLen = s.Length
		}
	}

	height := len(sorted)*streakRowHeight + 10
	maxBarWidth := float64(streakChartWidth - streakBarStartX - 40)

	var bars strings.Builder
	for i, s := range sorted {
		barWidth := float64(s.Length) / float64(maxLen) * maxBarWidth
		if barWidth < 2 {
			barWidth = 2
		}
		y := i*streakRowHeight + 4
		fmt.Fprintf(&bars, `<text x="%d" y="%d" font-size="11" text-anchor="end" fill="currentColor">%s</text>`,
			streakLabelX, y+11, s.Start.String())
		fmt.Fprintf(&bars, `<rect x="%d" y="%d" width="%.1f" height="14" rx="3" fill="%s"/>`,
			streakBarStartX, y, barWidth, colorHex)
		fmt.Fprintf(&bars, `<text x="%.1f" y="%d" font-size="11" fill="currentColor">%d</text>`,
			float64(streakBarStartX)+barWidth+6, y+11, s.Length)
	}

	return svg(streakChartWidth, height, "chart chart-bars", bars.String())
}

var weekdayLabels = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

const (
	weekdayChartWidth  = 350
	weekdayChartHeight = 120
	weekdayBarWidth    = 36
	weekdayGap         = 12
	weekdayBarMaxH     = 80
	weekdayBaselineY   = 100
)

// WeekdayBarChart shows total completions grouped by day of week, across a
// habit's full history — which days of the week it tends to get done.
// counts is indexed by time.Weekday (0=Sunday .. 6=Saturday).
func WeekdayBarChart(counts [7]int, colorHex string) template.HTML {
	max := 1
	for _, c := range counts {
		if c > max {
			max = c
		}
	}

	var bars strings.Builder
	for i, c := range counts {
		barHeight := float64(c) / float64(max) * weekdayBarMaxH
		x := 10 + i*(weekdayBarWidth+weekdayGap)
		y := float64(weekdayBaselineY) - barHeight
		if c > 0 {
			fmt.Fprintf(&bars, `<rect x="%d" y="%.1f" width="%d" height="%.1f" rx="2" fill="%s"/>`,
				x, y, weekdayBarWidth, barHeight, colorHex)
			fmt.Fprintf(&bars, `<text x="%d" y="%.1f" font-size="10" text-anchor="middle" fill="currentColor">%d</text>`,
				x+weekdayBarWidth/2, y-4, c)
		}
		fmt.Fprintf(&bars, `<text x="%d" y="%d" font-size="11" text-anchor="middle" fill="currentColor" opacity="0.7">%s</text>`,
			x+weekdayBarWidth/2, weekdayBaselineY+15, weekdayLabels[i])
	}

	return svg(weekdayChartWidth, weekdayChartHeight, "chart chart-weekday", bars.String())
}
