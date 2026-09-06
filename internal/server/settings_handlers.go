package server

import (
	"net/http"
	"strings"

	"rhythms/internal/auth"
	"rhythms/internal/store"
)

type SettingsPageData struct {
	Base
	DigestTime string
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	s.render.Page(w, "settings.html", SettingsPageData{
		Base:       s.baseFor(r, "settings"),
		DigestTime: user.DigestTime,
	})
}

func (s *Server) handleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	user := auth.UserFromContext(r.Context())
	digestTime := strings.TrimSpace(r.FormValue("digest_time"))
	if digestTime != "" && !isValidHHMM(digestTime) {
		s.render.Page(w, "settings.html", SettingsPageData{
			Base:       Base{LoggedIn: true, Nav: "settings", Flash: "Digest time must be in HH:MM form.", CSRFToken: s.csrfFor(r)},
			DigestTime: digestTime,
		})
		return
	}

	if err := store.UpdateDigestTime(s.db, user.ID, digestTime); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
