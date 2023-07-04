package main

import (
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var (
	down = flag.Bool("down", false, "run migration down")
)

func main() {
	flag.Parse()
	url := os.Getenv("MONGODB_URL")
	if url == "" {
		url = "mongodb://mongouser:mongopwd@localhost:27017/faceittha?authSource=admin&readPreference=primary&ssl=false&replicaSet=rs0"
	}
	log.Printf("using address: %s", url)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting working-directory: %v", err)
	}
	migrationsDir := "file:///" + wd + "/db/migrations"
	log.Printf("using migrations in dir: %s", migrationsDir)

	m, err := migrate.New(
		migrationsDir,
		url)
	m.Log = &verboseLogger{}

	if err != nil {
		log.Fatalf("New error: %v", err)
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

type verboseLogger struct{}

func (l *verboseLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v)
}

func (l *verboseLogger) Verbose() bool {
	return true
}
