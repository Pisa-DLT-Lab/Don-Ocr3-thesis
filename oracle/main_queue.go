//go:build queue

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

// This is a "Lite" Queue, that needs only to test the Attribution Oracle
// It distributes always the same job and accepts every result

func main() {
	addr := flag.String("queue_addr", ":20000", "Address to listen on")
	flag.Parse()

	http.HandleFunc("/requests/current", func(w http.ResponseWriter, r *http.Request) {
		// Create a mock job
		job := queryPayload{
			RequestID: fmt.Sprintf("req_%d", time.Now().Unix()),
			Statement: "Explain model X using Attribution Method",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	})

	http.HandleFunc("/requests/", func(w http.ResponseWriter, r *http.Request) {
		// Accept observations and results without doing nothing(mock)
		// URL like: /requests/{id}/observations o /requests/{id}/result
		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("QUEUE LITE STARTED on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}