require('express-async-errors');
const express = require('express');
const { todosRouter } = require('./routes/todos.js');
const { waitForDependencies } = require('./db.js');
const { log } = require('./log.js');

const app = express();
const port = Number(process.env.PORT ?? 8081);

app.use(express.json());

// The frontend is served from a different origin. `traceparent` is allowed
// so that the browser-instrumentation bonus in Module 7.4 works.
app.use((req, res, next) => {
  res.set('access-control-allow-origin', '*');
  res.set('access-control-allow-headers', 'content-type,traceparent,tracestate,x-client');
  res.set('access-control-allow-methods', 'GET,POST,PATCH,DELETE,OPTIONS');
  if (req.method === 'OPTIONS') return res.status(204).end();
  next();
});

app.get('/healthz', (_req, res) => res.json({ ok: true }));

app.use(todosRouter);

app.use((err, _req, res, _next) => {
  log.error('request failed: ' + err.message);
  res.status(500).json({ error: 'internal' });
});

waitForDependencies().then(() => {
  app.listen(port, () => log.info(`todo-api listening on ${port}`));
});
