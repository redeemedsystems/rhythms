package store

import (
	"database/sql"
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		t.Fatalf("parse date %q: %v", s, err)
	}
	return d
}

func TestIsComplete(t *testing.T) {
	boolHabit := &Habit{Type: HabitTypeBoolean}
	if !IsComplete(boolHabit, 1) {
		t.Error("boolean habit with amount 1 should be complete")
	}
	if IsComplete(boolHabit, 0) {
		t.Error("boolean habit with amount 0 should not be complete")
	}

	countHabit := &Habit{Type: HabitTypeCount, TargetCount: sql.NullInt64{Int64: 3, Valid: true}}
	if IsComplete(countHabit, 2) {
		t.Error("count habit at 2/3 should not be complete")
	}
	if !IsComplete(countHabit, 3) {
		t.Error("count habit at 3/3 should be complete")
	}
	if !IsComplete(countHabit, 4) {
		t.Error("count habit exceeding target should be complete")
	}

	noTarget := &Habit{Type: HabitTypeQuantity}
	if !IsComplete(noTarget, 1) {
		t.Error("quantity habit with no target should be complete on any activity")
	}
}

func TestCurrentStreak(t *testing.T) {
	h := &Habit{Type: HabitTypeBoolean}

	cases := []struct {
		name       string
		completion map[string]int
		today      string
		want       int
	}{
		{
			name:       "no history",
			completion: map[string]int{},
			today:      "2026-09-06",
			want:       0,
		},
		{
			name: "3-day streak ending today",
			completion: map[string]int{
				"2026-09-04": 1, "2026-09-05": 1, "2026-09-06": 1,
			},
			today: "2026-09-06",
			want:  3,
		},
		{
			name: "streak ending yesterday still counts if today not logged",
			completion: map[string]int{
				"2026-09-04": 1, "2026-09-05": 1,
			},
			today: "2026-09-06",
			want:  2,
		},
		{
			name: "gap two days ago breaks the streak",
			completion: map[string]int{
				"2026-09-03": 1, "2026-09-05": 1, "2026-09-06": 1,
			},
			today: "2026-09-06",
			want:  2,
		},
		{
			name: "missed yesterday and today resets to zero",
			completion: map[string]int{
				"2026-09-01": 1, "2026-09-02": 1,
			},
			today: "2026-09-06",
			want:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CurrentStreak(h, tc.completion, mustDate(t, tc.today))
			if got != tc.want {
				t.Errorf("CurrentStreak() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBestStreak(t *testing.T) {
	h := &Habit{Type: HabitTypeBoolean}
	dates := DateRange(mustDate(t, "2026-09-01"), mustDate(t, "2026-09-10"))

	completion := map[string]int{
		"2026-09-01": 1, "2026-09-02": 1,
		"2026-09-04": 1, "2026-09-05": 1, "2026-09-06": 1, "2026-09-07": 1,
		"2026-09-09": 1,
	}

	got := BestStreak(h, completion, dates)
	if got != 4 {
		t.Errorf("BestStreak() = %d, want 4", got)
	}
}

func TestDateRange(t *testing.T) {
	got := DateRange(mustDate(t, "2026-09-01"), mustDate(t, "2026-09-03"))
	want := []string{"2026-09-01", "2026-09-02", "2026-09-03"}
	if len(got) != len(want) {
		t.Fatalf("expected %d dates, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("date[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}
