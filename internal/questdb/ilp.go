package questdb

import (
	"bufio"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ILPWriter is a minimal InfluxDB Line Protocol writer for QuestDB's
// tcp ingest port (default 9009). It's the fast path for bar/tick/event
// ingestion — much higher throughput than issuing INSERT over /exec.
//
// Not safe for concurrent use; give each ingestion worker its own writer.
type ILPWriter struct {
	conn net.Conn
	w    *bufio.Writer
}

func DialILP(addr string) (*ILPWriter, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("questdb: dial ilp %s: %w", addr, err)
	}
	return &ILPWriter{conn: conn, w: bufio.NewWriter(conn)}, nil
}

func (w *ILPWriter) Close() error {
	if err := w.Flush(); err != nil {
		w.conn.Close()
		return err
	}
	return w.conn.Close()
}

func (w *ILPWriter) Flush() error {
	return w.w.Flush()
}

// Row builds and writes a single ILP line:
//
//	table,tag1=v1,tag2=v2 field1=1.0,field2="s" 1690000000000000000
//
// tags are indexed symbol columns (strings only), fields are typed columns.
// ts of zero value uses the server's current time.
func (w *ILPWriter) Row(table string, tags map[string]string, fields map[string]interface{}, ts time.Time) error {
	var b strings.Builder
	b.WriteString(escapeMeasurement(table))

	for _, k := range sortedKeys(tags) {
		v := tags[k]
		if v == "" {
			continue
		}
		b.WriteByte(',')
		b.WriteString(escapeKey(k))
		b.WriteByte('=')
		b.WriteString(escapeTagValue(v))
	}

	b.WriteByte(' ')
	first := true
	for _, k := range sortedFieldKeys(fields) {
		v := fields[k]
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(escapeKey(k))
		b.WriteByte('=')
		b.WriteString(formatFieldValue(v))
	}
	if first {
		return fmt.Errorf("questdb: row for %q has no fields", table)
	}

	if !ts.IsZero() {
		b.WriteByte(' ')
		b.WriteString(strconv.FormatInt(ts.UnixNano(), 10))
	}
	b.WriteByte('\n')

	_, err := w.w.WriteString(b.String())
	return err
}

func formatFieldValue(v interface{}) string {
	switch t := v.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 32)
	case int:
		return strconv.FormatInt(int64(t), 10) + "i"
	case int64:
		return strconv.FormatInt(t, 10) + "i"
	case bool:
		if t {
			return "t"
		}
		return "f"
	case string:
		return `"` + strings.ReplaceAll(t, `"`, `\"`) + `"`
	default:
		return fmt.Sprintf("%q", fmt.Sprintf("%v", t))
	}
}

func escapeMeasurement(s string) string {
	return strings.NewReplacer(",", "\\,", " ", "\\ ").Replace(s)
}

func escapeKey(s string) string {
	return strings.NewReplacer(",", "\\,", " ", "\\ ", "=", "\\=").Replace(s)
}

func escapeTagValue(s string) string {
	return escapeKey(s)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedFieldKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
