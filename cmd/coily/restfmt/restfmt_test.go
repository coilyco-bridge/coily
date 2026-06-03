package restfmt

import (
	"strings"
	"testing"
)

func TestParseOutput(t *testing.T) {
	cases := []struct {
		in      string
		want    Output
		ok      bool
		wantErr bool
	}{
		{"", "", false, false},
		{"json", OutputJSON, true, false},
		{"yaml", OutputYAML, true, false},
		{"text", OutputText, true, false},
		{"table", "", false, true},
		{"xml", "", false, true},
	}
	for _, c := range cases {
		got, ok, err := ParseOutput(c.in)
		if (err != nil) != c.wantErr || ok != c.ok || got != c.want {
			t.Errorf("ParseOutput(%q) = (%q,%v,%v), want (%q,%v,err=%v)", c.in, got, ok, err, c.want, c.ok, c.wantErr)
		}
	}
}

func TestNeedsRender(t *testing.T) {
	if NeedsRender("", false) {
		t.Error("no flags should not need render")
	}
	if !NeedsRender(".a", false) {
		t.Error("query set should need render")
	}
	if !NeedsRender("", true) {
		t.Error("output set should need render")
	}
}

const lists = `[{"id":"1","name":"To Do","closed":false},{"id":"2","name":"Doing","closed":true}]`

func TestRenderQueryTextRows(t *testing.T) {
	// Project to a list of [id,name] rows, render as text: tab-separated, one per line.
	out, err := Render("[].[id,name]", OutputText, []byte(lists))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "1\tTo Do\n2\tDoing\n"
	if string(out) != want {
		t.Errorf("text rows = %q, want %q", out, want)
	}
}

func TestRenderQueryScalarList(t *testing.T) {
	out, err := Render("[].name", OutputText, []byte(lists))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(out) != "To Do\nDoing\n" {
		t.Errorf("scalar list = %q", out)
	}
}

func TestRenderYAML(t *testing.T) {
	out, err := Render("[0].name", OutputYAML, []byte(lists))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.TrimSpace(string(out)) != "To Do" {
		t.Errorf("yaml scalar = %q", out)
	}
}

func TestRenderJSONProjection(t *testing.T) {
	out, err := Render("[?closed].id | [0]", OutputJSON, []byte(lists))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.TrimSpace(string(out)) != `"2"` {
		t.Errorf("json projection = %q, want \"2\"", out)
	}
}

func TestRenderIntegerNoTrailingDecimal(t *testing.T) {
	out, err := Render("count", OutputText, []byte(`{"count": 42}`))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(out) != "42\n" {
		t.Errorf("int text = %q, want 42", out)
	}
}

func TestRenderNonJSONErrors(t *testing.T) {
	if _, err := Render(".a", OutputJSON, []byte("not json")); err == nil {
		t.Error("expected error on non-JSON body")
	}
}

func TestRenderBadQueryErrors(t *testing.T) {
	if _, err := Render("[", OutputJSON, []byte(lists)); err == nil {
		t.Error("expected error on malformed JMESPath")
	}
}
