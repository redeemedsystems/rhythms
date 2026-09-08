package web

import "net/http"

// handleSettings renders the per-user settings page — currently just the
// push-notification toggle (formerly on /today), which every signed-in
// user manages for themselves rather than through the admin-only
// /admin/users page.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "page_settings", nil)
}
