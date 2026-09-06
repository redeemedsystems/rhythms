// Service worker for the Rhythms PWA. Two jobs: show a notification when a
// push arrives, and focus (or open) the app when that notification is
// tapped. Registered at root scope (see /sw.js in internal/web) so it can
// control the whole app, not just /static/.
//
// skipWaiting()/clients.claim(): without these, a newly-deployed sw.js sits
// "waiting" behind whatever service worker previously controlled this
// origin until every open tab is fully closed and reopened — on a domain
// that's had a *different* app's service worker before (as
// rhythms.redeemed.systems did), that old worker stays in control
// indefinitely, breaking push subscription state in ways that are hard to
// diagnose from the page's own JS. Forcing immediate takeover on install
// means a plain reload always gets the current worker.
self.addEventListener('install', function (event) {
    event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', function (event) {
    event.waitUntil(clients.claim());
});

self.addEventListener('push', function (event) {
    let data = { title: 'Rhythms', body: 'A habit is due.', url: '/today' };
    try {
        if (event.data) data = Object.assign(data, event.data.json());
    } catch (e) {
        // Malformed payload: fall back to the defaults above rather than
        // showing nothing.
    }

    event.waitUntil(
        self.registration.showNotification(data.title, {
            body: data.body,
            icon: '/static/icons/icon-192.png',
            badge: '/static/icons/icon-192.png',
            data: { url: data.url },
        })
    );
});

self.addEventListener('notificationclick', function (event) {
    event.notification.close();
    const url = (event.notification.data && event.notification.data.url) || '/today';

    event.waitUntil(
        clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (windowClients) {
            for (const client of windowClients) {
                if (client.url.includes(url) && 'focus' in client) return client.focus();
            }
            for (const client of windowClients) {
                if ('focus' in client) return client.focus().then(() => client.navigate(url));
            }
            if (clients.openWindow) return clients.openWindow(url);
        })
    );
});
