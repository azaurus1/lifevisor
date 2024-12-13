package data

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	RunMigrations() error
	InsertBucket(bucket Bucket) error
	InsertEvent(event Event) error
}

var repo Repository

type PostgresRepository struct {
	Conn *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		Conn: pool,
	}
}
