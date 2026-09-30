package yahoo

import (
	"encoding/json"
	"strings"
	"testing"
)

const sample = `{"chart":{"result":[{"timestamp":[1790640000,1790726400,1790812800],
"indicators":{"quote":[{"open":[10,11,null],"high":[12,13,null],"low":[9,10,null],
"close":[11,12,null],"volume":[1000,2000,null]}]}}],"error":null}}`

func TestParseBarsSkipsNulls(t *testing.T) {
	var cr chartResponse
	if err := json.Unmarshal([]byte(sample), &cr); err != nil {
		t.Fatal(err)
	}
	bars, err := parseBars("X", &cr)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 2 {
		t.Fatalf("want 2 bars (null row skipped), got %d", len(bars))
	}
	if bars[1].Close != 12 || bars[1].Volume != 2000 {
		t.Fatalf("unexpected bar %+v", bars[1])
	}
	if bars[0].TS.Hour() != 0 {
		t.Fatalf("ts not normalized to day: %v", bars[0].TS)
	}
}

func TestParseBarsError(t *testing.T) {
	var cr chartResponse
	_ = json.Unmarshal([]byte(`{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found"}}}`), &cr)
	if _, err := parseBars("X", &cr); err == nil {
		t.Fatal("expected error")
	}
}

func TestInsertSQLEscapes(t *testing.T) {
	sql := insertSQL(Instrument{"O'BR", "equity"}, []Bar{{Close: 1}})
	if !strings.Contains(sql, "'O''BR'") {
		t.Fatalf("symbol not escaped: %s", sql)
	}
}
