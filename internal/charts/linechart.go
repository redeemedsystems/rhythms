package charts

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"rhythms/internal/domain"
)

const (
	valueLineWidth   = 600
	valueLineHeight  = 140
	valueLinePadding = 10
)

const (
	scoreLineWidth   = 600
	scoreLineHeight  = 140
	scoreLinePadding = 10
)

// ScoreLineChart renders a score history (0.0-1.0) as a simple SVG
// polyline, ascending oldest-to-newest left to right. Used by the
// aggregate dashboard to plot the average score across all habits over
// time — a single habit's own score is already shown as a stat tile, so
// this chart earns its place only at the cross-habit level.
func ScoreLineChart(points []domain.ScorePoint, colorHex string) template.HTML {
	if len(points) < 2 {
		return emptyChart("chart chart-line", scoreLineWidth, scoreLineHeight, "Not enough data yet")
	}

	n := len(points)
	innerW := float64(scoreLineWidth - 2*scoreLinePadding)
	innerH := float64(scoreLineHeight - 2*scoreLinePadding)

	var path strings.Builder
	for i, p := range points {
		x := float64(scoreLinePadding) + float64(i)/float64(n-1)*innerW
		y := float64(scoreLinePadding) + (1-clamp01(p.Value))*innerH
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		fmt.Fprintf(&path, "%s%.1f,%.1f ", cmd, x, y)
	}

	inner := fmt.Sprintf(
		`<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="currentColor" stroke-opacity="0.15"/>`+
			`<path d="%s" fill="none" stroke="%s" stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>`,
		scoreLinePadding, float64(scoreLinePadding)+innerH/2, scoreLineWidth-scoreLinePadding, float64(scoreLinePadding)+innerH/2,
		path.String(), colorHex,
	)
	return svg(scoreLineWidth, scoreLineHeight, "chart chart-line", inner)
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

// NumericValueLineChart renders a numeric habit's actual recorded values as
// an SVG polyline, oldest-to-newest left to right, scaled to the observed
// min/max in entries — unlike a 0-1 score, raw values (reps, minutes,
// dollars...) have no fixed range to assume.
//
// entries must already be filtered to the desired display window and
// contain only real recorded rows, never auto-filled/placeholder days (e.g.
// from domain.DenseRange) — a placeholder's zero NumericValue would be
// visually indistinguishable from a genuinely recorded 0.
func NumericValueLineChart(entries []domain.Entry, colorHex string) template.HTML {
	if len(entries) < 2 {
		return emptyChart("chart chart-line", valueLineWidth, valueLineHeight, "Not enough data yet")
	}

	minV, maxV := entries[0].NumericValue, entries[0].NumericValue
	for _, e := range entries[1:] {
		if e.NumericValue < minV {
			minV = e.NumericValue
		}
		if e.NumericValue > maxV {
			maxV = e.NumericValue
		}
	}
	span := maxV - minV
	if span == 0 {
		// A flat series still needs a nonzero divisor below; padding the
		// span keeps the single observed value centered rather than
		// dividing by zero.
		span = 1
	}

	n := len(entries)
	innerW := float64(valueLineWidth - 2*valueLinePadding)
	innerH := float64(valueLineHeight - 2*valueLinePadding)

	var path strings.Builder
	for i, e := range entries {
		x := float64(valueLinePadding) + float64(i)/float64(n-1)*innerW
		y := float64(valueLinePadding) + (1-(e.NumericValue-minV)/span)*innerH
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		fmt.Fprintf(&path, "%s%.1f,%.1f ", cmd, x, y)
	}

	// Min/max labels: formatFloat's output is digits/./-/e only, safe to
	// embed raw like every other number this package formats (see package
	// doc comment on why raw string-building is safe here).
	inner := fmt.Sprintf(
		`<path d="%s" fill="none" stroke="%s" stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>`+
			`<text x="%d" y="12" font-size="11" fill="currentColor" opacity="0.7">%s</text>`+
			`<text x="%d" y="%d" font-size="11" fill="currentColor" opacity="0.7">%s</text>`,
		path.String(), colorHex,
		valueLinePadding, formatValue(maxV),
		valueLinePadding, valueLineHeight-4, formatValue(minV),
	)
	return svg(valueLineWidth, valueLineHeight, "chart chart-line", inner)
}

func formatValue(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
