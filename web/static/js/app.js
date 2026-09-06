document.body.addEventListener("htmx:configRequest", (evt) => {
  const meta = document.querySelector('meta[name="csrf-token"]');
  if (meta) {
    evt.detail.headers["X-CSRF-Token"] = meta.content;
  }
});
