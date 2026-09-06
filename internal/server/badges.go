package server

// streakBadge returns a milestone emoji for a current streak length, or ""
// below the first tier. Tiers are checked highest-first so a 400-day streak
// gets the trophy, not the sparkle.
func streakBadge(streak int) string {
	tiers := []struct {
		days  int
		emoji string
	}{
		{365, "🏆"},
		{100, "💯"},
		{30, "🔥"},
		{7, "✨"},
	}
	for _, t := range tiers {
		if streak >= t.days {
			return t.emoji
		}
	}
	return ""
}
