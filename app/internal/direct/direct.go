package direct

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/azaurus1/lifevisor/internal/data"
	"github.com/jackc/pgx/v5/pgxpool"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// This is for using a DSN and directly uploading to the DB
func DirectInitialisation(ctx context.Context, dbType, sqlitePath, connString string, concurreny int) error {
	var db data.Repository
	var events []data.Event
	var buckets []data.Bucket

	// 1. get the sqlite db
	sqliteConn, err := sqlite.OpenConn(sqlitePath, sqlite.OpenReadOnly)
	if err != nil {
		return err
	}
	// 2. get the db conn
	if dbType == "pg" {
		pgConn, err := pgxpool.New(ctx, connString)
		if err != nil {
			return err
		}
		db = data.NewPostgresRepository(pgConn)
	}

	// 3. run migrations
	err = db.RunMigrations()
	if err != nil {
		return err
	}

	// 4. read everything in the sqlite DB

	// buckets first

	err = sqlitex.ExecuteTransient(sqliteConn, "SELECT * from bucketmodel;", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			// create new bucket
			var bucket data.Bucket

			layout := "2006-01-02T15:04:05.999999"
			parsedTime, err := time.Parse(layout, stmt.ColumnText(2))
			if err != nil {
				log.Println(stmt.ColumnText(2))
				log.Println("error in bucket time")
				return err
			}

			bucket.Key = stmt.ColumnInt(0)
			bucket.ID = stmt.ColumnText(1)
			bucket.Created = parsedTime
			bucket.Name = stmt.ColumnText(3)
			bucket.Type = stmt.ColumnText(4)
			bucket.Client = stmt.ColumnText(5)
			bucket.Hostname = stmt.ColumnText(6)

			// add to events
			buckets = append(buckets, bucket)

			return nil
		},
	})
	if err != nil {
		return err
	}

	// then events

	err = sqlitex.ExecuteTransient(sqliteConn, "SELECT * from eventmodel;", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			// create new event
			var event data.Event

			layout := "2006-01-02 15:04:05.999999-07:00"
			parsedTime, err := time.Parse(layout, stmt.ColumnText(2))
			if err != nil {
				log.Println("error in event time")
				return err
			}

			event.ID = stmt.ColumnInt(0)
			event.BucketID = stmt.ColumnInt(1)
			event.Timestamp = parsedTime
			event.Duration = stmt.ColumnFloat(3)
			event.DataStr = stmt.ColumnText(4)

			// add to events
			events = append(events, event)

			return nil
		},
	})
	if err != nil {
		return err
	}

	// 5. push to db
	var bucketCount int

	for _, bucket := range buckets {
		db.InsertBucket(bucket)
		bucketCount++
	}

	log.Printf("wrote %v buckets to db", bucketCount)

	var wg sync.WaitGroup
	eventCh := make(chan data.Event, concurreny)

	// Worker function
	worker := func(eventCh <-chan data.Event, wg *sync.WaitGroup) {
		defer wg.Done()
		for event := range eventCh {
			db.InsertEvent(event) // Process event
		}
	}

	// Start workers
	for i := 0; i < concurreny; i++ {
		wg.Add(1)
		go worker(eventCh, &wg)
	}

	// Send events to workers
	for _, event := range events {
		eventCh <- event
	}

	log.Printf("Successfully loaded %d buckets and %d events to the remote database", len(buckets), len(events))
	return nil
}

func DirectSync(ctx context.Context, dbType, sourcePath, connString string, interval int) error {
	var db data.Repository
	var events []data.Event
	var buckets []data.Bucket

	// Normalize to UTC for consistency
	currentTime := time.Now().UTC()
	cutoffTime := currentTime.Add(-time.Duration(interval) * time.Second)

	// Open the SQLite connection
	sqliteConn, err := sqlite.OpenConn(sourcePath, sqlite.OpenReadOnly)
	if err != nil {
		return err
	}
	defer sqliteConn.Close()

	// Connect to the remote database
	if dbType == "pg" {
		pgConn, err := pgxpool.New(ctx, connString)
		if err != nil {
			return err
		}
		defer pgConn.Close()
		db = data.NewPostgresRepository(pgConn)
	}

	// Read buckets from SQLite database
	err = sqlitex.ExecuteTransient(sqliteConn, "SELECT * FROM bucketmodel WHERE created >= ?;", &sqlitex.ExecOptions{
		Args: []interface{}{cutoffTime.Format("2006-01-02T15:04:05.999999")},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			var bucket data.Bucket

			parsedTime, err := time.Parse("2006-01-02T15:04:05.999999", stmt.ColumnText(2))
			if err != nil {
				log.Println("Error parsing bucket time:", stmt.ColumnText(2))
				return err
			}

			bucket.Key = stmt.ColumnInt(0)
			bucket.ID = stmt.ColumnText(1)
			bucket.Created = parsedTime
			bucket.Name = stmt.ColumnText(3)
			bucket.Type = stmt.ColumnText(4)
			bucket.Client = stmt.ColumnText(5)
			bucket.Hostname = stmt.ColumnText(6)

			buckets = append(buckets, bucket)
			return nil
		},
	})
	if err != nil {
		return err
	}

	// Read events from SQLite database
	err = sqlitex.ExecuteTransient(sqliteConn, "SELECT * FROM eventmodel WHERE timestamp >= ?;", &sqlitex.ExecOptions{
		Args: []interface{}{cutoffTime.Format("2006-01-02 15:04:05.999999-07:00")},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			var event data.Event

			parsedTime, err := time.Parse("2006-01-02 15:04:05.999999-07:00", stmt.ColumnText(2))
			if err != nil {
				log.Println("Error parsing event time")
				return err
			}

			event.ID = stmt.ColumnInt(0)
			event.BucketID = stmt.ColumnInt(1)
			event.Timestamp = parsedTime
			event.Duration = stmt.ColumnFloat(3)
			event.DataStr = stmt.ColumnText(4)

			events = append(events, event)
			return nil
		},
	})
	if err != nil {
		return err
	}

	// Push buckets to the remote database
	for _, bucket := range buckets {
		err := db.InsertBucket(bucket)
		if err != nil {
			log.Printf("Error inserting bucket: %v", err)
		}
	}

	// Push events to the remote database concurrently
	const workerCount = 1000
	var wg sync.WaitGroup
	eventCh := make(chan data.Event, workerCount)

	worker := func(eventCh <-chan data.Event, wg *sync.WaitGroup) {
		defer wg.Done()
		for event := range eventCh {
			if err := db.InsertEvent(event); err != nil {
				log.Printf("Error inserting event: %v", err)
			}
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(eventCh, &wg)
	}

	for _, event := range events {
		eventCh <- event
	}
	close(eventCh)

	wg.Wait()

	log.Printf("Successfully synced %d buckets and %d events to the remote database", len(buckets), len(events))
	return nil
}
