package server

import (
	"reflect"
	"testing"

	"rhythms/internal/store"
)

func TestGroupHabits(t *testing.T) {
	habits := []*store.Habit{
		{ID: 1, Category: ""},
		{ID: 2, Category: "Health"},
		{ID: 3, Category: ""},
		{ID: 4, Category: "Health"},
		{ID: 5, Category: "Work"},
	}

	groups := groupHabits(habits)

	var names []string
	for _, g := range groups {
		names = append(names, g.Name)
	}
	want := []string{"Health", "Work", ""}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("group order = %v, want %v (uncategorized must sort last)", names, want)
	}

	health := groups[0]
	if len(health.Habits) != 2 || health.Habits[0].ID != 2 || health.Habits[1].ID != 4 {
		t.Fatalf("Health group = %+v, want habits 2 then 4 in order", health.Habits)
	}

	uncategorized := groups[2]
	if len(uncategorized.Habits) != 2 || uncategorized.Habits[0].ID != 1 || uncategorized.Habits[1].ID != 3 {
		t.Fatalf("uncategorized group = %+v, want habits 1 then 3 in order", uncategorized.Habits)
	}
}

func TestGroupHabitsAllUncategorized(t *testing.T) {
	habits := []*store.Habit{{ID: 1}, {ID: 2}}
	groups := groupHabits(habits)
	if len(groups) != 1 || groups[0].Name != "" {
		t.Fatalf("groups = %+v, want a single uncategorized group", groups)
	}
}
