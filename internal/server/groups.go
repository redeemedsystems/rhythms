package server

import "rhythms/internal/store"

// HabitGroup is a named bucket of habits sharing a category, in display
// order.
type HabitGroup struct {
	Name   string
	Habits []*store.Habit
}

// groupHabits buckets habits by their Category field, preserving each
// habit's relative order within its bucket (habits already come sorted by
// sort_order, id from the store). The uncategorized bucket ("") always
// sorts last, after every named category in first-seen order.
func groupHabits(habits []*store.Habit) []HabitGroup {
	index := map[string]int{}
	var groups []HabitGroup

	for _, h := range habits {
		if i, ok := index[h.Category]; ok {
			groups[i].Habits = append(groups[i].Habits, h)
			continue
		}
		index[h.Category] = len(groups)
		groups = append(groups, HabitGroup{Name: h.Category, Habits: []*store.Habit{h}})
	}

	if i, ok := index[""]; ok && i != len(groups)-1 {
		uncategorized := groups[i]
		groups = append(groups[:i], groups[i+1:]...)
		groups = append(groups, uncategorized)
	}

	return groups
}

// HabitStatusGroup is groupHabitStatuses's per-category bucket, used by the
// /today page.
type HabitStatusGroup struct {
	Name  string
	Items []HabitWithStatus
}

// groupHabitStatuses is groupHabits' counterpart for /today's
// already-decorated HabitWithStatus items - same bucketing rule (by
// Category, uncategorized bucket last), applied after today's amounts and
// streaks have been attached.
func groupHabitStatuses(items []HabitWithStatus) []HabitStatusGroup {
	index := map[string]int{}
	var groups []HabitStatusGroup

	for _, it := range items {
		if i, ok := index[it.Category]; ok {
			groups[i].Items = append(groups[i].Items, it)
			continue
		}
		index[it.Category] = len(groups)
		groups = append(groups, HabitStatusGroup{Name: it.Category, Items: []HabitWithStatus{it}})
	}

	if i, ok := index[""]; ok && i != len(groups)-1 {
		uncategorized := groups[i]
		groups = append(groups[:i], groups[i+1:]...)
		groups = append(groups, uncategorized)
	}

	return groups
}

// HabitStatsGroup is groupHabitStats' per-category bucket, used by the
// dashboard.
type HabitStatsGroup struct {
	Name  string
	Items []HabitStats
}

// groupHabitStats is groupHabits' counterpart for the dashboard's per-habit
// HabitStats - same bucketing rule (by Category, uncategorized bucket
// last).
func groupHabitStats(stats []HabitStats) []HabitStatsGroup {
	index := map[string]int{}
	var groups []HabitStatsGroup

	for _, s := range stats {
		cat := s.Habit.Category
		if i, ok := index[cat]; ok {
			groups[i].Items = append(groups[i].Items, s)
			continue
		}
		index[cat] = len(groups)
		groups = append(groups, HabitStatsGroup{Name: cat, Items: []HabitStats{s}})
	}

	if i, ok := index[""]; ok && i != len(groups)-1 {
		uncategorized := groups[i]
		groups = append(groups[:i], groups[i+1:]...)
		groups = append(groups, uncategorized)
	}

	return groups
}
