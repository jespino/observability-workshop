const express = require('express');
const { pool } = require('../db.js');
const {
  listTodos, getTodo, createTodo, completeTodo, deleteTodo,
  listSummaries, setSlug,
} = require('../repo.js');
const { readSummary, writeSummary, invalidate } = require('../cache.js');
const { notifyUrgent, notifyStats } = require('../notify.js');
const { enrich } = require('../enrich.js');
const { log } = require('../log.js');

const todosRouter = express.Router();

todosRouter.get('/lists', async (_req, res) => {
  const summaries = await listSummaries();
  res.json(summaries);
});

todosRouter.get('/todos', async (req, res) => {
  const list = req.query.list ?? 'inbox';

  const cached = readSummary(list);
  if (cached.hit) {
    return res.json(cached.value);
  }

  const todos = await listTodos(list);
  writeSummary(list, todos);
  res.json(todos);
});

todosRouter.get('/todos/:id', async (req, res) => {
  const todo = await getTodo(req.params.id);
  if (!todo) return res.status(404).json({ error: 'not found' });
  res.json(todo);
});

todosRouter.post('/todos', async (req, res) => {
  const todo = await createTodo(req.body);
  const enriched = await enrich(todo);
  await setSlug(todo.id, enriched.slug);

  if (todo.tags.includes('urgent')) {
    await notifyUrgent(enriched);
  }

  invalidate(todo.list);
  log.info('todo created ' + todo.id);
  res.status(201).json(enriched);
});

todosRouter.patch('/todos/:id/complete', async (req, res) => {
  const todo = await completeTodo(req.params.id);
  if (!todo) return res.status(404).json({ error: 'not found or already complete' });
  invalidate(todo.list);
  log.info('todo completed ' + todo.id);
  res.json(todo);
});

todosRouter.delete('/todos/:id', async (req, res) => {
  const todo = await getTodo(req.params.id);
  const gone = await deleteTodo(req.params.id);
  if (!gone) return res.status(404).json({ error: 'not found' });
  invalidate(todo.list);
  res.status(204).end();
});

todosRouter.get('/stats', async (_req, res) => {
  const client = await pool.connect();
  try {
    const { rows } = await client.query(
      `SELECT count(*) FILTER (WHERE completed_at IS NULL) AS open,
              count(*) FILTER (WHERE completed_at IS NOT NULL) AS done
       FROM todos`,
    );

    const summary = await listSummaries();
    await notifyStats(summary);

    res.json({ open: Number(rows[0].open), done: Number(rows[0].done), lists: summary });
  } finally {
    client.release();
  }
});

module.exports = { todosRouter };
