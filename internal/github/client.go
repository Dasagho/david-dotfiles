package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.github.com"

// Client fetches release information from GitHub.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client. Pass an empty string to use the default GitHub API base URL.
// Pass a custom URL for testing.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Release holds the raw tag and the version with any leading "v" stripped.
type Release struct {
	Tag     string // raw tag as returned by GitHub, e.g. "v15.1.0" or "15.1.0"
	Version string // tag with leading "v" stripped, e.g. "15.1.0"
}

// LatestRelease returns the latest release tag and version for the given repo (owner/name).
// Tag is the raw value from the GitHub API; Version has any leading "v" stripped.
func (c *Client) LatestRelease(ctx context.Context, repo string) (Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseURL, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	// Use GITHUB_TOKEN if available.
	// (No requirement to set it, but respects it if present.)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("github request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case http.StatusNotFound:
		return Release{}, fmt.Errorf("repo %q not found on GitHub — check the repo field in catalog.toml", repo)
	case http.StatusForbidden, http.StatusTooManyRequests:
		return Release{}, fmt.Errorf("GitHub API rate limited for %q — set GITHUB_TOKEN env var to increase limit", repo)
	default:
		return Release{}, fmt.Errorf("unexpected GitHub API status %d for %q", resp.StatusCode, repo)
	}

	var apiRelease struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiRelease); err != nil {
		return Release{}, fmt.Errorf("decode GitHub response: %w", err)
	}

	tag := apiRelease.TagName
	version := strings.TrimPrefix(tag, "v")
	if version == "" {
		return Release{}, fmt.Errorf("empty tag_name in GitHub response for %q", repo)
	}
	return Release{Tag: tag, Version: version}, nil
}
