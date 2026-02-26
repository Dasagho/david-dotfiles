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

// LatestVersion returns the latest release version for the given repo (owner/name).
// The leading "v" is stripped from the tag name.
func (c *Client) LatestVersion(ctx context.Context, repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseURL, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	// Use GITHUB_TOKEN if available.
	// (No requirement to set it, but respects it if present.)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("github request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case http.StatusNotFound:
		return "", fmt.Errorf("repo %q not found on GitHub — check the repo field in catalog.toml", repo)
	case http.StatusForbidden, http.StatusTooManyRequests:
		return "", fmt.Errorf("GitHub API rate limited for %q — set GITHUB_TOKEN env var to increase limit", repo)
	default:
		return "", fmt.Errorf("unexpected GitHub API status %d for %q", resp.StatusCode, repo)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode GitHub response: %w", err)
	}

	version := strings.TrimPrefix(release.TagName, "v")
	if version == "" {
		return "", fmt.Errorf("empty tag_name in GitHub response for %q", repo)
	}
	return version, nil
}
