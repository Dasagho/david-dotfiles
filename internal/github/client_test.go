package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/dsaleh/david-dotfiles/internal/github"
)

func TestLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name": "v1.2.3"}`))
	}))
	defer srv.Close()

	client := gh.NewClient(srv.URL)
	version, err := client.LatestVersion(context.Background(), "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "1.2.3" {
		t.Errorf("expected 1.2.3, got %s", version)
	}
}

func TestLatestVersion_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := gh.NewClient(srv.URL)
	_, err := client.LatestVersion(context.Background(), "owner/repo")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestLatestVersion_rateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := gh.NewClient(srv.URL)
	_, err := client.LatestVersion(context.Background(), "owner/repo")
	if err == nil {
		t.Fatal("expected error for 403")
	}
}
