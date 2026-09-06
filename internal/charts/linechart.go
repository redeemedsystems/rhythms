package charts

import (
	"fmt"
	"html/template"
	"strings"

	"rhythms/internal/domain"
)

const (
	lineWidth   = 600
	lineHeight  = 140
	linePadding = 10
)

// ScoreLineChart renders a habit's score history (0.0-1.0) as a simple SVG
// polyline, ascending oldest-to-newest left to right.
func ScoreLineChart(points []domain.ScorePoint, colorHex string) template.HTML {
	if len(points) < 2 {
		return emptyChart("chart chart-line", lineWidth, lineHeight, "Not enough data yet")
	}

	n := len(points)
	innerW := float64(lineWidth - 2*linePadding)
	innerH := float64(lineHeight - 2*linePadding)

	var path strings.Builder
	for i, p := range points {
		x := float64(linePadding) + float64(i)/float64(n-1)*innerW
		y := float64(linePadding) + (1-clamp01(p.Value))*innerH
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		fmt.Fprintf(&path, "%s%.1f,%.1f ", cmd, x, y)
	}

	inner := fmt.Sprintf(
		`<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="currentColor" stroke-opacity="0.15"/>`+
			`<path d="%s" fill="none" stroke="%s" stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>`,
		linePadding, float64(linePadding)+innerH/2, lineWidth-linePadding, float64(linePadding)+innerH/2,
		path.String(), colorHex,
	)
	return svg(lineWidth, lineHeight, "chart chart-line", inner)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
