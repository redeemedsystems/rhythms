// Web Push subscription glue for the /today page. Feature-detects rather
// than assuming support — Push requires HTTPS (or localhost), so on a plain
// HTTP self-hosted deployment this quietly does nothing rather than erroring.
(function () {
    function base64UrlToUint8Array(base64Url) {
        const padding = '='.repeat((4 - (base64Url.length % 4)) % 4);
        const base64 = (base64Url + padding).replace(/-/g, '+').replace(/_/g, '/');
        const raw = atob(base64);
        const arr = new Uint8Array(raw.length);
        for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
        return arr;
    }

    async function registerServiceWorker() {
        if (!('serviceWorker' in navigator)) return null;
        try {
            return await navigator.serviceWorker.register('/sw.js');
        } catch (e) {
            console.warn('service worker registration failed', e);
            return null;
        }
    }

    async function updateButton(btn, registration) {
        if (!registration || !('PushManager' in window)) {
            btn.hidden = true;
            return;
        }
        const sub = await registration.pushManager.getSubscription();
        btn.hidden = false;
        btn.textContent = sub ? 'Disable reminders on this device' : 'Enable reminders on this device';
        btn.dataset.subscribed = sub ? '1' : '0';
    }

    async function subscribe(registration) {
        const meta = document.querySelector('meta[name="vapid-public-key"]');
        const key = meta && meta.content;
        if (!key) {
            alert('Push notifications are not configured on this server.');
            return;
        }
        const sub = await registration.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey: base64UrlToUint8Array(key),
        });
        await fetch('/push/subscribe', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(sub.toJSON()),
        });
    }

    async function unsubscribe(registration) {
        const sub = await registration.pushManager.getSubscription();
        if (!sub) return;
        await fetch('/push/unsubscribe', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ endpoint: sub.endpoint }),
        });
        await sub.unsubscribe();
    }

    document.addEventListener('DOMContentLoaded', async function () {
        const btn = document.getElementById('push-toggle');
        if (!btn) return; // only present on /today

        const registration = await registerServiceWorker();
        await updateButton(btn, registration);

        btn.addEventListener('click', async function () {
            if (!registration) return;
            try {
                if (btn.dataset.subscribed === '1') {
                    await unsubscribe(registration);
                } else {
                    if (Notification.permission === 'default') {
                        const perm = await Notification.requestPermission();
                        if (perm !== 'granted') return;
                    } else if (Notification.permission === 'denied') {
                        alert('Notifications are blocked for this site in your browser settings.');
                        return;
                    }
                    await subscribe(registration);
                }
            } catch (e) {
                console.error('push subscription toggle failed', e);
            }
            await updateButton(btn, registration);
        });
    });
})();
