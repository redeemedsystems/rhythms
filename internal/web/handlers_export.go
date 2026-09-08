package web

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"time"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

var filenameSanitizer = regexp.MustCompile(`[^ a-zA-Z0-9._-]+`)

// sanitizeFilename mirrors uHabits' own CSV-export filename sanitizer:
// strip anything but a small safe character set, cap at 100 characters.
func sanitizeFilename(name string) string {
	s := filenameSanitizer.ReplaceAllString(name, "")
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}

// handleHabitExportCSV streams one habit's computed entry history as CSV, in
// the same Date/Value/Notes shape as uHabits' own per-habit Checkmarks.csv —
// Value uses the same symbolic names (YES_MANUAL, YES_AUTO, ...) for boolean
// habits, or the raw fixed-point integer for numeric ones (a real quirk of
// upstream's single shared column, reproduced here rather than "fixed", so
// re-importing behaves the same way it would against the original app).
func (s *Server) handleHabitExportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h, err := s.habits.Get(r.Context(), mustUser(r).ID, id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}

	original, err := s.entries.ListAll(r.Context(), id)
	if err != nil {
		s.serverError(w, err)
		return
	}
	computed := domain.ComputeEntries(h, original)
	sort.Slice(computed, func(i, j int) bool { return computed[j].Date.Before(computed[i].Date) }) // newest first, matching upstream

	filename := sanitizeFilename(h.Name) + ".csv"
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	cw := csv.NewWriter(w)
	cw.Write([]string{"Date", "Value", "Notes"})
	for _, e := range computed {
		cw.Write([]string{e.Date.String(), domain.FormattedValue(e.Value), e.Notes})
	}
	cw.Flush()
}

// handleBackup streams a consistent snapshot of the whole SQLite database.
// It uses SQLite's own VACUUM INTO rather than copying the file's bytes
// directly, since the live file may be mid-write or split across a WAL —
// VACUUM INTO guarantees a single, complete, importable database file no
// matter what state the original is in.
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	tmpFile, err := os.CreateTemp("", "rhythms-backup-*.db")
	if err != nil {
		s.serverError(w, err)
		return
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)
	os.Remove(tmpPath) // VACUUM INTO refuses to write to an existing file

	if _, err := s.db.ExecContext(r.Context(), `VACUUM INTO ?`, tmpPath); err != nil {
		s.serverError(w, err)
		return
	}

	filename := fmt.Sprintf("rhythms-backup-%s.db", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.sqlite3")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(w, r, tmpPath)
}
