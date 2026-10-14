package main

// Business logic with no library instrumentation of its own.
// Module 8.2 wraps this in the first custom span.

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var priorityLabels = map[int]string{1: "high", 2: "normal", 3: "low"}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

type Enriched struct {
	Todo
	PriorityLabel string `json:"priority_label"`
	Overdue       bool   `json:"overdue"`
	Source        string `json:"source"`
}

func enrich(todo Todo) (Enriched, error) {
	title := strings.TrimSpace(todo.Title)

	initial, err := firstLetter(title)
	if err != nil {
		return Enriched{}, err
	}
	slug := nonAlnum.ReplaceAllString(strings.ToLower(title), "-")
	if len(slug) > 24 {
		slug = slug[:24]
	}
	slug = initial + "-" + slug

	overdue := todo.DueAt != nil && todo.CompletedAt == nil && todo.DueAt.Before(time.Now())

	label, ok := priorityLabels[todo.Priority]
	if !ok {
		label = "normal"
	}

	todo.Slug = &slug
	return Enriched{Todo: todo, PriorityLabel: label, Overdue: overdue, Source: "enricher"}, nil
}

func firstLetter(title string) (string, error) {
	if title == "" {
		return "", errors.New("cannot take the first letter of an empty title")
	}
	return strings.ToUpper(title[:1]), nil
}
