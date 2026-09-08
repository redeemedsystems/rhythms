package store

import (
	"context"
	"path/filepath"
	"testing"

	"rhythms/internal/domain"
)

func TestPushSubscriptionRepoUpsertListDelete(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	userID, err := NewUserRepo(db).Create(ctx, domain.User{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	repo := NewPushSubscriptionRepo(db)

	sub := domain.PushSubscription{Endpoint: "https://push.example/abc", P256dh: "p256dh-key", Auth: "auth-key"}
	if err := repo.Upsert(ctx, userID, sub); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	subs, err := repo.List(ctx, userID)
	if err != nil || len(subs) != 1 {
		t.Fatalf("List: subs=%v err=%v", subs, err)
	}
	sub.UserID = userID
	if subs[0] != sub {
		t.Errorf("List[0] = %+v, want %+v", subs[0], sub)
	}

	// Upserting the same endpoint again must update, not duplicate.
	sub.P256dh = "new-p256dh"
	if err := repo.Upsert(ctx, userID, sub); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	subs, err = repo.List(ctx, userID)
	if err != nil || len(subs) != 1 || subs[0].P256dh != "new-p256dh" {
		t.Fatalf("List after overwrite: subs=%v err=%v", subs, err)
	}

	if err := repo.Delete(ctx, sub.Endpoint); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	subs, err = repo.List(ctx, userID)
	if err != nil || len(subs) != 0 {
		t.Fatalf("List after delete: subs=%v err=%v", subs, err)
	}
}

func TestPushSubscriptionRepoUpsertTransfersOwnerOnConflict(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	users := NewUserRepo(db)
	user1, err := users.Create(ctx, domain.User{Email: "one@example.com"})
	if err != nil {
		t.Fatalf("seed user 1: %v", err)
	}
	user2, err := users.Create(ctx, domain.User{Email: "two@example.com"})
	if err != nil {
		t.Fatalf("seed user 2: %v", err)
	}
	repo := NewPushSubscriptionRepo(db)

	sub := domain.PushSubscription{Endpoint: "https://push.example/shared", P256dh: "k", Auth: "a"}
	if err := repo.Upsert(ctx, user1, sub); err != nil {
		t.Fatalf("Upsert (user1): %v", err)
	}
	if err := repo.Upsert(ctx, user2, sub); err != nil {
		t.Fatalf("Upsert (user2): %v", err)
	}

	subs1, err := repo.List(ctx, user1)
	if err != nil || len(subs1) != 0 {
		t.Errorf("List(user1) after re-subscribe = %v, want none (ownership should have transferred)", subs1)
	}
	subs2, err := repo.List(ctx, user2)
	if err != nil || len(subs2) != 1 {
		t.Errorf("List(user2) after re-subscribe = %v, want the transferred subscription", subs2)
	}
}

func TestSettingsEnsureVAPIDKeysStable(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	repo := NewSettingsRepo(db)

	pub1, priv1, err := repo.EnsureVAPIDKeys(ctx)
	if err != nil {
		t.Fatalf("EnsureVAPIDKeys (first): %v", err)
	}
	if pub1 == "" || priv1 == "" {
		t.Fatal("expected non-empty generated keys")
	}

	pub2, priv2, err := repo.EnsureVAPIDKeys(ctx)
	if err != nil {
		t.Fatalf("EnsureVAPIDKeys (second): %v", err)
	}
	if pub1 != pub2 || priv1 != priv2 {
		t.Error("EnsureVAPIDKeys must return the same keypair once generated — existing push subscriptions are only valid for the key they were created under")
	}
}

func TestReminderLogRepoWasSentMarkSent(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	userID, err := NewUserRepo(db).Create(ctx, domain.User{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	habitID, err := NewHabitRepo(db).Create(ctx, userID, domain.Habit{Name: "Meditate"})
	if err != nil {
		t.Fatalf("Create habit: %v", err)
	}
	repo := NewReminderLogRepo(db)
	today := domain.Today()

	sent, err := repo.WasSent(ctx, habitID, today)
	if err != nil || sent {
		t.Fatalf("WasSent before mark: sent=%v err=%v", sent, err)
	}

	if err := repo.MarkSent(ctx, habitID, today); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}
	sent, err = repo.WasSent(ctx, habitID, today)
	if err != nil || !sent {
		t.Fatalf("WasSent after mark: sent=%v err=%v", sent, err)
	}

	// Marking sent twice must not error (ON CONFLICT DO NOTHING).
	if err := repo.MarkSent(ctx, habitID, today); err != nil {
		t.Errorf("second MarkSent should be a no-op, got error: %v", err)
	}

	// A different day is independent.
	sent, err = repo.WasSent(ctx, habitID, today.AddDays(1))
	if err != nil || sent {
		t.Errorf("WasSent for a different day: sent=%v err=%v, want false", sent, err)
	}
}
