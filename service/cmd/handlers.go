package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/azaurus1/lifevisor-service/internal/data"
)

func (app *Config) UploadBucket(w http.ResponseWriter, r *http.Request) {
	var bucket data.Bucket
	err := json.NewDecoder(r.Body).Decode(&bucket)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error unmarshalling bucket data: %v", err), http.StatusBadRequest)
		return
	}

	err = app.Repo.InsertBucket(bucket)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error inserting bucket: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Bucket uploaded successfully"))
}

func (app *Config) UploadEvent(w http.ResponseWriter, r *http.Request) {
	var event data.Event
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error unmarshalling event data: %v", err), http.StatusBadRequest)
		return
	}

	err = app.Repo.InsertEvent(event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error inserting event: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event uploaded successfully"))
}
