require('express-async-errors');
const express = require('express');
const { todosRouter } = require('./routes/todos.js');
const { waitForDependencies } = require('./db.js');
const { log } = require('./log.js');

const app = express();
const port = Number(process.env.PORT ?? 8081);

app.use(express.json());

// The frontend is served from a different origin, so allow any origin, method
// and header.
app.use((req, res, next) => {
  res.set('access-control-allow-origin', '*');
  res.set('access-control-allow-headers', '*');
  res.set('access-control-allow-methods', '*');
  res.set('access-control-max-age', '600');
  if (req.method === 'OPTIONS') return res.status(204).end();
  next();
});

app.get('/healthz', (_req, res) => res.json({ ok: true }));

app.use(todosRouter);

app.use((err, _req, res, _next) => {
  log.error('request failed: ' + err.message);
  res.status(500).json({ error: 'internal' });
});

// Exit promptly when asked to stop — which nodemon does on every file save. Once the OpenTelemetry SDK is loaded it registers a SIGTERM handler of
// its own to flush telemetry, and that stops Node from exiting by itself; the
// half second here is the SDK's window to flush.
process.on('SIGTERM', () => setTimeout(() => process.exit(0), 500));

waitForDependencies().then(() => {
  app.listen(port, () => log.info(`todo-api listening on ${port}`));
});
