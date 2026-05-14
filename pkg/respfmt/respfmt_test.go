package respfmt_test

import (
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/respfmt"
)

func TestRender_DefaultsToYAML(t *testing.T) {
	got, err := respfmt.Render([]byte(`{"a":1}`), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "a: 1") {
		t.Errorf("default output should be yaml; got %q", got)
	}
}

func TestRender_EmptyInputReturnsEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "\n"} {
		got, err := respfmt.Render([]byte(in), "", "")
		if err != nil {
			t.Errorf("empty input %q: unexpected err %v", in, err)
		}
		if len(got) != 0 {
			t.Errorf("empty input %q: got %q, want empty", in, got)
		}
	}
}

func TestRender_RejectsBadJSON(t *testing.T) {
	if _, err := respfmt.Render([]byte("not json"), "", ""); err == nil {
		t.Error("non-json input should error")
	}
}

func TestRender_RejectsUnknownOutput(t *testing.T) {
	if _, err := respfmt.Render([]byte(`{}`), "", "csv"); err == nil {
		t.Error("unknown output should error")
	}
}

func TestRender_JMESPathProjection(t *testing.T) {
	raw := []byte(`{"items":[{"id":"a","n":1},{"id":"b","n":2}]}`)
	got, err := respfmt.Render(raw, "items[].id", respfmt.OutputJSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"a"`, `"b"`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("projection missing %q in %s", want, got)
		}
	}
}

func TestRender_JSON(t *testing.T) {
	got, err := respfmt.Render([]byte(`{"a":1}`), "", respfmt.OutputJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"a": 1`) {
		t.Errorf("json output should be pretty; got %q", got)
	}
}

func TestRender_YAMLStream(t *testing.T) {
	got, err := respfmt.Render([]byte(`[{"a":1},{"a":2}]`), "", respfmt.OutputYAMLStream)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if strings.Count(s, "---") != 2 {
		t.Errorf("yaml-stream should emit one document marker per list element; got %q", s)
	}
}

func TestRender_Text_ListOfRows(t *testing.T) {
	raw := []byte(`[["a","1"],["b","2"]]`)
	got, err := respfmt.Render(raw, "", respfmt.OutputText)
	if err != nil {
		t.Fatal(err)
	}
	want := "a\t1\nb\t2\n"
	if string(got) != want {
		t.Errorf("text rows = %q, want %q", got, want)
	}
}

func TestRender_Text_ListOfMapsSortsKeys(t *testing.T) {
	raw := []byte(`[{"name":"alpha","id":"a1"},{"name":"beta","id":"b2"}]`)
	got, err := respfmt.Render(raw, "", respfmt.OutputText)
	if err != nil {
		t.Fatal(err)
	}
	// id sorts before name alphabetically.
	want := "a1\talpha\nb2\tbeta\n"
	if string(got) != want {
		t.Errorf("text maps = %q, want %q", got, want)
	}
}

func TestRender_Table_ListOfMaps(t *testing.T) {
	raw := []byte(`[{"id":"a","name":"alpha"}]`)
	got, err := respfmt.Render(raw, "", respfmt.OutputTable)
	if err != nil {
		t.Fatal(err)
	}
	// tablewriter v1 renders headers uppercased and uses unicode box
	// characters by default. We verify the data lands and the box is
	// drawn; exact glyphs are an implementation detail of tablewriter.
	for _, want := range []string{"ID", "NAME", "a", "alpha"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("table missing %q in:\n%s", want, got)
		}
	}
	if !strings.ContainsAny(string(got), "+┌") {
		t.Errorf("table should have a top border in:\n%s", got)
	}
}

func TestRender_Table_RejectsScalar(t *testing.T) {
	if _, err := respfmt.Render([]byte(`{"a":1}`), "", respfmt.OutputTable); err == nil {
		t.Error("table on a non-list should error")
	}
}

func TestRender_RejectsBadJMESPath(t *testing.T) {
	if _, err := respfmt.Render([]byte(`{}`), "[[[", ""); err == nil {
		t.Error("malformed jmespath should error")
	}
}

func TestRender_NumberFormatting(t *testing.T) {
	// integer-valued floats should render without .0
	got, err := respfmt.Render([]byte(`[1,2,3]`), "", respfmt.OutputText)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1\n2\n3\n" {
		t.Errorf("integer-valued floats should drop .0; got %q", got)
	}
}
