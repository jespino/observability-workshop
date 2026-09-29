import os
from dataclasses import dataclass, field
from datetime import datetime

from psycopg.rows import dict_row

from todo_api.db import pool

LIST_PAGE_THRESHOLD = int(os.environ.get("LIST_PAGE_THRESHOLD", "20"))


@dataclass
class Todo:
    id: str
    title: str
    list: str
    slug: str | None
    tags: list[str] = field(default_factory=list)
    priority: int = 2
    due_at: datetime | None = None
    created_at: datetime | None = None
    completed_at: datetime | None = None

    def to_dict(self) -> dict:
        return {
            "id": str(self.id),
            "title": self.title,
            "list": self.list,
            "slug": self.slug,
            "tags": self.tags,
            "priority": self.priority,
            "due_at": self.due_at.isoformat() if self.due_at else None,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }


def _row_to_todo(row: dict, tags: list[str]) -> Todo:
    return Todo(
        id=row["id"], title=row["title"], list=row["list"], slug=row["slug"],
        tags=tags, priority=row["priority"], due_at=row["due_at"],
        created_at=row["created_at"], completed_at=row["completed_at"],
    )


def list_todos(list_name: str) -> list[Todo]:
    with pool.connection() as conn:
        conn.row_factory = dict_row
        rows = conn.execute(
            "SELECT * FROM todos WHERE list = %s ORDER BY created_at DESC",
            (list_name,),
        ).fetchall()

        if len(rows) <= LIST_PAGE_THRESHOLD:
            tag_rows = conn.execute(
                "SELECT todo_id, tag FROM todo_tags WHERE todo_id = ANY(%s)",
                ([r["id"] for r in rows],),
            ).fetchall()
            by_todo: dict[str, list[str]] = {}
            for t in tag_rows:
                by_todo.setdefault(t["todo_id"], []).append(t["tag"])
            return [_row_to_todo(r, by_todo.get(r["id"], [])) for r in rows]

        out = []
        for row in rows:
            tag_rows = conn.execute(
                "SELECT tag FROM todo_tags WHERE todo_id = %s", (row["id"],)
            ).fetchall()
            out.append(_row_to_todo(row, [t["tag"] for t in tag_rows]))
        return out


def get_todo(todo_id: str) -> Todo | None:
    with pool.connection() as conn:
        conn.row_factory = dict_row
        row = conn.execute("SELECT * FROM todos WHERE id = %s", (todo_id,)).fetchone()
        if row is None:
            return None
        tag_rows = conn.execute(
            "SELECT tag FROM todo_tags WHERE todo_id = %s", (todo_id,)
        ).fetchall()
        return _row_to_todo(row, [t["tag"] for t in tag_rows])


def create_todo(payload: dict | None) -> Todo:
    payload = payload or {}
    title = payload.get("title")
    list_name = payload.get("list", "inbox")
    tags = payload.get("tags", [])
    priority = payload.get("priority", 2)
    due_at = payload.get("due_at")

    with pool.connection() as conn:
        conn.row_factory = dict_row
        row = conn.execute(
            """INSERT INTO todos (title, list, priority, due_at)
               VALUES (%s, %s, %s, %s) RETURNING *""",
            (title, list_name, priority, due_at),
        ).fetchone()

        for tag in tags:
            conn.execute(
                "INSERT INTO todo_tags (todo_id, tag) VALUES (%s, %s) ON CONFLICT DO NOTHING",
                (row["id"], tag),
            )

    return _row_to_todo(row, tags)


def set_slug(todo_id: str, slug: str) -> None:
    with pool.connection() as conn:
        conn.execute("UPDATE todos SET slug = %s WHERE id = %s", (slug, todo_id))


def complete_todo(todo_id: str) -> Todo | None:
    with pool.connection() as conn:
        conn.row_factory = dict_row
        row = conn.execute(
            """UPDATE todos SET completed_at = now()
               WHERE id = %s AND completed_at IS NULL RETURNING *""",
            (todo_id,),
        ).fetchone()
        if row is None:
            return None
        tag_rows = conn.execute(
            "SELECT tag FROM todo_tags WHERE todo_id = %s", (todo_id,)
        ).fetchall()
        return _row_to_todo(row, [t["tag"] for t in tag_rows])


def delete_todo(todo_id: str) -> bool:
    with pool.connection() as conn:
        cur = conn.execute("DELETE FROM todos WHERE id = %s", (todo_id,))
        return cur.rowcount > 0


def list_summaries() -> list[dict]:
    with pool.connection() as conn:
        conn.row_factory = dict_row
        rows = conn.execute(
            """SELECT l.name AS list,
                      count(t.id) FILTER (WHERE t.completed_at IS NULL) AS open,
                      count(t.id) AS total
               FROM lists l LEFT JOIN todos t ON t.list = l.name
               GROUP BY l.name ORDER BY l.name"""
        ).fetchall()
        return [{"list": r["list"], "open": int(r["open"]), "total": int(r["total"])} for r in rows]


def overdue_count() -> list[dict]:
    with pool.connection() as conn:
        conn.row_factory = dict_row
        rows = conn.execute(
            """SELECT list, count(*) AS count
               FROM todos
               WHERE completed_at IS NULL AND due_at IS NOT NULL AND due_at < now()
               GROUP BY list"""
        ).fetchall()
        return [{"list": r["list"], "count": int(r["count"])} for r in rows]
