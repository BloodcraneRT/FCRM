package questdb

import (
	"bufio"
	"net"
	"testing"
	"time"
)

func TestILPWriter_RowFormat(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		line, _ := bufio.NewReader(conn).ReadString('\n')
		received <- line
	}()

	w, err := DialILP(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	ts := time.Unix(1700000000, 0).UTC()
	if err := w.Row("market_bars",
		map[string]string{"symbol": "AAPL"},
		map[string]interface{}{"close": 190.5, "volume": int64(1000)},
		ts,
	); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case line := <-received:
		want := "market_bars,symbol=AAPL close=190.5,volume=1000i 1700000000000000000\n"
		if line != want {
			t.Fatalf("got  %q\nwant %q", line, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ILP line")
	}
}

func TestILPWriter_RejectsRowWithNoFields(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	w, err := DialILP(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := w.Row("market_bars", nil, nil, time.Now()); err == nil {
		t.Fatal("expected error for row with no fields")
	}
}
