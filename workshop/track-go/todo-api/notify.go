package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

var notifyURL = envString("NOTIFY_URL", "http://notify-service:8083")

var notifyClient = &http.Client{
	Timeout: time.Duration(envInt("NOTIFY_TIMEOUT_MS", 2000)) * time.Millisecond,
}

func post(ctx context.Context, path string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, notifyURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")

	res, err := notifyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	out := map[string]any{}
	err = json.NewDecoder(res.Body).Decode(&out)
	return out, err
}

func notifyUrgent(ctx context.Context, todo Enriched) (map[string]any, error) {
	return post(ctx, "/notify", map[string]any{
		"id":    todo.ID,
		"list":  todo.List,
		"title": todo.Title,
		"tags":  todo.Tags,
	})
}

func notifyStats(ctx context.Context, summary []ListSummary) (map[string]any, error) {
	return post(ctx, "/stats-ping", map[string]any{"lists": len(summary)})
}

func envString(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
