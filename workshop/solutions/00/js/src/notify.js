const NOTIFY_URL = process.env.NOTIFY_URL ?? 'http://notify-py:8083';
const TIMEOUT_MS = Number(process.env.NOTIFY_TIMEOUT_MS ?? 2000);

async function post(path, body) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  try {
    const res = await fetch(`${NOTIFY_URL}${path}`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(body),
      signal: controller.signal,
    });
    return await res.json();
  } finally {
    clearTimeout(timer);
  }
}

function notifyUrgent(todo) {
  return post('/notify', {
    id: todo.id,
    list: todo.list,
    title: todo.title,
    tags: todo.tags,
  });
}

function notifyStats(summary) {
  return post('/stats-ping', { lists: summary.length });
}

module.exports = { notifyUrgent, notifyStats };
