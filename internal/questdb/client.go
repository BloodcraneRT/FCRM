// Package questdb is the only place in the codebase that should know how to
// talk to QuestDB. It wraps the HTTP /exec endpoint for DDL/queries and the
// ILP (InfluxDB Line Protocol) TCP port for high-throughput row ingestion.
// No ORM, no query builder — QuestDB's SQL dialect has enough quirks
// (designated timestamp, SAMPLE BY, LATEST ON) that hiding it behind an
// abstraction would cost more than it saves.
package questdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpAddr string
	http     *http.Client
}

func NewClient(httpAddr string) *Client {
	return &Client{
		httpAddr: strings.TrimRight(httpAddr, "/"),
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// QueryResult mirrors QuestDB's /exec JSON response shape.
type QueryResult struct {
	Query   string          `json:"query"`
	Columns []Column        `json:"columns"`
	Dataset [][]interface{} `json:"dataset"`
	Count   int             `json:"count"`
	Error   string          `json:"error"`
}

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Exec runs DDL or a statement where you don't need the result rows
// (CREATE TABLE, ALTER TABLE, single INSERT, etc).
func (c *Client) Exec(ctx context.Context, sql string) error {
	_, err := c.Query(ctx, sql)
	return err
}

// Query runs any SQL statement against /exec and returns the parsed result.
func (c *Client) Query(ctx context.Context, sql string) (*QueryResult, error) {
	endpoint := fmt.Sprintf("%s/exec?%s", c.httpAddr, url.Values{"query": {sql}}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("questdb: build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("questdb: request failed (is QuestDB running at %s?): %w", c.httpAddr, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("questdb: read response: %w", err)
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("questdb: decode response (status %d): %s", resp.StatusCode, string(body))
	}
	if result.Error != "" {
		return &result, fmt.Errorf("questdb: %s", result.Error)
	}
	return &result, nil
}

// Ping verifies QuestDB is reachable; used by source_health checks and CLI status.
func (c *Client) Ping(ctx context.Context) error {
	return c.Exec(ctx, "SELECT 1")
}
