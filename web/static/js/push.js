function urlBase64ToUint8Array(base64String) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = window.atob(base64);
  return Uint8Array.from([...raw].map((c) => c.charCodeAt(0)));
}

function csrfHeaders() {
  const meta = document.querySelector('meta[name="csrf-token"]');
  return meta ? { "X-CSRF-Token": meta.content, "Content-Type": "application/json" } : { "Content-Type": "application/json" };
}

async function enablePushNotifications() {
  if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
    alert("Push notifications aren't supported on this browser.");
    return;
  }

  const permission = await Notification.requestPermission();
  if (permission !== "granted") return;

  const reg = await navigator.serviceWorker.ready;
  const keyRes = await fetch("/api/vapid-public-key");
  const { publicKey } = await keyRes.json();

  const sub = await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(publicKey),
  });

  await fetch("/push/subscribe", {
    method: "POST",
    headers: csrfHeaders(),
    body: JSON.stringify(sub.toJSON()),
  });

  document.dispatchEvent(new CustomEvent("rhythms:push-enabled"));
}

async function disablePushNotifications() {
  if (!("serviceWorker" in navigator)) return;
  const reg = await navigator.serviceWorker.ready;
  const sub = await reg.pushManager.getSubscription();
  if (!sub) return;

  await fetch("/push/unsubscribe", {
    method: "POST",
    headers: csrfHeaders(),
    body: JSON.stringify({ endpoint: sub.endpoint }),
  });
  await sub.unsubscribe();

  document.dispatchEvent(new CustomEvent("rhythms:push-disabled"));
}
