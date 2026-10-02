package fred

import (
	"strings"
	"testing"
)

func TestParseCSVSkipsGaps(t *testing.T) {
	in := "observation_date,DGS10\n2026-09-26,4.10\n2026-09-27,.\n2026-09-28,4.15\n"
	rows, err := parseCSV(strings.NewReader(in), "DGS10")
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if !strings.Contains(rows[1], "'DGS10',4.15") {
		t.Errorf("bad row %s", rows[1])
	}
}
