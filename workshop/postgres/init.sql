-- Workshop schema and seed data.
--
-- Deliberately small: six lists, a few hundred todos, one list that is much
-- bigger than the others.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE lists (
    name TEXT PRIMARY KEY
);

CREATE TABLE todos (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT        NOT NULL,
    list         TEXT        NOT NULL REFERENCES lists(name),
    slug         TEXT,
    priority     INT         NOT NULL DEFAULT 2,
    due_at       TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE todo_tags (
    todo_id UUID NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (todo_id, tag)
);

CREATE INDEX todos_list_idx ON todos (list);
CREATE INDEX todos_due_idx  ON todos (due_at) WHERE completed_at IS NULL;

INSERT INTO lists (name) VALUES
    ('work'), ('home'), ('reading'), ('shopping'), ('someday'), ('inbox');

-- 'work' is deliberately much bigger than the other lists. Most requests read a
-- small list; a minority read the big one. Keep it that way.
INSERT INTO todos (title, list, slug, priority, due_at, created_at, completed_at)
SELECT
    'Work item ' || i,
    'work',
    'W' || i,
    1 + (i % 3),
    now() + make_interval(hours => (i % 96) - 36),
    now() - make_interval(hours => (i % 240)),
    CASE WHEN i % 4 = 0 THEN now() - make_interval(hours => (i % 50)) ELSE NULL END
FROM generate_series(1, 140) AS i;

INSERT INTO todos (title, list, slug, priority, due_at, created_at, completed_at)
SELECT
    initcap(l.name) || ' item ' || i,
    l.name,
    upper(left(l.name, 1)) || i,
    1 + (i % 3),
    now() + make_interval(hours => (i * 7 % 96) - 36),
    now() - make_interval(hours => (i * 11 % 240)),
    CASE WHEN i % 4 = 0 THEN now() - make_interval(hours => (i % 50)) ELSE NULL END
FROM (SELECT name FROM lists WHERE name <> 'work') AS l
CROSS JOIN generate_series(1, 12) AS i;

INSERT INTO todo_tags (todo_id, tag)
SELECT id, 'urgent' FROM todos WHERE priority = 1 AND completed_at IS NULL;

INSERT INTO todo_tags (todo_id, tag)
SELECT id, 'infra' FROM todos WHERE list = 'work' AND (extract(epoch FROM created_at)::bigint % 3) = 0;

INSERT INTO todo_tags (todo_id, tag)
SELECT id, 'later' FROM todos WHERE list IN ('someday', 'reading');
