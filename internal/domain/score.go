package domain

import "math"

const scoreHalfLifeConstant = 13.0

// ScorePoint is a habit's EMA "score" (0.0-1.0) on one day.
type ScorePoint struct {
	Date  Date
	Value float64
}

// scoreCompute is uHabits' Score.compute, verbatim: an exponential moving
// average whose decay rate depends on the habit's frequency — a habit
// required more often has a shorter half-life (reacts faster to misses).
func scoreCompute(freq, previousScore, checkmarkValue float64) float64 {
	multiplier := math.Pow(0.5, math.Sqrt(freq)/scoreHalfLifeConstant)
	return previousScore*multiplier + checkmarkValue*(1-multiplier)
}

// ScoreSeries computes the running score for every day in a dense computed-
// entry timeline (see DenseRange), ascending oldest-first. Ported from
// uHabits' ScoreList.recompute.
//
// checkmarkValue for a day is the fraction of a rolling window (size =
// Denominator days, ending on that day) that's been completed — a genuine
// sliding-window sum maintained incrementally, not recomputed per day. A
// SKIP day leaves the score unchanged from the previous day rather than
// contributing a neutral value.
func ScoreSeries(h Habit, dense []Entry) []ScorePoint {
	// See the matching guard in buildIntervals: a zero-value Frequency must
	// not reach the rolling-window arithmetic below (a zero denominator
	// would make every day "subtract itself", silently corrupting scores).
	numerator := max(1, h.Freq.Numerator)
	denominator := max(1, h.Freq.Denominator)
	freq := float64(numerator) / float64(denominator) // the EMA multiplier always uses the un-doubled ratio
	isNumerical := h.Type == Numerical
	isAtMost := h.TargetType == AtMost

	// Non-daily boolean habits double numerator+denominator to smooth out
	// irregular repetition schedules (e.g. a weekly habit done on different
	// days each week) — affects only the rolling window and normalizer
	// below, never the EMA multiplier itself.
	if !isNumerical && freq < 1.0 {
		numerator *= 2
		denominator *= 2
	}

	previous := 0.0
	if isNumerical && isAtMost {
		previous = 1.0
	}

	rollingSum := 0.0
	result := make([]ScorePoint, len(dense))
	for i, e := range dense {
		rollingSum += scoreContribution(isNumerical, e)
		if i-denominator >= 0 {
			rollingSum -= scoreContribution(isNumerical, dense[i-denominator])
		}

		if e.Value == Skip {
			result[i] = ScorePoint{Date: e.Date, Value: previous}
			continue
		}

		var checkmarkValue float64
		if isNumerical {
			checkmarkValue = numericCheckmarkValue(rollingSum, h.TargetValue, isAtMost)
		} else {
			checkmarkValue = math.Min(1.0, rollingSum/float64(numerator))
		}

		previous = scoreCompute(freq, previous, checkmarkValue)
		result[i] = ScorePoint{Date: e.Date, Value: previous}
	}
	return result
}

func scoreContribution(isNumerical bool, e Entry) float64 {
	if isNumerical {
		return math.Max(0, e.NumericValue)
	}
	if e.Value == YesManual {
		return 1.0
	}
	return 0.0
}

func numericCheckmarkValue(rollingSum, targetValue float64, isAtMost bool) float64 {
	if !isAtMost {
		if targetValue > 0 {
			return math.Min(1.0, rollingSum/targetValue)
		}
		return 1.0
	}
	if targetValue > 0 {
		v := 1 - (rollingSum-targetValue)/targetValue
		return math.Min(1.0, math.Max(0.0, v))
	}
	if rollingSum > 0 {
		return 0.0
	}
	return 1.0
}
