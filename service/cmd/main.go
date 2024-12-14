package main

import (
	"log"
	"net/http"
	"os"

	"github.com/azaurus1/lifevisor-service/internal/data"
)

type Config struct {
	Server *http.Server
	Repo   data.Repository
}

func main() {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		log.Fatal("DSN environment variable is not set")
	}

	app := Config{}

	conn, err := data.ConnectToDB(dsn)
	if err != nil {
		log.Fatal("could not connect to db: ", err)
	}

	dataRepo := data.NewPostgresRepository(conn)
	app.Repo = dataRepo

	// run migrations
	log.Println("Running data migrations...")
	app.Repo.RunMigrations()

	http.HandleFunc("/buckets", app.UploadBucket)
	http.HandleFunc("/events", app.UploadEvent)

	app.Server = &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	log.Println("Server starting on :8080")
	if err := app.Server.ListenAndServe(); err != nil {
		log.Fatal("Server failed: ", err)
	}
}
