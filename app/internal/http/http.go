package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/azaurus1/lifevisor/internal/data"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func HttpInitialisation(ctx context.Context, sqlitePath, url string, concurrency int) error {
	var events []data.Event
	var buckets []data.Bucket

	// 1. get the sqlite db
	sqliteConn, err := sqlite.OpenConn(sqlitePath, sqlite.OpenReadOnly)
	if err != nil {
		return err
	}

	// 2. read everything in the sqlite DB

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

			// add to buckets
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

	// 3. Push to HTTP service

	var bucketCount int

	// Send buckets to HTTP
	for _, bucket := range buckets {
		err := sendToHTTP(url+"/buckets", bucket)
		if err != nil {
			return err
		}
		bucketCount++
	}

	log.Printf("wrote %v buckets to the HTTP service", bucketCount)

	// Use a WaitGroup to handle concurrency for events
	var wg sync.WaitGroup
	eventCh := make(chan data.Event, concurrency)

	// Worker function to process events in parallel
	worker := func(eventCh <-chan data.Event, wg *sync.WaitGroup) {
		defer wg.Done()
		for event := range eventCh {
			err := sendToHTTP(url+"/events", event)
			if err != nil {
				log.Printf("Error uploading event: %v", err)
			}
		}
	}

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(eventCh, &wg)
	}

	// Send events to workers
	for _, event := range events {
		eventCh <- event
	}

	// Close the channel after sending all events
	close(eventCh)

	// Wait for all workers to finish processing
	wg.Wait()

	log.Printf("Successfully loaded %d buckets and %d events to the HTTP service", len(buckets), len(events))
	return nil
}

func HttpSync(ctx context.Context, sourcePath, connString string, interval int) error {
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

	// Push buckets to the HTTP service
	for _, bucket := range buckets {
		err := sendToHTTP(connString+"/buckets", bucket)
		if err != nil {
			log.Printf("Error sending bucket to HTTP service: %v", err)
		}
	}

	// Push events to the HTTP service concurrently
	const workerCount = 1000
	var wg sync.WaitGroup
	eventCh := make(chan data.Event, workerCount)

	worker := func(eventCh <-chan data.Event, wg *sync.WaitGroup) {
		defer wg.Done()
		for event := range eventCh {
			if err := sendToHTTP(connString+"/events", event); err != nil {
				log.Printf("Error sending event to HTTP service: %v", err)
			}
		}
	}

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(eventCh, &wg)
	}

	// Send events to workers
	for _, event := range events {
		eventCh <- event
	}
	close(eventCh)

	wg.Wait()

	log.Printf("Successfully synced %d buckets and %d events to the HTTP service", len(buckets), len(events))
	return nil
}

// Helper function to send data to the HTTP service
func sendToHTTP(endpoint string, data any) error {
	// Marshal data into JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	// Create HTTP POST request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set content-type header
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status: %v", resp.Status)
	}

	return nil
}
