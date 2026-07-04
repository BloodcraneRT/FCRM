// Package githubtrends tracks repo stars/forks over time as a proxy for
// developer adoption in a technology vertical (e.g. AI infra, semis tooling,
// crypto infra). Uses the GitHub REST API; works unauthenticated at 60
// req/hour or authenticated at 5,000 req/hour with a token.
package githubtrends

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// Repo is one GitHub repo to track, tagged with the investment vertical it
// signals for (e.g. "ai-infra", "crypto-infra", "cybersecurity").
type Repo struct {
	OwnerSlashName string
	Vertical       string
}

type Connector struct {
	Token string // optional; raises rate limit from 60/hr to 5000/hr
	Repos []Repo
	http  *http.Client
}

func New(token string, repos []Repo) *Connector {
	return &Connector{Token: token, Repos: repos, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Connector) Name() string { return "github_trends" }

type repoResponse struct {
	StargazersCount int `json:"stargazers_count"`
	ForksCount      int `json:"forks_count"`
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	rows := 0
	var firstErr error
	for _, r := range c.Repos {
		stars, forks, err := c.fetchRepo(ctx, r.OwnerSlashName)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		sql := fmt.Sprintf(
			"INSERT INTO github_trends (ts, repo, vertical, stars, forks, source) VALUES (now(), %s, %s, %d, %d, 'github_api')",
			sqlLit(r.OwnerSlashName), sqlLit(r.Vertical), stars, forks,
		)
		if err := db.Exec(ctx, sql); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		rows++
	}
	return rows, firstErr
}

func (c *Connector) fetchRepo(ctx context.Context, ownerSlashName string) (stars, forks int, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", ownerSlashName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("github_trends: fetch %s: %w", ownerSlashName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("github_trends: %s returned %d", ownerSlashName, resp.StatusCode)
	}

	var parsed repoResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, 0, fmt.Errorf("github_trends: decode %s: %w", ownerSlashName, err)
	}
	return parsed.StargazersCount, parsed.ForksCount, nil
}

func sqlLit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
