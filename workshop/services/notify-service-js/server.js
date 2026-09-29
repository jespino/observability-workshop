// notify-service — the downstream service, pre-instrumented.
//
// Written in JavaScript so that participants on the Python track see a trace
// cross a language boundary. Participants never edit this file.

import express from 'express';

const app = express();
const port = Number(process.env.PORT ?? 8084);

app.use(express.json());

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function deliver(tags = []) {
  if (tags.includes('urgent')) {
    await sleep(750 + Math.random() * 100);
    return 'pager';
  }
  await sleep(15 + Math.random() * 15);
  return 'digest';
}

app.get('/healthz', (_req, res) => res.json({ ok: true }));

app.post('/notify', async (req, res) => {
  const channel = await deliver(req.body?.tags);
  console.log('notification queued', req.body?.id, 'via', channel);
  res.json({ queued: true, channel });
});

app.post('/stats-ping', async (_req, res) => {
  await sleep(50 + Math.random() * 70);
  res.json({ ok: true });
});

app.listen(port, () => console.log(`notify-service listening on ${port}`));
