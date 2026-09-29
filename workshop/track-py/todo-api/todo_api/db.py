import os
import time

import psycopg
from psycopg_pool import ConnectionPool

DATABASE_URL = os.environ["DATABASE_URL"]

pool = ConnectionPool(DATABASE_URL, min_size=1, max_size=10, open=False)


def wait_for_dependencies(attempts: int = 30) -> None:
    for _ in range(attempts):
        try:
            with psycopg.connect(DATABASE_URL, connect_timeout=2) as conn:
                conn.execute("SELECT 1")
            pool.open()
            return
        except Exception:
            time.sleep(1)
    raise RuntimeError("postgres did not become ready")
