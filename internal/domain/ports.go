package domain

import "context"

// HabitRepo persists Habit records: CRUD, archive/unarchive, and reorder.
// Implemented by internal/store, consumed by internal/web — repos in this
// file exist so handlers (and the reminder scheduler) can be tested against
// fakes without a real database; see internal/web/fakes_test.go and
// internal/reminder/scheduler_test.go.
//
// Every method below except GetAny takes userID and scopes to it — a habit
// that exists but belongs to a different user is indistinguishable from a
// nonexistent one (ErrNotFound for both), so a guessed/stolen id never
// leaks whether it belongs to someone else. The repo enforces this, not
// the caller, precisely so a handler bug (forgetting to check ownership)
// can't silently cross tenants.
type HabitRepo interface {
	List(ctx context.Context, userID int64, includeArchived bool) ([]Habit, error)
	Get(ctx context.Context, userID, id int64) (Habit, error)

	// GetAny bypasses ownership scoping entirely. It exists solely for
	// trusted background code that legitimately needs to look up any
	// user's habit (the reminder scheduler, scheduling for everyone) — it
	// must never be reachable from a web handler, which always has an
	// attacker-controlled id and must use the scoped Get instead.
	GetAny(ctx context.Context, id int64) (Habit, error)

	Create(ctx context.Context, userID int64, h Habit) (int64, error)
	Update(ctx context.Context, userID int64, h Habit) error
	SetArchived(ctx context.Context, userID, id int64, archived bool) error
	Delete(ctx context.Context, userID, id int64) error

	// Reorder assigns positions 0..len(orderedIDs)-1 in the given order, in
	// a single transaction. It's the persistence side of drag-reorder. Any
	// id in orderedIDs that isn't owned by userID is rejected wholesale
	// (ErrNotFound) rather than silently skipped, so a caller never gets a
	// false "success" for an id it didn't actually reorder.
	Reorder(ctx context.Context, userID int64, orderedIDs []int64) error
}

// EntryRepo persists per-day Entry records for habits.
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
	// List returns only userID's own subscriptions — the reminder
	// scheduler calls this once per due reminder, scoped to that habit's
	// owner, so a notification only ever reaches its owner's devices.
	List(ctx context.Context, userID int64) ([]PushSubscription, error)
	Upsert(ctx context.Context, userID int64, s PushSubscription) error

	// Delete stays unscoped by user: the endpoint is an unguessable,
	// high-entropy secret URL assigned by the push service, not something
	// an attacker can enumerate. Worst case of a wrong caller deleting one
	// is that device stops getting notifications — not a data leak.
	Delete(ctx context.Context, endpoint string) error
}

// ReminderLogRepo dedupes reminder sends: a poll-based scheduler checks
// "at or past due" every tick, so it needs to remember it already sent
// today's notification for a given habit.
type ReminderLogRepo interface {
	WasSent(ctx context.Context, habitID int64, date Date) (bool, error)
	MarkSent(ctx context.Context, habitID int64, date Date) error
}

// UserRepo persists accounts. An admin invites an email (Create), which
// creates a User with no GoogleSub yet; that email's first successful
// Google sign-in calls SetGoogleSub to activate it. Signing in with an
// email that has no User row at all is rejected elsewhere (there's no
// open signup) — this repo just stores what admins have invited.
type UserRepo interface {
	Get(ctx context.Context, id int64) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context) ([]User, error)
	Create(ctx context.Context, u User) (int64, error)
	SetGoogleSub(ctx context.Context, id int64, sub string) error
	Delete(ctx context.Context, id int64) error
}
