package data

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
)

func (u *PostgresRepository) RunMigrations() error {
	migrations := &migrate.FileMigrationSource{
		Dir: "./migrations", // Path to migration files
	}

	// Get connection string from the pool config
	connString := u.Conn.Config().ConnString()
	log.Println("Connection string; ", connString)

	// Create a new connection config from the connection string

	dbSQL := stdlib.OpenDBFromPool(u.Conn)
	defer dbSQL.Close()

	// Run migrations using the sql.DB connection
	n, err := migrate.Exec(dbSQL, "postgres", migrations, migrate.Up)
	if err != nil {
		log.Println("Error running migrations: ", err)
		return err
	}

	log.Printf("Applied %d migrations successfully!", n)
	return nil
}

func (u *PostgresRepository) InsertBucket(bucket Bucket) error {
	ctx := context.Background()

	stmt := `insert into bucketmodel (key, id, created, name, type, client, hostname) values ($1, $2, $3, $4, $5, $6, $7) on conflict (key) do nothing`
	_, err := u.Conn.Exec(ctx, stmt, bucket.Key, bucket.ID, bucket.Created, bucket.Name, bucket.Type, bucket.Client, bucket.Hostname)
	if err != nil {
		return err
	}

	return nil

}

func (u *PostgresRepository) InsertEvent(event Event) error {
	ctx := context.Background()

	stmt := `insert into eventmodel (id, bucket_id, timestamp, duration, datastr) values ($1, $2, $3, $4, $5) on conflict (id) do nothing`
	_, err := u.Conn.Exec(ctx, stmt, event.ID, event.BucketID, event.Timestamp, event.Duration, event.DataStr)
	if err != nil {
		return err
	}

	return nil
}
