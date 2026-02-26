package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/dsaleh/david-dotfiles/internal/github"
)

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name": "v1.2.3"}`))
	}))
	defer srv.Close()

	client := gh.NewClient(srv.URL)
	rel, err := client.LatestRelease(context.Background(), "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.Tag != "v1.2.3" {
		t.Errorf("expected tag v1.2.3, got %s", rel.Tag)
	}
	if rel.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", rel.Version)
	}
}

// Some repos (e.g. ripgrep) use a bare version tag without a "v" prefix.
func TestLatestRelease_bareTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name": "15.1.0"}`))
	}))
	defer srv.Close()

	client := gh.NewClient(srv.URL)
	rel, err := client.LatestRelease(context.Background(), "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.Tag != "15.1.0" {
		t.Errorf("expected tag 15.1.0, got %s", rel.Tag)
	}
	if rel.Version != "15.1.0" {
		t.Errorf("expected version 15.1.0, got %s", rel.Version)
	}
}

func TestLatestRelease_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := gh.NewClient(srv.URL)
	_, err := client.LatestRelease(context.Background(), "owner/repo")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestLatestRelease_rateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := gh.NewClient(srv.URL)
	_, err := client.LatestRelease(context.Background(), "owner/repo")
	if err == nil {
		t.Fatal("expected error for 403")
	}
}
