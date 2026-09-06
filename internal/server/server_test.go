package server

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"rhythms/internal/config"
	"rhythms/internal/mail"
	"rhythms/internal/push"
	"rhythms/internal/store"
	"rhythms/web"
)

// newTestServer starts a test server backed by a fresh temp database. An
// optional mailer lets a test capture outgoing emails (e.g. a password reset
// link); it defaults to mail.LogSender{}, which discards them.
func newTestServer(t *testing.T, mailer ...mail.Sender) (*httptest.Server, *http.Client) {
	t.Helper()

	var m mail.Sender = mail.LogSender{}
	if len(mailer) > 0 {
		m = mailer[0]
	}

	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	renderer, err := NewRenderer(web.FS)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	cfg := config.Config{SecureCookies: false, VAPIDSubscriber: "mailto:test@example.com"}

	pub, priv, err := push.LoadOrGenerateVAPIDKeys(db)
	if err != nil {
		t.Fatalf("LoadOrGenerateVAPIDKeys: %v", err)
	}
	sender := push.NewSender(pub, priv, cfg.VAPIDSubscriber)

	srv := New(db, renderer, web.FS, cfg, pub, sender, m)
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return ts, client
}

var csrfRe = regexp.MustCompile(`name="csrf-token" content="([^"]+)"`)

func extractCSRF(t *testing.T, body string) string {
	t.Helper()
	m := csrfRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("could not find csrf token in body: %s", body)
	}
	return m[1]
}

// httpResult is the outcome of a single request: its status code and fully
// read body. The response is always closed before doRequest returns, so
// callers never need to manage the body's lifetime themselves.
type httpResult struct {
	status int
	body   string
	header http.Header
}

func doRequest(t *testing.T, client *http.Client, method, url string, headers map[string]string) httpResult {
	t.Helper()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return httpResult{status: resp.StatusCode, body: string(b), header: resp.Header}
}

func doGet(t *testing.T, client *http.Client, url string) httpResult {
	t.Helper()
	return doRequest(t, client, http.MethodGet, url, nil)
}

func doPostForm(t *testing.T, client *http.Client, targetURL string, form url.Values) httpResult {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", targetURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return httpResult{status: resp.StatusCode, body: string(b), header: resp.Header}
}

func TestFullUserJourney(t *testing.T) {
	ts, client := newTestServer(t)

	// Register.
	res := doPostForm(t, client, ts.URL+"/register", url.Values{
		"email":    {"journey@example.com"},
		"password": {"correct-horse-battery"},
		"timezone": {"America/Chicago"},
	})
	if res.status != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect from register, got %d", res.status)
	}

	// Today view, no habits yet.
	res = doGet(t, client, ts.URL+"/today")
	if res.status != http.StatusOK {
		t.Fatalf("expected 200 from /today, got %d", res.status)
	}
	if !strings.Contains(res.body, "No habits yet") {
		t.Fatalf("expected empty-state message, got: %s", res.body)
	}
	csrf := extractCSRF(t, res.body)

	// Create a boolean habit.
	res = doPostForm(t, client, ts.URL+"/habits/new", url.Values{
		"csrf_token":    {csrf},
		"name":          {"Take vitamins"},
		"type":          {"boolean"},
		"schedule_kind": {"daily"},
	})
	if res.status != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect from habit create, got %d", res.status)
	}

	// Habits list should show it.
	res = doGet(t, client, ts.URL+"/habits")
	if !strings.Contains(res.body, "Take vitamins") {
		t.Fatalf("expected habit in list, got: %s", res.body)
	}

	// Today view should now show the habit card with an ID we can log against.
	res = doGet(t, client, ts.URL+"/today")
	idRe := regexp.MustCompile(`id="habit-(\d+)"`)
	m := idRe.FindStringSubmatch(res.body)
	if m == nil {
		t.Fatalf("expected habit card id in today view, got: %s", res.body)
	}
	habitID := m[1]

	// Log it via the htmx endpoint.
	res = doRequest(t, client, http.MethodPost, ts.URL+"/habits/"+habitID+"/log", map[string]string{
		"HX-Request":   "true",
		"X-CSRF-Token": csrf,
	})
	if res.status != http.StatusOK {
		t.Fatalf("expected 200 from habit log, got %d", res.status)
	}
	if strings.Contains(res.body, "<html") {
		t.Fatalf("partial response should not include full page layout, got: %s", res.body)
	}
	if !strings.Contains(res.body, "Done") {
		t.Fatalf("expected habit card to show as done, got: %s", res.body)
	}

	// Dashboard should render without error and mention the habit.
	res = doGet(t, client, ts.URL+"/dashboard")
	if res.status != http.StatusOK || !strings.Contains(res.body, "Take vitamins") {
		t.Fatalf("expected dashboard to show habit, status=%d body=%s", res.status, res.body)
	}

	// Logging without a valid CSRF token must be rejected.
	res = doRequest(t, client, http.MethodPost, ts.URL+"/habits/"+habitID+"/log", map[string]string{
		"HX-Request":   "true",
		"X-CSRF-Token": "wrong-token",
	})
	if res.status != http.StatusForbidden {
		t.Fatalf("expected 403 for bad csrf token, got %d", res.status)
	}

	// Logout, then confirm /today redirects to /login.
	res = doRequest(t, client, http.MethodPost, ts.URL+"/logout", map[string]string{
		"X-CSRF-Token": csrf,
	})
	if res.status != http.StatusSeeOther {
		t.Fatalf("expected redirect from logout, got %d", res.status)
	}

	res = doGet(t, client, ts.URL+"/today")
	if res.status != http.StatusSeeOther {
		t.Fatalf("expected redirect to login after logout, got %d", res.status)
	}
	if loc := res.header.Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	ts, client := newTestServer(t)

	res := doPostForm(t, client, ts.URL+"/register", url.Values{
		"email":    {"short@example.com"},
		"password": {"short"},
	})
	if res.status != http.StatusOK {
		t.Fatalf("expected 200 (re-rendered form) for invalid password, got %d", res.status)
	}
	if !strings.Contains(res.body, "at least 8 characters") {
		t.Fatalf("expected validation message, got: %s", res.body)
	}
}

// capturingMailer records sent emails instead of delivering them, so a test
// can pull the reset link out of the body.
type capturingMailer struct {
	to, subject, body string
}

func (m *capturingMailer) Send(to, subject, body string) error {
	m.to, m.subject, m.body = to, subject, body
	return nil
}

func TestPasswordResetFlow(t *testing.T) {
	mailer := &capturingMailer{}
	ts, client := newTestServer(t, mailer)

	doPostForm(t, client, ts.URL+"/register", url.Values{
		"email":    {"reset@example.com"},
		"password": {"original-password"},
	})
	// Registering logs the client in; use a fresh cookie-less client for the
	// reset flow so it isn't riding the registration session.
	client = &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	res := doPostForm(t, client, ts.URL+"/forgot-password", url.Values{"email": {"reset@example.com"}})
	if res.status != http.StatusOK || !strings.Contains(res.body, "we've sent a link") {
		t.Fatalf("expected generic sent message, status=%d body=%s", res.status, res.body)
	}
	if mailer.to != "reset@example.com" {
		t.Fatalf("expected email sent to reset@example.com, got %q", mailer.to)
	}

	tokenRe := regexp.MustCompile(`token=([\w-]+)`)
	m := tokenRe.FindStringSubmatch(mailer.body)
	if m == nil {
		t.Fatalf("expected reset link in email body: %s", mailer.body)
	}
	token := m[1]

	// Requesting a reset for an unknown email must produce the same message
	// and must not send mail.
	mailer.to = ""
	res = doPostForm(t, client, ts.URL+"/forgot-password", url.Values{"email": {"nobody@example.com"}})
	if res.status != http.StatusOK || !strings.Contains(res.body, "we've sent a link") {
		t.Fatalf("expected generic sent message for unknown email, status=%d body=%s", res.status, res.body)
	}
	if mailer.to != "" {
		t.Fatalf("expected no email sent for unknown address, got %q", mailer.to)
	}

	// A bad token is rejected.
	res = doPostForm(t, client, ts.URL+"/reset-password", url.Values{
		"token":    {"not-a-real-token"},
		"password": {"new-password-123"},
	})
	if !strings.Contains(res.body, "invalid or has expired") {
		t.Fatalf("expected invalid token message, got: %s", res.body)
	}

	// The real token resets the password.
	res = doPostForm(t, client, ts.URL+"/reset-password", url.Values{
		"token":    {token},
		"password": {"new-password-123"},
	})
	if res.status != http.StatusSeeOther || res.header.Get("Location") != "/login?reset=success" {
		t.Fatalf("expected redirect to /login?reset=success, got status=%d location=%q", res.status, res.header.Get("Location"))
	}

	// The token is single-use.
	res = doPostForm(t, client, ts.URL+"/reset-password", url.Values{
		"token":    {token},
		"password": {"another-password-456"},
	})
	if !strings.Contains(res.body, "invalid or has expired") {
		t.Fatalf("expected token reuse to be rejected, got: %s", res.body)
	}

	// Old password no longer works; new password does.
	res = doPostForm(t, client, ts.URL+"/login", url.Values{
		"email":    {"reset@example.com"},
		"password": {"original-password"},
	})
	if !strings.Contains(res.body, "Invalid email or password") {
		t.Fatalf("expected old password to be rejected, got: %s", res.body)
	}

	res = doPostForm(t, client, ts.URL+"/login", url.Values{
		"email":    {"reset@example.com"},
		"password": {"new-password-123"},
	})
	if res.status != http.StatusSeeOther {
		t.Fatalf("expected new password to log in, got status=%d body=%s", res.status, res.body)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	ts, client := newTestServer(t)

	doPostForm(t, client, ts.URL+"/register", url.Values{
		"email":    {"wrongpw@example.com"},
		"password": {"correct-horse-battery"},
	})

	res := doPostForm(t, client, ts.URL+"/login", url.Values{
		"email":    {"wrongpw@example.com"},
		"password": {"totally-wrong"},
	})
	if !strings.Contains(res.body, "Invalid email or password") {
		t.Fatalf("expected invalid credentials message, got: %s", res.body)
	}
}
