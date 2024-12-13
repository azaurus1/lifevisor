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

var initCmd = &cobra.Command{
	Use:   "init [dbtype] [source-path] [connection-string] [concurrency]",
	Short: "Run initial load of data to the specified database type",
	Args:  cobra.ExactArgs(4), // Three arguments: dbtype, source-path, and connection-string
	Run: func(cmd *cobra.Command, args []string) {
		dbType := args[0]
		sourcePath := args[1]
		connString := args[2]

		concurreny, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatal("error converting concurrency to int: ", err)
		}

		// Call the Initialize method
		err = Initialisation(dbType, sourcePath, connString, concurreny)
		if err != nil {
			cmd.PrintErrln("Error during initialisation:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func Initialisation(dbType, sqlitePath, connString string, concurreny int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
