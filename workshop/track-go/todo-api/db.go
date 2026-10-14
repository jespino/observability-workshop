package main

import (
	"database/sql"
	"errors"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var db = mustOpen()

func mustOpen() *sql.DB {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(10)
	return db
}

func waitForDependencies() error {
	for attempt := 0; attempt < 30; attempt++ {
		if err := db.Ping(); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("postgres did not become ready")
}
