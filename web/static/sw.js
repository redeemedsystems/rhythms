// Service worker for the Rhythms PWA. Two jobs: show a notification when a
// push arrives, and focus (or open) the app when that notification is
// tapped. Registered at root scope (see /sw.js in internal/web) so it can
// control the whole app, not just /static/.

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
