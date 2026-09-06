// Drag-reorder glue: SortableJS handles the drag interaction, this just
// posts the resulting order to the server. Deliberately the one place this
// project uses hand-written JS instead of pure HTMX — drag-and-drop isn't
// something HTMX covers on its own.
function initHabitSortable() {
    var list = document.querySelector('#habit-list ul.habit-list');
    if (!list || list.dataset.sortableInit) return;
    list.dataset.sortableInit = '1';

    new Sortable(list, {
        handle: '.drag-handle',
        animation: 150,
        ghostClass: 'habit-row-sortable-ghost',
        dragClass: 'dragging',
        onEnd: function () {
            var ids = Array.prototype.map.call(
                list.querySelectorAll('[data-habit-id]'),
                function (el) { return el.dataset.habitId; }
            );
            var params = new URLSearchParams();
            ids.forEach(function (id) { params.append('id', id); });
            fetch('/habits/reorder', { method: 'POST', body: params });
        }
    });
}

document.addEventListener('DOMContentLoaded', initHabitSortable);
document.body.addEventListener('htmx:afterSettle', initHabitSortable);
