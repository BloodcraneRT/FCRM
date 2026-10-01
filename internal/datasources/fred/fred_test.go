package fred

import (
	"strings"
	"testing"
)

func TestParseCSVSkipsMissing(t *testing.T) {
	in := "observation_date,DGS10\n2026-09-25,5.17\n2026-09-26,.\n2026-09-29,5.26\n"
	obs, err := ParseCSV(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(obs) != 2 || obs[1].Value != 5.26 || obs[1].TS.Day() != 29 {
		t.Fatalf("unexpected obs: %+v", obs)
	}
}
