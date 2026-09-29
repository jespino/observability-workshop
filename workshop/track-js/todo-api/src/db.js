const pg = require('pg');

const pool = new pg.Pool({
  connectionString: process.env.DATABASE_URL,
  max: 10,
});

async function waitForDependencies() {
  for (let attempt = 0; attempt < 30; attempt++) {
    try {
      await pool.query('SELECT 1');
      return;
    } catch {
      await new Promise((r) => setTimeout(r, 1000));
    }
  }
  throw new Error('postgres did not become ready');
}

module.exports = { pool, waitForDependencies };
