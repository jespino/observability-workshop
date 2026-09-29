from flask import Blueprint, jsonify, request
from psycopg.rows import dict_row

from todo_api.cache import invalidate, read_summary, write_summary
from todo_api.db import pool
from todo_api.enrich import enrich
from todo_api.log import log
from todo_api.notify import notify_stats, notify_urgent
from todo_api.repo import (
    complete_todo, create_todo, delete_todo, get_todo,
    list_summaries, list_todos, set_slug,
)

bp = Blueprint("todos", __name__)


@bp.get("/lists")
def get_lists():
    return jsonify(list_summaries())


@bp.get("/todos")
def get_todos():
    list_name = request.args.get("list", "inbox")

    hit, _key, value = read_summary(list_name)
    if hit:
        return jsonify(value)

    todos = [t.to_dict() for t in list_todos(list_name)]
    write_summary(list_name, todos)
    return jsonify(todos)


@bp.get("/todos/<todo_id>")
def get_one(todo_id):
    todo = get_todo(todo_id)
    if todo is None:
        return jsonify(error="not found"), 404
    return jsonify(todo.to_dict())


@bp.post("/todos")
def post_todo():
    todo = create_todo(request.get_json(silent=True))
    enriched = enrich(todo)
    set_slug(todo.id, enriched["slug"])

    if "urgent" in todo.tags:
        notify_urgent(enriched)

    invalidate(todo.list)
    log.info("todo created %s", todo.id)
    return jsonify(enriched), 201


@bp.patch("/todos/<todo_id>/complete")
def patch_complete(todo_id):
    todo = complete_todo(todo_id)
    if todo is None:
        return jsonify(error="not found or already complete"), 404
    invalidate(todo.list)
    log.info("todo completed %s", todo.id)
    return jsonify(todo.to_dict())


@bp.delete("/todos/<todo_id>")
def delete_one(todo_id):
    todo = get_todo(todo_id)
    if not delete_todo(todo_id):
        return jsonify(error="not found"), 404
    invalidate(todo.list)
    return "", 204


@bp.get("/stats")
def get_stats():
    with pool.connection() as conn:
        conn.row_factory = dict_row
        row = conn.execute(
            """SELECT count(*) FILTER (WHERE completed_at IS NULL) AS open,
                      count(*) FILTER (WHERE completed_at IS NOT NULL) AS done
               FROM todos"""
        ).fetchone()

        summary = list_summaries()
        notify_stats(summary)

        return jsonify(open=int(row["open"]), done=int(row["done"]), lists=summary)
