// Package respfmt renders a JSON response body through an optional
// JMESPath projection and one of five output formats: yaml (default),
// json, text, table, yaml-stream. The shape mirrors the aws CLI's
// `--query` / `--output` surface so an operator's existing muscle memory
// transfers (with the default flipped to yaml to match coily's
// file-output convention).
//
// Pure helper: no cli wiring, no I/O. The caller decides where to write.
// Wired into REST wrappers (OpenAPI-generated and gh REST rewriter)
// per coilysiren/coily#134.
package respfmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jmespath/go-jmespath"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

// Output names mirror the aws CLI vocabulary verbatim.
const (
	OutputYAML       = "yaml"
	OutputYAMLStream = "yaml-stream"
	OutputJSON       = "json"
	OutputText       = "text"
	OutputTable      = "table"
)

// validOutputs is the closed set the renderer accepts. Empty string is
// also accepted at the API boundary and treated as the default.
var validOutputs = map[string]bool{
	OutputYAML: true, OutputYAMLStream: true, OutputJSON: true,
	OutputText: true, OutputTable: true,
}

// Render applies the optional JMESPath projection to raw and formats the
// result per output. Empty raw returns empty bytes, no error. Empty query
// passes the parsed value through unchanged. Empty output defaults to
// OutputYAML.
//
// Errors:
//   - raw is non-empty but not parseable JSON.
//   - query is non-empty but not a valid JMESPath expression.
//   - output is non-empty but not in validOutputs.
//   - the requested format cannot represent the projection (e.g. text on
//     a non-list-of-records value).
func Render(raw []byte, query, output string) ([]byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	if output == "" {
		output = OutputYAML
	}
	if !validOutputs[output] {
		return nil, fmt.Errorf("respfmt: unknown output %q (want yaml | yaml-stream | json | text | table)", output)
	}

	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("respfmt: parse response as json: %w", err)
	}

	if query != "" {
		projected, err := jmespath.Search(query, data)
		if err != nil {
			return nil, fmt.Errorf("respfmt: jmespath %q: %w", query, err)
		}
		data = projected
	}

	switch output {
	case OutputJSON:
		return renderJSON(data)
	case OutputYAML:
		return renderYAML(data)
	case OutputYAMLStream:
		return renderYAMLStream(data)
	case OutputText:
		return renderText(data)
	case OutputTable:
		return renderTable(data)
	}
	// Unreachable: validOutputs gate above covers every case.
	return nil, fmt.Errorf("respfmt: unhandled output %q", output)
}

func renderJSON(data any) ([]byte, error) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("respfmt: render json: %w", err)
	}
	return append(out, '\n'), nil
}

func renderYAML(data any) ([]byte, error) {
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("respfmt: render yaml: %w", err)
	}
	return out, nil
}

// renderYAMLStream emits each top-level list element as its own yaml
// document separated by `---\n`. For non-list values, falls back to a
// single document, matching aws CLI's behavior.
func renderYAMLStream(data any) ([]byte, error) {
	list, ok := data.([]any)
	if !ok {
		return renderYAML(data)
	}
	var buf bytes.Buffer
	for _, item := range list {
		buf.WriteString("---\n")
		out, err := yaml.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("respfmt: render yaml-stream: %w", err)
		}
		buf.Write(out)
	}
	return buf.Bytes(), nil
}

// renderText emits tab-separated rows. Accepts the aws CLI shapes:
//
//   - scalar: one line, the scalar's stringification.
//   - list of scalars: one scalar per line.
//   - list of lists (rows): tab-joined per row.
//   - list of maps: tab-joined per row, columns sorted by key.
//
// Other shapes (single map, nested maps) decline with an error so the
// caller can pick a richer format.
func renderText(data any) ([]byte, error) {
	switch v := data.(type) {
	case nil:
		return nil, nil
	case []any:
		var buf bytes.Buffer
		for _, item := range v {
			buf.WriteString(textRow(item))
			buf.WriteByte('\n')
		}
		return buf.Bytes(), nil
	default:
		return append([]byte(textRow(v)), '\n'), nil
	}
}

func textRow(item any) string {
	switch v := item.(type) {
	case nil:
		return ""
	case []any:
		parts := make([]string, len(v))
		for i, e := range v {
			parts[i] = scalarString(e)
		}
		return strings.Join(parts, "\t")
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = scalarString(v[k])
		}
		return strings.Join(parts, "\t")
	default:
		return scalarString(v)
	}
}

// renderTable emits a bordered ASCII table. Accepts:
//
//   - list of maps: column headers from the union of keys (sorted).
//   - list of lists: numeric column headers.
//
// Other shapes decline so the caller can pick a richer format. Non-list
// inputs return an error rather than guessing a single-row layout that
// would be confusing in a table.
func renderTable(data any) ([]byte, error) {
	list, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("respfmt: --output table requires a list at the projection root")
	}
	if len(list) == 0 {
		return nil, nil
	}
	headers, rows, err := tableRows(list)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	tw := tablewriter.NewWriter(&buf)
	headerArgs := make([]any, len(headers))
	for i, h := range headers {
		headerArgs[i] = h
	}
	tw.Header(headerArgs...)
	for _, r := range rows {
		rowArgs := make([]any, len(r))
		for i, c := range r {
			rowArgs[i] = c
		}
		if err := tw.Append(rowArgs...); err != nil {
			return nil, fmt.Errorf("respfmt: append table row: %w", err)
		}
	}
	if err := tw.Render(); err != nil {
		return nil, fmt.Errorf("respfmt: render table: %w", err)
	}
	return buf.Bytes(), nil
}

func tableRows(list []any) ([]string, [][]string, error) {
	switch list[0].(type) {
	case map[string]any:
		return tableRowsFromMaps(list)
	case []any:
		return tableRowsFromLists(list)
	default:
		return nil, nil, fmt.Errorf("respfmt: --output table requires list rows to be maps or lists, got %T", list[0])
	}
}

func tableRowsFromMaps(list []any) ([]string, [][]string, error) {
	keySet := map[string]bool{}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("respfmt: --output table rows must be uniform; mixed types not supported")
		}
		for k := range m {
			keySet[k] = true
		}
	}
	headers := make([]string, 0, len(keySet))
	for k := range keySet {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	rows := make([][]string, 0, len(list))
	for _, item := range list {
		m := item.(map[string]any)
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = scalarString(m[h])
		}
		rows = append(rows, row)
	}
	return headers, rows, nil
}

func tableRowsFromLists(list []any) ([]string, [][]string, error) {
	width := len(list[0].([]any))
	headers := make([]string, width)
	for i := range headers {
		headers[i] = fmt.Sprintf("col%d", i)
	}
	rows := make([][]string, 0, len(list))
	for _, item := range list {
		r, ok := item.([]any)
		if !ok || len(r) != width {
			return nil, nil, fmt.Errorf("respfmt: --output table rows must be uniform-width lists")
		}
		row := make([]string, width)
		for i, e := range r {
			row[i] = scalarString(e)
		}
		rows = append(rows, row)
	}
	return headers, rows, nil
}

// scalarString formats a JSON-decoded value for text/table cells. Nested
// containers are JSON-encoded inline so a stringification of "everything
// fits in a cell" is consistent across text and table outputs.
func scalarString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "True"
		}
		return "False"
	case float64:
		// JSON unmarshal produces float64 for all numbers; render
		// integer-valued floats without a trailing .0.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	default:
		out, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%v", x)
		}
		return string(out)
	}
}
