// Package restfmt renders a JSON REST response through an optional JMESPath
// --query projection and an --output formatter (json | yaml | text), mirroring
// the AWS CLI's output surface. It is the shared core behind coily's
// --query/--output flags on REST wrappers (coilyco-bridge/coily#46), kept
// dependency-light and free of any wrapper-specific types so every generated
// surface (trello, gh-rest, sentry, modio, discord) can call it.
package restfmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmespath/go-jmespath"
	"gopkg.in/yaml.v3"
)

// Output is an --output format name.
type Output string

const (
	OutputJSON Output = "json"
	OutputYAML Output = "yaml"
	OutputText Output = "text"
)

// ParseOutput validates an --output value. An empty string is allowed and
// reported via ok=false so callers can fall back to their default.
func ParseOutput(s string) (Output, bool, error) {
	if s == "" {
		return "", false, nil
	}
	switch Output(s) {
	case OutputJSON, OutputYAML, OutputText:
		return Output(s), true, nil
	default:
		return "", false, fmt.Errorf("restfmt: unknown --output %q (want json, yaml, or text)", s)
	}
}

// Render applies an optional JMESPath query to raw JSON and formats the result
// per out. When query is empty the whole body is formatted. raw must be valid
// JSON; a caller that may hold non-JSON should guard with NeedsRender first.
func Render(query string, out Output, raw []byte) ([]byte, error) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("restfmt: response is not JSON, cannot apply --query/--output: %w", err)
	}
	if query != "" {
		projected, err := jmespath.Search(query, data)
		if err != nil {
			return nil, fmt.Errorf("restfmt: --query %q: %w", query, err)
		}
		data = projected
	}
	switch out {
	case OutputJSON:
		return marshalJSON(data)
	case OutputYAML:
		return marshalYAML(data)
	case OutputText:
		return marshalText(data)
	default:
		return nil, fmt.Errorf("restfmt: unknown output %q", out)
	}
}

// NeedsRender reports whether the caller supplied either flag. When neither is
// set, a wrapper should preserve its existing default output untouched.
func NeedsRender(query string, outSet bool) bool {
	return query != "" || outSet
}

func marshalJSON(data any) ([]byte, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("restfmt: json encode: %w", err)
	}
	return append(b, '\n'), nil
}

func marshalYAML(data any) ([]byte, error) {
	b, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("restfmt: yaml encode: %w", err)
	}
	return b, nil
}

// marshalText mirrors `aws --output text`: a list of records prints one
// tab-separated row per element, a list of scalars prints one per line, a map
// prints its values tab-separated, and a scalar prints verbatim. Null renders
// empty.
func marshalText(data any) ([]byte, error) {
	var b bytes.Buffer
	switch v := data.(type) {
	case nil:
		// empty
	case []any:
		for _, el := range v {
			b.WriteString(textRow(el))
			b.WriteByte('\n')
		}
	case map[string]any:
		b.WriteString(textRow(v))
		b.WriteByte('\n')
	default:
		b.WriteString(scalarText(v))
		b.WriteByte('\n')
	}
	return b.Bytes(), nil
}

// textRow renders one element of a top-level list as a tab-separated line.
func textRow(el any) string {
	switch v := el.(type) {
	case []any:
		parts := make([]string, len(v))
		for i, f := range v {
			parts[i] = scalarText(f)
		}
		return strings.Join(parts, "\t")
	case map[string]any:
		// Deterministic field order is the caller's job via a --query
		// projection to a list; a bare object falls back to JMESPath's
		// natural map iteration being unstable, so sort keys for stability.
		keys := sortedKeys(v)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = scalarText(v[k])
		}
		return strings.Join(parts, "\t")
	default:
		return scalarText(v)
	}
}

func scalarText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		// JSON numbers decode to float64; print integers without a trailing .0.
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	case bool:
		return fmt.Sprintf("%t", t)
	default:
		// Nested object/array inside a text cell: compact JSON, like aws.
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// simple insertion sort to avoid importing sort for a tiny slice
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
