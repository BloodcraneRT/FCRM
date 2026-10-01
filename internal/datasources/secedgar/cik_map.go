package secedgar

// CommonCIKs is a small offline fallback map (verified against SEC
// company_tickers.json on 2026-09-30; prefer ResolveCIKs for live lookups).
// It is a small seed map of ticker -> zero-padded CIK for well-known
// companies, so the platform works out of the box without requiring the user
// to look up CIKs manually. Extend freely, or resolve CIKs dynamically via
// https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&company=... for
// tickers not listed here.
var CommonCIKs = map[string]string{
	"AAPL":  "0000320193",
	"MSFT":  "0000789019",
	"NVDA":  "0001045810",
	"GOOGL": "0001652044",
	"AMZN":  "0001018724",
	"META":  "0001326801",
	"TSLA":  "0001318605",
	"AVAV":  "0001368622",
	"KTOS":  "0001069258",
	"MP":    "0001801368",
	"CRCL":  "0001876042",
	"BE":    "0001664703",
	"GEV":   "0001996810",
	"VST":   "0001692819",
}
