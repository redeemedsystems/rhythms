package domain

// ComputeEntries derives the "computed" entry timeline from a habit's full
// raw history: for numeric habits it's just a copy (no auto-fill concept
// applies); for boolean habits it fills in YES_AUTO on days the habit's
// frequency didn't strictly require a check-in, so e.g. a weekly habit
// checked only on Sundays doesn't look "broken" the other six days.
//
// original must be the habit's ENTIRE known history, not a display-bounded
// window — the interval construction below looks arbitrarily far back
// through manual check-ins, so truncating the input silently produces wrong
// auto-fill near the truncation boundary. Callers needing a narrow window
// (a 7-day strip, a chart range) should call this once over full history and
// then slice the result, never the other way around.
//
// This is a line-for-line port of uHabits' EntryList.recomputeFrom /
// buildIntervals / snapIntervalsTogether / buildEntriesFromInterval
// (uhabits-core/.../models/EntryList.kt), fetched and read directly rather
// than worked from a paraphrase, because getting the interval anchoring
// exactly right was flagged as the single highest bug-risk part of this
// rewrite.
func ComputeEntries(h Habit, original []Entry) []Entry {
	if h.Type == Numerical {
		out := make([]Entry, len(original))
		copy(out, original)
		return out
	}

	manualDesc := manualDatesDescending(original)
	intervals := buildIntervals(h.Freq, manualDesc)
	snapIntervalsTogether(intervals)
	dense := buildEntriesFromInterval(original, intervals)

	// Matches recomputeFrom's final filter: only entries with a real value
	// (or notes) are kept — bare Unknown placeholders are dropped.
	out := make([]Entry, 0, len(dense))
	for _, e := range dense {
		if e.Value != Unknown || e.Notes != "" {
			out = append(out, e)
		}
	}
	return out
}

func manualDatesDescending(original []Entry) []Date {
	var dates []Date
	for _, e := range original {
		if e.Value == YesManual {
			dates = append(dates, e.Date)
		}
	}
	// Insertion sort descending (newest first) — habit histories are small
	// (single-user, at most a few thousand days), so O(n^2) is irrelevant.
	for i := 1; i < len(dates); i++ {
		for j := i; j > 0 && dates[j].After(dates[j-1]); j-- {
			dates[j], dates[j-1] = dates[j-1], dates[j]
		}
	}
	return dates
}

// interval is a contiguous span of days over which a non-daily habit's
// quota (Numerator check-ins within Denominator days) has been met, so every
// day in [Begin, End] counts toward the streak even if not manually checked.
// Center is the newest of the Numerator manual check-ins that satisfied the
// quota — it always falls within [Begin, End] and never moves once created.
type interval struct {
	Begin, Center, End Date
}

// buildIntervals scans every window of Numerator consecutive manual
// check-ins (newest-first, one candidate per check-in) and, wherever that
// window fits inside Denominator days, emits an interval covering it.
// Adjacent windows overlap heavily by construction; snapIntervalsTogether
// resolves that afterward. manualDesc must be sorted newest-first.
func buildIntervals(freq Frequency, manualDesc []Date) []interval {
	// A zero-value (or otherwise invalid) Frequency isn't a state any UI
	// path should produce, but Habit carries no constructor-enforced
	// invariant — a malformed value here must not corrupt the loop bound
	// below (num-1 going negative would index manualDesc out of range).
	num := max(1, freq.Numerator)
	den := max(1, freq.Denominator)

	var intervals []interval
	for i := num - 1; i < len(manualDesc); i++ {
		begin := manualDesc[i]        // older end of this num-sized cluster
		center := manualDesc[i-num+1] // newer end of this num-sized cluster

		size := den
		if den == 30 || den == 31 { // "monthly" frequency: use begin's actual calendar month length
			if begin.IsLastDayOfMonth() {
				size = begin.AddDays(1).DaysInMonth()
			} else {
				size = begin.DaysInMonth()
			}
		}

		if begin.DaysUntil(center) < size {
			intervals = append(intervals, interval{
				Begin:  begin,
				Center: center,
				End:    begin.AddDays(size - 1),
			})
		}
	}
	return intervals
}

// snapIntervalsTogether walks the newest-first interval list and, wherever
// two neighboring intervals overlap in time, slides the older one backward
// (never past its own Center) just far enough to abut the newer one instead
// of overlapping it. Intervals with a genuine calendar gap between them are
// left untouched — those days are legitimately outside any quota window.
func snapIntervalsTogether(intervals []interval) {
	for i := 1; i < len(intervals); i++ {
		curr := intervals[i]
		next := intervals[i-1] // the newer neighbor, already finalized

		gapNextToCurrent := next.Begin.DaysUntil(curr.End) // curr.End - next.Begin; >=0 means overlap
		gapCenterToEnd := curr.Center.DaysUntil(curr.End)  // how far curr.End can shrink before passing Center

		if gapNextToCurrent >= 0 {
			shift := min(gapCenterToEnd, gapNextToCurrent+1)
			intervals[i] = interval{
				Begin:  curr.Begin.AddDays(-shift),
				Center: curr.Center,
				End:    curr.End.AddDays(-shift),
			}
		}
	}
}

// buildEntriesFromInterval produces a dense, newest-first day-by-day entry
// list spanning every date touched by original or by any interval. Days
// inside an interval default to YES_AUTO; everything else defaults to
// Unknown. Original entries are then overlaid: SKIP and YES_MANUAL always
// win as recorded, but any other original value (No, or a stray YES_AUTO)
// occurring inside an interval is forced to YES_AUTO — the interval's
// coverage takes precedence over an explicit "not done" within a window
// whose quota was still met some other day.
func buildEntriesFromInterval(original []Entry, intervals []interval) []Entry {
	if len(original) == 0 {
		return nil
	}

	from, to := original[0].Date, original[0].Date
	for _, e := range original {
		if e.Date.Before(from) {
			from = e.Date
		}
		if e.Date.After(to) {
			to = e.Date
		}
	}
	for _, iv := range intervals {
		if iv.Begin.Before(from) {
			from = iv.Begin
		}
		if iv.End.After(to) {
			to = iv.End
		}
	}

	n := from.DaysUntil(to) + 1
	result := make([]Entry, n) // index 0 = to (newest) ... index n-1 = from (oldest)
	for i := range result {
		result[i] = Entry{Date: to.AddDays(-i), Value: Unknown}
	}
	offsetOf := func(d Date) int { return d.DaysUntil(to) }

	for _, iv := range intervals {
		for d := iv.End; !d.Before(iv.Begin); d = d.AddDays(-1) {
			result[offsetOf(d)] = Entry{Date: d, Value: YesAuto}
		}
	}

	for _, e := range original {
		offset := offsetOf(e.Date)
		value := e.Value
		if result[offset].Value != Unknown && e.Value != Skip && e.Value != YesManual {
			value = YesAuto
		}
		result[offset] = Entry{Date: e.Date, Value: value, NumericValue: e.NumericValue, Notes: e.Notes}
	}

	return result
}
