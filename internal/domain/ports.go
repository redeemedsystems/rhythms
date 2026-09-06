package domain

import "context"

// HabitRepo and EntryRepo are implemented by internal/store and consumed by
// internal/web, so handlers can be tested against fakes without a real DB.
type HabitRepo interface {
	List(ctx context.Context, includeArchived bool) ([]Habit, error)
	Get(ctx context.Context, id int64) (Habit, error)
	Create(ctx context.Context, h Habit) (int64, error)
	Update(ctx context.Context, h Habit) error
	SetArchived(ctx context.Context, id int64, archived bool) error
	Delete(ctx context.Context, id int64) error
}

type EntryRepo interface {
	// ListRange returns entries in [from, to], one per date that has a row.
	// Dates with no row are simply absent from the result, not zero-valued.
	ListRange(ctx context.Context, habitID int64, from, to Date) ([]Entry, error)
	Get(ctx context.Context, habitID int64, date Date) (Entry, bool, error)
	Upsert(ctx context.Context, e Entry) error
}
