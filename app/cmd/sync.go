package cmd

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/azaurus1/lifevisor/internal/data"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// this will be the scheduled task

var syncCmd = &cobra.Command{
	Use:   "sync [dbtype] [source-path] [connection-string]",
	Short: "Sync activity watch data by interval",
	Args:  cobra.ExactArgs(4), // Three arguments: dbtype, source-path, and connection-string
	Run: func(cmd *cobra.Command, args []string) {
		dbType := args[0]
		sourcePath := args[1]
		connString := args[2]
		interval, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatal("cannot convert interval to int: ", err)
		}

		// Call the Sync method
		err = Sync(dbType, sourcePath, connString, interval)
		if err != nil {
			cmd.PrintErrln("Error during initialisation:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func Sync(dbType, sourcePath, connString string, interval int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
