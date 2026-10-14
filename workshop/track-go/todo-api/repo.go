package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
	"time"
)

var listPageThreshold = envInt("LIST_PAGE_THRESHOLD", 20)

type Todo struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	List        string     `json:"list"`
	Slug        *string    `json:"slug"`
	Tags        []string   `json:"tags"`
	Priority    int        `json:"priority"`
	DueAt       *time.Time `json:"due_at"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type NewTodo struct {
	Title    string     `json:"title"`
	List     string     `json:"list"`
	Tags     []string   `json:"tags"`
	Priority int        `json:"priority"`
	DueAt    *time.Time `json:"due_at"`
}

type ListSummary struct {
	List  string `json:"list"`
	Open  int    `json:"open"`
	Total int    `json:"total"`
}

type OverdueCount struct {
	List  string
	Count int64
}

const todoColumns = `id, title, list, slug, priority, due_at, created_at, completed_at`

type scanner interface{ Scan(dest ...any) error }

func scanTodo(row scanner) (Todo, error) {
	var t Todo
	err := row.Scan(&t.ID, &t.Title, &t.List, &t.Slug, &t.Priority, &t.DueAt, &t.CreatedAt, &t.CompletedAt)
	t.Tags = []string{}
	return t, err
}

func tagsOf(ctx context.Context, id string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT tag FROM todo_tags WHERE todo_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func listTodos(ctx context.Context, list string) ([]Todo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+todoColumns+` FROM todos WHERE list = $1 ORDER BY created_at DESC`, list)
	if err != nil {
		return nil, err
	}
	todos := []Todo{}
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		todos = append(todos, t)
	}
	rows.Close()

	if len(todos) <= listPageThreshold {
		ids := make([]string, len(todos))
		for i, t := range todos {
			ids[i] = t.ID
		}
		tagRows, err := db.QueryContext(ctx,
			`SELECT todo_id, tag FROM todo_tags WHERE todo_id = ANY($1::uuid[])`, ids)
		if err != nil {
			return nil, err
		}
		defer tagRows.Close()
		byTodo := map[string][]string{}
		for tagRows.Next() {
			var id, tag string
			if err := tagRows.Scan(&id, &tag); err != nil {
				return nil, err
			}
			byTodo[id] = append(byTodo[id], tag)
		}
		for i := range todos {
			if tags, ok := byTodo[todos[i].ID]; ok {
				todos[i].Tags = tags
			}
		}
		return todos, tagRows.Err()
	}

	for i := range todos {
		tags, err := tagsOf(ctx, todos[i].ID)
		if err != nil {
			return nil, err
		}
		todos[i].Tags = tags
	}
	return todos, nil
}

func getTodo(ctx context.Context, id string) (*Todo, error) {
	t, err := scanTodo(db.QueryRowContext(ctx, `SELECT `+todoColumns+` FROM todos WHERE id = $1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if t.Tags, err = tagsOf(ctx, id); err != nil {
		return nil, err
	}
	return &t, nil
}

func createTodo(ctx context.Context, body []byte) (Todo, error) {
	in := NewTodo{List: "inbox", Priority: 2}
	_ = json.Unmarshal(body, &in)
	if in.Tags == nil {
		in.Tags = []string{}
	}

	t, err := scanTodo(db.QueryRowContext(ctx,
		`INSERT INTO todos (title, list, priority, due_at)
		 VALUES ($1, $2, $3, $4) RETURNING `+todoColumns,
		in.Title, in.List, in.Priority, in.DueAt))
	if err != nil {
		return Todo{}, err
	}

	for _, tag := range in.Tags {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO todo_tags (todo_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			t.ID, tag); err != nil {
			return Todo{}, err
		}
	}
	t.Tags = in.Tags
	return t, nil
}

func setSlug(ctx context.Context, id, slug string) error {
	_, err := db.ExecContext(ctx, `UPDATE todos SET slug = $1 WHERE id = $2`, slug, id)
	return err
}

func completeTodo(ctx context.Context, id string) (*Todo, error) {
	t, err := scanTodo(db.QueryRowContext(ctx,
		`UPDATE todos SET completed_at = now() WHERE id = $1 AND completed_at IS NULL
		 RETURNING `+todoColumns, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if t.Tags, err = tagsOf(ctx, id); err != nil {
		return nil, err
	}
	return &t, nil
}

func deleteTodo(ctx context.Context, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func listSummaries(ctx context.Context) ([]ListSummary, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT l.name AS list,
		        count(t.id) FILTER (WHERE t.completed_at IS NULL) AS open,
		        count(t.id) AS total
		 FROM lists l LEFT JOIN todos t ON t.list = l.name
		 GROUP BY l.name ORDER BY l.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ListSummary{}
	for rows.Next() {
		var s ListSummary
		if err := rows.Scan(&s.List, &s.Open, &s.Total); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func overdueCount(ctx context.Context) ([]OverdueCount, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT list, count(*) AS count
		 FROM todos
		 WHERE completed_at IS NULL AND due_at IS NOT NULL AND due_at < now()
		 GROUP BY list`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OverdueCount{}
	for rows.Next() {
		var c OverdueCount
		if err := rows.Scan(&c.List, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func envInt(name string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil {
		return n
	}
	return def
}
