// Package config centralizes environment-driven configuration for all
// fin-terminal services (API, worker, CLI). Every field has a sane local
// default so `docker compose up` works with zero setup.
package config

import (
	"os"
	"strconv"
)

type Config struct {
	QuestDBHTTPAddr string // REST/SQL endpoint, e.g. http://localhost:9000
	QuestDBILPAddr  string // line-protocol ingest endpoint, e.g. localhost:9009
	QuestDBPGAddr   string // postgres-wire endpoint, e.g. localhost:8812

	SQLitePath string // local app metadata (watchlists, settings, notes index)

	APIAddr string // fin-terminal API server bind address

	// Third-party API keys. All optional; connectors degrade gracefully
	// (skip + log) when a key is missing rather than failing the process.
	GitHubToken        string
	FinnhubAPIKey      string
	AlphaVantageAPIKey string
	FREDAPIKey         string
	NewsAPIKey         string
}

func Load() Config {
	return Config{
		QuestDBHTTPAddr: getEnv("QUESTDB_HTTP_ADDR", "http://localhost:9000"),
		QuestDBILPAddr:  getEnv("QUESTDB_ILP_ADDR", "localhost:9009"),
		QuestDBPGAddr:   getEnv("QUESTDB_PG_ADDR", "localhost:8812"),

		SQLitePath: getEnv("SQLITE_PATH", "./data/fin-terminal.db"),

		APIAddr: getEnv("API_ADDR", ":8081"),

		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		FinnhubAPIKey:      os.Getenv("FINNHUB_API_KEY"),
		AlphaVantageAPIKey: os.Getenv("ALPHAVANTAGE_API_KEY"),
		FREDAPIKey:         os.Getenv("FRED_API_KEY"),
		NewsAPIKey:         os.Getenv("NEWS_API_KEY"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
