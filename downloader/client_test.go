package downloader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientReturnsCachedTrack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/tracks" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(response{
			Status:   "ready",
			Title:    "Cantina Band",
			Filename: "Cantina Band [abc123].dca",
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	filename, title, err := client.Download(context.Background(), "cantina band")
	if err != nil {
		t.Fatal(err)
	}
	if filename != "Cantina Band [abc123].dca" || title != "Cantina Band" {
		t.Fatalf("unexpected track: %q, %q", filename, title)
	}
}

func TestClientPollsPendingJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tracks":
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(response{Status: "pending", JobID: "job-1"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/jobs/job-1":
			json.NewEncoder(w).Encode(response{
				Status:   "ready",
				Title:    "Cantina Band",
				Filename: "Cantina Band [abc123].dca",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	client.pollInterval = time.Millisecond

	filename, _, err := client.Download(context.Background(), "cantina band")
	if err != nil {
		t.Fatal(err)
	}
	if filename != "Cantina Band [abc123].dca" {
		t.Fatalf("unexpected filename %q", filename)
	}
}
