// A small in-process cache in front of the list read path.
//
// No network, no extra container: just a Map with time-to-live entries.

const TTL_MS = 30_000;

const store = new Map();

function cacheKey(list) {
  return `todos:summary:${list}:${Date.now()}`;
}

function readSummary(list) {
  const key = cacheKey(list);
  const entry = store.get(key);
  const fresh = entry !== undefined && entry.expires > Date.now();
  return { hit: fresh, key, value: fresh ? entry.value : null };
}

function writeSummary(list, value) {
  const key = cacheKey(list);
  store.set(key, { value, expires: Date.now() + TTL_MS });
  return key;
}

function invalidate(list) {
  const prefix = `todos:summary:${list}:`;
  for (const key of store.keys()) {
    if (key.startsWith(prefix)) store.delete(key);
  }
}

function size() {
  return store.size;
}

module.exports = { cacheKey, readSummary, writeSummary, invalidate, size };
