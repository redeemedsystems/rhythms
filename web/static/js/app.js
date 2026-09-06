// Event names match the vendored htmx 4.0.0 build (web/htmx/htmx.min.js),
// which uses colon-delimited names (htmx:config:request) instead of the
// camelCase names (htmx:configRequest) older htmx versions use, and nests
// the request/response under evt.detail.ctx rather than evt.detail directly.
document.body.addEventListener("htmx:config:request", (evt) => {
  const meta = document.querySelector('meta[name="csrf-token"]');
  if (meta) {
    evt.detail.ctx.request.headers["X-CSRF-Token"] = meta.content;
  }
});

// A 403 here almost always means this tab's CSRF token went stale (e.g. the
// session was rotated by a password reset in another tab/device). Reload so
// the page picks up the current session's token instead of leaving the user
// stuck on a dead button.
document.body.addEventListener("htmx:response:error", (evt) => {
  if (evt.detail.ctx.response.status === 403) {
    window.location.reload();
  }
});
