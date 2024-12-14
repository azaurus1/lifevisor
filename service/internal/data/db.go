package data

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectToDB(dsn string) (*pgxpool.Pool, error) {
	// Establish the connection pool
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		return nil, err
	}

	// Return the connection pool
	return pool, nil
}
