package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"rhythms/internal/domain"
)

func newTestUserRepo(t *testing.T) *UserRepo {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewUserRepo(db)
}

func TestUserRepoCreateGetGetByEmailList(t *testing.T) {
	ctx := context.Background()
	repo := newTestUserRepo(t)

	id, err := repo.Create(ctx, domain.User{Email: "a@example.com", IsAdmin: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Email != "a@example.com" || !got.IsAdmin {
		t.Errorf("Get = %+v, want Email=a@example.com IsAdmin=true", got)
	}
	if got.GoogleSub != "" {
		t.Errorf("Get.GoogleSub = %q, want empty (not yet signed in)", got.GoogleSub)
	}

	byEmail, err := repo.GetByEmail(ctx, "a@example.com")
	if err != nil || byEmail.ID != id {
		t.Errorf("GetByEmail = %+v err=%v, want id %d", byEmail, err, id)
	}

	users, err := repo.List(ctx)
	if err != nil || len(users) != 1 {
		t.Fatalf("List: users=%v err=%v", users, err)
	}
}

func TestUserRepoGetNotFound(t *testing.T) {
	repo := newTestUserRepo(t)
	if _, err := repo.Get(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(999) error = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetByEmail(context.Background(), "nobody@example.com"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByEmail(nobody) error = %v, want ErrNotFound", err)
	}
}

func TestUserRepoCreateDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	repo := newTestUserRepo(t)

	if _, err := repo.Create(ctx, domain.User{Email: "dup@example.com"}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if _, err := repo.Create(ctx, domain.User{Email: "dup@example.com"}); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("second Create error = %v, want ErrAlreadyExists", err)
	}
}

func TestUserRepoSetGoogleSubActivates(t *testing.T) {
	ctx := context.Background()
	repo := newTestUserRepo(t)

	id, err := repo.Create(ctx, domain.User{Email: "invited@example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.SetGoogleSub(ctx, id, "google-sub-123"); err != nil {
		t.Fatalf("SetGoogleSub: %v", err)
	}
	got, err := repo.Get(ctx, id)
	if err != nil || got.GoogleSub != "google-sub-123" {
		t.Errorf("Get after SetGoogleSub = %+v err=%v, want GoogleSub=google-sub-123", got, err)
	}
}

func TestUserRepoDelete(t *testing.T) {
	ctx := context.Background()
	repo := newTestUserRepo(t)

	id, err := repo.Create(ctx, domain.User{Email: "temp@example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete error = %v, want ErrNotFound", err)
	}
}

func TestUserRepoEnsureAdminCreatesThenIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := newTestUserRepo(t)

	if err := repo.EnsureAdmin(ctx, "admin@example.com"); err != nil {
		t.Fatalf("EnsureAdmin (create): %v", err)
	}
	u, err := repo.GetByEmail(ctx, "admin@example.com")
	if err != nil || !u.IsAdmin {
		t.Fatalf("GetByEmail after EnsureAdmin = %+v err=%v, want IsAdmin=true", u, err)
	}

	if err := repo.EnsureAdmin(ctx, "admin@example.com"); err != nil {
		t.Fatalf("EnsureAdmin (idempotent): %v", err)
	}
	users, err := repo.List(ctx)
	if err != nil || len(users) != 1 {
		t.Fatalf("List after repeated EnsureAdmin: users=%v err=%v, want exactly 1", users, err)
	}
}

func TestUserRepoEnsureAdminPromotesExistingUser(t *testing.T) {
	ctx := context.Background()
	repo := newTestUserRepo(t)

	id, err := repo.Create(ctx, domain.User{Email: "promote@example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.EnsureAdmin(ctx, "promote@example.com"); err != nil {
		t.Fatalf("EnsureAdmin (promote): %v", err)
	}
	got, err := repo.Get(ctx, id)
	if err != nil || !got.IsAdmin {
		t.Errorf("Get after EnsureAdmin promotion = %+v err=%v, want IsAdmin=true", got, err)
	}
}
