package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

type adminUsersVM struct {
	Users []domain.User
	Error string
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.users.List(r.Context())
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.render(w, r, "page_admin_users", adminUsersVM{Users: users})
}

// handleAdminUserCreate invites an email — it's added to the allowlist
// immediately (GoogleSub stays empty), and that email can then complete
// Google sign-in for the first time. There's no separate "send an invite
// email" step; the admin is expected to tell the person directly.
func (s *Server) handleAdminUserCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not parse form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.PostForm.Get("email"))
	if email == "" {
		s.renderAdminUsersError(w, r, "Email is required.")
		return
	}

	_, err := s.users.Create(r.Context(), domain.User{
		Email:   email,
		IsAdmin: r.PostForm.Get("is_admin") == "on",
	})
	if errors.Is(err, store.ErrAlreadyExists) {
		s.renderAdminUsersError(w, r, "That email has already been invited.")
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

// handleAdminUserDelete removes a user — cascading (ON DELETE CASCADE) to
// all of their habits, entries, reminders, and push subscriptions. Refuses
// to let an admin delete their own account, so a solo admin can never
// accidentally lock themselves out.
func (s *Server) handleAdminUserDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if id == mustUser(r).ID {
		s.renderAdminUsersError(w, r, "You can't remove your own account.")
		return
	}
	if err := s.users.Delete(r.Context(), id); err != nil && !errors.Is(err, store.ErrNotFound) {
		s.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

func (s *Server) renderAdminUsersError(w http.ResponseWriter, r *http.Request, msg string) {
	users, err := s.users.List(r.Context())
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.render(w, r, "page_admin_users", adminUsersVM{Users: users, Error: msg})
}
