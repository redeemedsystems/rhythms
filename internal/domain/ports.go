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

	// Reorder assigns positions 0..len(orderedIDs)-1 in the given order, in
	// a single transaction. It's the persistence side of drag-reorder.
	Reorder(ctx context.Context, orderedIDs []int64) error
}

type EntryRepo interface {
	// ListAll returns every known entry for a habit, oldest first. Required
	// by ComputeEntries, which needs the habit's entire history to build
	// auto-fill intervals correctly — see ComputeEntries' doc comment.
	ListAll(ctx context.Context, habitID int64) ([]Entry, error)

	// ListRange returns entries in [from, to], one per date that has a row.
	// Dates with no row are simply absent from the result, not zero-valued.
	// Both Value and NumericValue are always decoded from the stored row;
	// which one is meaningful depends on the habit's type (see Upsert).
	ListRange(ctx context.Context, habitID int64, from, to Date) ([]Entry, error)
	Get(ctx context.Context, habitID int64, date Date) (Entry, bool, error)

	// Upsert stores an entry. habitType decides how it's encoded: boolean
	// habits store Value (the EntryValue enum) directly, numeric habits
	// store NumericValue as a fixed-point integer (*1000) in the same
	// column — matching uHabits' own schema convention of one dual-purpose
	// column per habit type (see the rewrite plan's deviation #6).
	Upsert(ctx context.Context, habitType HabitType, e Entry) error
}

// ReminderRepo manages the optional per-habit reminder row. A habit with no
// reminder configured simply has no row — Get's second return distinguishes
// that from a genuinely-zero hour/minute.
type ReminderRepo interface {
	Get(ctx context.Context, habitID int64) (Reminder, bool, error)
	Set(ctx context.Context, r Reminder) error
	Delete(ctx context.Context, habitID int64) error

	// ListActive returns every habit id that has a reminder configured,
	// paired with the reminder itself — what the scheduler polls each tick.
	ListActive(ctx context.Context) (map[int64]Reminder, error)
}

// PushSubscriptionRepo persists browsers' Web Push registrations.
type PushSubscriptionRepo interface {
	List(ctx context.Context) ([]PushSubscription, error)
	Upsert(ctx context.Context, s PushSubscription) error
	Delete(ctx context.Context, endpoint string) error
}

// ReminderLogRepo dedupes reminder sends: a poll-based scheduler checks
// "at or past due" every tick, so it needs to remember it already sent
// today's notification for a given habit.
type ReminderLogRepo interface {
	WasSent(ctx context.Context, habitID int64, date Date) (bool, error)
	MarkSent(ctx context.Context, habitID int64, date Date) error
}
