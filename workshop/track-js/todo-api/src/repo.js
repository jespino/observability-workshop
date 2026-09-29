const { pool } = require('./db.js');

const LIST_PAGE_THRESHOLD = Number(process.env.LIST_PAGE_THRESHOLD ?? 20);

function rowToTodo(row, tags) {
  return {
    id: row.id,
    title: row.title,
    list: row.list,
    slug: row.slug,
    tags,
    priority: row.priority,
    due_at: row.due_at,
    created_at: row.created_at,
    completed_at: row.completed_at,
  };
}

async function listTodos(list) {
  const { rows } = await pool.query(
    `SELECT * FROM todos WHERE list = $1 ORDER BY created_at DESC`,
    [list],
  );

  if (rows.length <= LIST_PAGE_THRESHOLD) {
    const { rows: tagRows } = await pool.query(
      `SELECT todo_id, tag FROM todo_tags WHERE todo_id = ANY($1::uuid[])`,
      [rows.map((r) => r.id)],
    );
    const byTodo = new Map();
    for (const t of tagRows) {
      if (!byTodo.has(t.todo_id)) byTodo.set(t.todo_id, []);
      byTodo.get(t.todo_id).push(t.tag);
    }
    return rows.map((r) => rowToTodo(r, byTodo.get(r.id) ?? []));
  }

  const out = [];
  for (const row of rows) {
    const { rows: tagRows } = await pool.query(
      `SELECT tag FROM todo_tags WHERE todo_id = $1`,
      [row.id],
    );
    out.push(rowToTodo(row, tagRows.map((t) => t.tag)));
  }
  return out;
}

async function getTodo(id) {
  const { rows } = await pool.query(`SELECT * FROM todos WHERE id = $1`, [id]);
  if (rows.length === 0) return null;
  const { rows: tagRows } = await pool.query(
    `SELECT tag FROM todo_tags WHERE todo_id = $1`,
    [id],
  );
  return rowToTodo(rows[0], tagRows.map((t) => t.tag));
}

async function createTodo(payload) {
  const { title, list = 'inbox', tags = [], priority = 2, due_at = null } = payload ?? {};

  const { rows } = await pool.query(
    `INSERT INTO todos (title, list, priority, due_at)
     VALUES ($1, $2, $3, $4) RETURNING *`,
    [title, list, priority, due_at],
  );
  const row = rows[0];

  for (const tag of tags) {
    await pool.query(
      `INSERT INTO todo_tags (todo_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
      [row.id, tag],
    );
  }

  return rowToTodo(row, tags);
}

async function setSlug(id, slug) {
  await pool.query(`UPDATE todos SET slug = $1 WHERE id = $2`, [slug, id]);
}

async function completeTodo(id) {
  const { rows } = await pool.query(
    `UPDATE todos SET completed_at = now() WHERE id = $1 AND completed_at IS NULL RETURNING *`,
    [id],
  );
  if (rows.length === 0) return null;
  const { rows: tagRows } = await pool.query(
    `SELECT tag FROM todo_tags WHERE todo_id = $1`,
    [id],
  );
  return rowToTodo(rows[0], tagRows.map((t) => t.tag));
}

async function deleteTodo(id) {
  const { rowCount } = await pool.query(`DELETE FROM todos WHERE id = $1`, [id]);
  return rowCount > 0;
}

async function listSummaries() {
  const { rows } = await pool.query(
    `SELECT l.name AS list,
            count(t.id) FILTER (WHERE t.completed_at IS NULL) AS open,
            count(t.id) AS total
     FROM lists l LEFT JOIN todos t ON t.list = l.name
     GROUP BY l.name ORDER BY l.name`,
  );
  return rows.map((r) => ({ list: r.list, open: Number(r.open), total: Number(r.total) }));
}

async function overdueCount() {
  const { rows } = await pool.query(
    `SELECT list, count(*) AS count
     FROM todos
     WHERE completed_at IS NULL AND due_at IS NOT NULL AND due_at < now()
     GROUP BY list`,
  );
  return rows.map((r) => ({ list: r.list, count: Number(r.count) }));
}

module.exports = {
  listTodos, getTodo, createTodo, setSlug, completeTodo,
  deleteTodo, listSummaries, overdueCount,
};
