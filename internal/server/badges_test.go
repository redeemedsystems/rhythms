package server

import "testing"

func TestStreakBadge(t *testing.T) {
	cases := []struct {
		streak int
		want   string
	}{
		{0, ""},
		{6, ""},
		{7, "✨"},
		{29, "✨"},
		{30, "🔥"},
		{99, "🔥"},
		{100, "💯"},
		{364, "💯"},
		{365, "🏆"},
		{1000, "🏆"},
	}
	for _, c := range cases {
		if got := streakBadge(c.streak); got != c.want {
			t.Errorf("streakBadge(%d) = %q, want %q", c.streak, got, c.want)
		}
	}
}
