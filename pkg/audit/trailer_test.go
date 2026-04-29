package audit_test

import (
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/audit"
)

func TestRecord_Trailer_RoundTrip(t *testing.T) {
	w := tempWriter(t)
	if err := w.Append(audit.Record{Verb: "test", Timestamp: 1714435200}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	records := read(t, w.Path)
	if len(records) != 1 {
		t.Fatalf("got %d records", len(records))
	}
	rec := records[0]
	if rec.ID == "" {
		t.Fatal("ID should be auto-populated")
	}
	if !strings.HasPrefix(rec.ID, "0") && !strings.Contains(rec.ID, "-") {
		t.Errorf("ID %q does not look like a UUID", rec.ID)
	}
	short := rec.ShortID()
	if len(short) != 8 {
		t.Errorf("ShortID len = %d, want 8 (got %q)", len(short), short)
	}
	trailer := rec.Trailer()
	if trailer == "" {
		t.Fatal("Trailer is empty")
	}
	if !strings.HasPrefix(trailer, "coily://1714435200/") {
		t.Errorf("trailer %q should start with coily://1714435200/", trailer)
	}
	ts, gotShort, ok := audit.ParseTrailer(trailer)
	if !ok {
		t.Fatal("ParseTrailer: not ok")
	}
	if ts != 1714435200 {
		t.Errorf("ts = %d, want 1714435200", ts)
	}
	if gotShort != short {
		t.Errorf("short = %q, want %q", gotShort, short)
	}
}

func TestParseTrailer_Rejects(t *testing.T) {
	cases := []string{
		"",
		"coily://",
		"coily://abc/short",
		"http://1714435200/short",
		"coily://1714435200",
		"coily:///short",
	}
	for _, in := range cases {
		if _, _, ok := audit.ParseTrailer(in); ok {
			t.Errorf("ParseTrailer(%q): got ok=true, want false", in)
		}
	}
}

func TestRecord_TrailerEmptyWhenNoID(t *testing.T) {
	r := audit.Record{Timestamp: 100}
	if r.Trailer() != "" {
		t.Errorf("Trailer with empty ID = %q, want empty", r.Trailer())
	}
}
