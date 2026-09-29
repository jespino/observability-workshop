// Business logic with no library instrumentation of its own.
// Module 7.2 wraps this in the first custom span.

const PRIORITY_LABELS = { 1: 'high', 2: 'normal', 3: 'low' };

async function enrich(todo) {
  const title = todo.title ?? '';

  const initial = title.trim().at(0).toUpperCase();
  const slug = `${initial}-${title.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').slice(0, 24)}`;

  const overdue =
    todo.due_at !== null &&
    todo.completed_at === null &&
    new Date(todo.due_at).getTime() < Date.now();

  return {
    ...todo,
    slug,
    priority_label: PRIORITY_LABELS[todo.priority] ?? 'normal',
    overdue,
    source: 'enricher',
  };
}

module.exports = { enrich };
