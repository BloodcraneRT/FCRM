package stooq

import (
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	in := "Date,Open,High,Low,Close,Volume\n2026-09-29,100,105,99,104.5,12345\n2026-09-30,104,106,103,105,999\n"
	rows, err := parseCSV(strings.NewReader(in), "NVDA", "equity")
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if !strings.Contains(rows[0], "'NVDA'") || !strings.Contains(rows[0], "104.5,12345") {
		t.Errorf("bad row: %s", rows[0])
	}
}

func TestParseCSVNoData(t *testing.T) {
	if _, err := parseCSV(strings.NewReader("No data"), "X", "equity"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSymbolMapping(t *testing.T) {
	if stooqSymbol("NVDA") != "nvda.us" || stooqSymbol("^SPX") != "^spx" || assetClass("^SPX") != "index" {
		t.Fatal("mapping wrong")
	}
}
