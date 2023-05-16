package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var (
	down                  = flag.Bool("down", false, "run migration down")
)

func main() {
	flag.Parse()
	url := os.Getenv("POSTGRESQL_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	db, err := sql.Open("postgres", url)
	defer db.Close()
	if err != nil {
		log.Fatalf("error opening db connection: %v", err)
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("error invoking wihtInstance: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting working-directory: %v", err)
	}
	migrationsDir := "file:///" + wd + "/db/migrations"
	log.Printf("using migrations in dir: %s", migrationsDir)
	m, err := migrate.NewWithDatabaseInstance(
		migrationsDir,
		"postgres", driver)
	if err != nil {
		log.Fatalf("NewWithDatabaseInstance error: %v", err)
	}
	if *down {
		if err := m.Down(); err != nil {
			log.Fatalf("error migrating down: %v", err)
		}
	} else {
		if err := m.Up(); err != nil {
			log.Fatalf("error migrating up: %v", err)
		}
	}
}
