package main

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
)

func routes(mux *http.ServeMux) {
	mux.Handle("GET /lists", handler(getLists))
	mux.Handle("GET /todos", handler(getTodos))
	mux.Handle("GET /todos/{id}", handler(getOne))
	mux.Handle("POST /todos", handler(postTodo))
	mux.Handle("PATCH /todos/{id}/complete", handler(patchComplete))
	mux.Handle("DELETE /todos/{id}", handler(deleteOne))
	mux.Handle("GET /stats", handler(getStats))
}

func getLists(w http.ResponseWriter, r *http.Request) error {
	summaries, err := listSummaries(r.Context())
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, summaries)
}

func getTodos(w http.ResponseWriter, r *http.Request) error {
	list := r.URL.Query().Get("list")
	if list == "" {
		list = "inbox"
	}

	hit, _, value := readSummary(list)
	if hit {
		return writeJSON(w, http.StatusOK, value)
	}

	todos, err := listTodos(r.Context(), list)
	if err != nil {
		return err
	}
	writeSummary(list, todos)
	return writeJSON(w, http.StatusOK, todos)
}

func getOne(w http.ResponseWriter, r *http.Request) error {
	todo, err := getTodo(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	if todo == nil {
		return writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
	return writeJSON(w, http.StatusOK, todo)
}

func postTodo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	body, _ := io.ReadAll(r.Body)

	todo, err := createTodo(ctx, body)
	if err != nil {
		return err
	}
	enriched, err := enrich(todo)
	if err != nil {
		return err
	}
	if err := setSlug(ctx, todo.ID, *enriched.Slug); err != nil {
		return err
	}

	if slices.Contains(todo.Tags, "urgent") {
		if _, err := notifyUrgent(ctx, enriched); err != nil {
			return err
		}
	}

	invalidate(todo.List)
	log.Info("todo created " + todo.ID)
	return writeJSON(w, http.StatusCreated, enriched)
}

func patchComplete(w http.ResponseWriter, r *http.Request) error {
	todo, err := completeTodo(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	if todo == nil {
		return writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found or already complete"})
	}
	invalidate(todo.List)
	log.Info("todo completed " + todo.ID)
	return writeJSON(w, http.StatusOK, todo)
}

func deleteOne(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	todo, err := getTodo(ctx, r.PathValue("id"))
	if err != nil {
		return err
	}
	gone, err := deleteTodo(ctx, r.PathValue("id"))
	if err != nil {
		return err
	}
	if !gone {
		return writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
	invalidate(todo.List)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func getStats(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var open, done int
	if err := conn.QueryRowContext(ctx,
		`SELECT count(*) FILTER (WHERE completed_at IS NULL) AS open,
		        count(*) FILTER (WHERE completed_at IS NOT NULL) AS done
		 FROM todos`).Scan(&open, &done); err != nil {
		return err
	}

	summary, err := listSummaries(ctx)
	if err != nil {
		return err
	}
	if _, err := notifyStats(ctx, summary); err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, map[string]any{"open": open, "done": done, "lists": summary})
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
