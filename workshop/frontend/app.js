// Shared frontend. Identical for both tracks — it just points at a different
// port. Module 7.4's bonus exercise adds browser instrumentation here.

const $ = (id) => document.getElementById(id);

const state = {
  api: localStorage.getItem('workshop-api') || 'http://localhost:8081',
  list: 'work',
};

$('track').value = state.api;
$('track').addEventListener('change', (e) => {
  state.api = e.target.value;
  localStorage.setItem('workshop-api', state.api);
  refresh();
});

function say(message, isError = false) {
  const el = $('status');
  el.textContent = message;
  el.className = isError ? 'error' : '';
  if (message) setTimeout(() => { if (el.textContent === message) el.textContent = ''; }, 4000);
}

async function api(path, options = {}) {
  const res = await fetch(state.api + path, {
    ...options,
    headers: { 'content-type': 'application/json', 'x-client': 'web', ...options.headers },
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.status === 204 ? null : res.json();
}

async function refresh() {
  try {
    const lists = await api('/lists');
    $('lists').innerHTML = lists
      .map((l) => `<button data-list="${l.list}"${l.list === state.list ? ' class="active"' : ''}>${l.list} <span>${l.open}</span></button>`)
      .join('');

    const todos = await api(`/todos?list=${encodeURIComponent(state.list)}`);
    $('todos').innerHTML = todos.length === 0
      ? '<li class="empty">Nothing here.</li>'
      : todos.map(renderTodo).join('');
  } catch (err) {
    say(`Could not load: ${err.message}`, true);
  }
}

function renderTodo(t) {
  const tags = (t.tags || []).map((tag) => `<em>${tag}</em>`).join('');
  return `<li${t.completed_at ? ' class="done"' : ''}>
    <span class="p p${t.priority}"></span>
    <span class="title">${escapeHtml(t.title)}</span>
    ${tags}
    ${t.completed_at ? '' : `<button data-complete="${t.id}">done</button>`}
    <button data-delete="${t.id}" class="ghost">×</button>
  </li>`;
}

const escapeHtml = (s) => String(s).replace(/[&<>"']/g, (c) =>
  ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

document.addEventListener('click', async (e) => {
  const btn = e.target.closest('button');
  if (!btn) return;

  try {
    if (btn.dataset.list) {
      state.list = btn.dataset.list;
      return refresh();
    }
    if (btn.dataset.complete) {
      await api(`/todos/${btn.dataset.complete}/complete`, { method: 'PATCH' });
      say('Completed.');
      return refresh();
    }
    if (btn.dataset.delete) {
      await api(`/todos/${btn.dataset.delete}`, { method: 'DELETE' });
      say('Deleted.');
      return refresh();
    }
  } catch (err) {
    say(err.message, true);
  }
});

$('new-todo').addEventListener('submit', async (e) => {
  e.preventDefault();
  const body = {
    title: $('title').value,
    list: state.list,
    tags: $('tags').value.split(',').map((s) => s.trim()).filter(Boolean),
    priority: Number($('priority').value),
  };
  try {
    await api('/todos', { method: 'POST', body: JSON.stringify(body) });
    $('title').value = '';
    $('tags').value = '';
    say('Added.');
    refresh();
  } catch (err) {
    say(`Could not add: ${err.message}`, true);
  }
});

refresh();
setInterval(refresh, 15000);
