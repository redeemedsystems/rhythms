document.body.addEventListener("htmx:configRequest", (evt) => {
  const meta = document.querySelector('meta[name="csrf-token"]');
  if (meta) {
    evt.detail.headers["X-CSRF-Token"] = meta.content;
  }
});

// A 403 here almost always means this tab's CSRF token went stale (e.g. the
// session was rotated by a password reset in another tab/device). Reload so
// the page picks up the current session's token instead of leaving the user
// stuck on a dead button.
document.body.addEventListener("htmx:responseError", (evt) => {
  if (evt.detail.xhr.status === 403) {
    window.location.reload();
  }
});
