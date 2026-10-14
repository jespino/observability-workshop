package main

import (
	"log/slog"
	"os"
)

// The application's logger: JSON lines on stdout.
var log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
