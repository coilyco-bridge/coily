#!/usr/bin/env python3
"""Generate a coily REST-wrapper Go file from an OpenAPI 3.x spec.

Output mirrors the hand-written shape of cmd/coily/glama/glama.go:
- one cli.Command tree per package
- subcommand groups by OpenAPI tag (first tag wins)
- one leaf per (path, method)
- path params -> positional positional args in declaration order
- query params -> --<param> string flags
- request body -> --body (literal | @file | -)
- bearer auth fetched from AWS SSM via the strict shell.Runner
- Trello-style key+token auth supported (two SSM params, query string)
- audit verb names: "ops.<pkg>.<tag-slug>.<op-slug>"

The generator does NOT model response schemas. JSON is streamed to stdout
pretty-printed; non-JSON bodies pass through verbatim. This keeps the
generator compact and the output Go free of generated type definitions
that would rot the moment the spec changes shape.

Usage:

  python3 scripts/openapi-to-coily.py \\
    --spec https://glama.ai/api/mcp/openapi.json \\
    --pkg glama \\
    --usage "Wrap the Glama MCP v1 REST API." \\
    --base-url https://glama.ai/api/mcp \\
    --auth-mode bearer-when-required \\
    --ssm-param /glama/api-key \\
    --mount pkg \\
    --out cmd/coily/glama/glama.go
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.parse
import urllib.request
from typing import Any

# ---------- spec loading ----------


def _fetch_json(url: str) -> Any:
    req = urllib.request.Request(
        url,
        headers={
            "User-Agent": "coily-openapi-to-coily/1.0 (+https://github.com/coilysiren/coily)",
            "Accept": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=60) as f:
        return json.load(f)


def load_spec(src: str) -> dict[str, Any]:
    """Load an OpenAPI spec from a URL or local path. Resolves $ref entries
    that point to sibling files (Sentry's spec is split this way)."""
    if src.startswith(("http://", "https://")):
        base = src
        spec = _fetch_json(src)
    else:
        base = os.path.abspath(src)
        with open(src, encoding="utf-8") as f:
            spec = json.load(f)
    return _resolve_refs(spec, base)


def _resolve_refs(node: Any, base: str, depth: int = 0) -> Any:
    """Walk the spec replacing $ref entries that point to external files
    with their content. Only resolves file-relative refs. Internal refs
    (#/components/...) are left alone; the generator does not need them."""
    if depth > 8:
        return node
    if isinstance(node, dict):
        if "$ref" in node and isinstance(node["$ref"], str):
            ref = node["$ref"]
            if ref.startswith("#"):
                return node
            new_url = _join_ref(base, ref)
            try:
                if new_url.startswith(("http://", "https://")):
                    sub = _fetch_json(new_url)
                else:
                    with open(new_url, encoding="utf-8") as f:
                        sub = json.load(f)
            except Exception as e:
                print(
                    f"warn: ref {ref} failed: {e}", file=sys.stderr
                )
                return {}
            return _resolve_refs(sub, new_url, depth + 1)
        return {k: _resolve_refs(v, base, depth) for k, v in node.items()}
    if isinstance(node, list):
        return [_resolve_refs(v, base, depth) for v in node]
    return node


def _join_ref(base: str, ref: str) -> str:
    if ref.startswith(("http://", "https://")):
        return ref
    if base.startswith(("http://", "https://")):
        return urllib.parse.urljoin(base, ref)
    return os.path.normpath(os.path.join(os.path.dirname(base), ref))


# ---------- naming ----------

SLUG_RE = re.compile(r"[^a-z0-9]+")


def go_str(s: str, maxlen: int = 100) -> str:
    """Escape a string for embedding inside a Go double-quoted literal.
    Strips newlines, escapes backslashes and double quotes, truncates."""
    s = (s or "").replace("\r", " ").replace("\n", " ").replace("\t", " ")
    # Collapse runs of whitespace.
    s = re.sub(r"\s+", " ", s).strip()
    if len(s) > maxlen:
        s = s[: maxlen - 1].rstrip() + "…"
    s = s.replace("\\", "\\\\").replace('"', "'")
    return s


def slug(s: str) -> str:
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1-\2", s)
    s = SLUG_RE.sub("-", s.lower()).strip("-")
    return s or "x"


def go_ident(s: str) -> str:
    parts = re.split(r"[^A-Za-z0-9]+", s)
    return "".join(p[:1].upper() + p[1:] for p in parts if p) or "X"


def derive_op_name(method: str, path: str, op: dict) -> str:
    op_id = op.get("operationId")
    if op_id:
        return slug(op_id)
    parts: list[str] = []
    for seg in path.split("/"):
        if not seg:
            continue
        if seg.startswith("{") and seg.endswith("}"):
            parts.append("by-" + seg[1:-1])
        else:
            parts.append(seg)
    return slug(f"{method}-{'-'.join(parts)}")


# ---------- spec iteration ----------


def iter_operations(spec: dict[str, Any]):
    paths = spec.get("paths", {}) or {}
    for path, item in paths.items():
        if not isinstance(item, dict):
            continue
        common_params = item.get("parameters", []) or []
        for method in (
            "get",
            "post",
            "put",
            "delete",
            "patch",
        ):
            op = item.get(method)
            if not isinstance(op, dict):
                continue
            if op.get("deprecated"):
                continue
            params = list(common_params) + list(op.get("parameters", []) or [])
            yield method, path, op, params


def split_params(params: list[dict]) -> tuple[list[dict], list[dict]]:
    path_p, query_p = [], []
    seen = set()
    for p in params:
        if not isinstance(p, dict):
            continue
        loc, name = p.get("in"), p.get("name")
        if not name or (loc, name) in seen:
            continue
        seen.add((loc, name))
        if loc == "path":
            path_p.append(p)
        elif loc == "query":
            query_p.append(p)
    return path_p, query_p


def has_security(op: dict, spec_security: list) -> bool:
    sec = op.get("security", spec_security)
    return bool(sec)


# ---------- code emission ----------

HEADER = '''// Package {pkg} wraps the {title} REST API ({base_url}).
//
// Generated by scripts/openapi-to-coily.py from {spec_url}. Do not edit
// by hand: re-run the generator. The generator emits one cli.Command per
// (path, method) tuple, grouped by OpenAPI tag. Path parameters are
// positional in declaration order; query parameters become --<name>
// string flags; request bodies arrive via --body (literal | @file | -).
// Responses stream to stdout pretty-printed (JSON) or verbatim (other).
//
// Two persistent root flags, --query (JMESPath projection) and --output
// (json|yaml|text), fold coily's REST projection rail (coilyco-bridge/coily#46)
// into every leaf via restfmt. When neither is set the response is emitted
// byte-for-byte as before. An op that already owns an API --query parameter
// exposes the projection as --jq instead, mirroring the gh-rest --jq rail.
//
// {auth_doc}
package {pkg}

import (
\t"bytes"
\t"context"
\t"encoding/json"
\t"fmt"
\t"io"
\t"net/http"
\t"net/url"
\t"os"
\t"strings"
\t"time"

\t"forgejo.coilysiren.me/coilyco-bridge/coily/cmd/coily/restfmt"
\t"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/audit"
\t"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/shell"
\t"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
\t"github.com/urfave/cli/v3"
)

const (
\tapiBase = "{base_url}"
\thttpTimeout = 60 * time.Second
{ssm_consts}
)

// Command returns the cli.Command tree for `coily {mount} {pkg}`.
func Command(r *shell.Runner, w *audit.Writer) *cli.Command {{
\treturn &cli.Command{{
\t\tName:  "{pkg}",
\t\tUsage: "{usage}",
\t\tFlags: []cli.Flag{{
\t\t\t// urfave/cli v3 flags propagate to every subcommand by default
\t\t\t// (Local defaults false), so these reach every generated leaf. A
\t\t\t// leaf that declares a same-named local flag shadows them.
\t\t\t&cli.StringFlag{{Name: "query", Usage: "JMESPath projection applied to the JSON response (coily#46)"}},
\t\t\t&cli.StringFlag{{Name: "output", Usage: "output format: json (default), yaml, text (coily#46)"}},
\t\t}},
\t\tCommands: []*cli.Command{{
{root_subs}
\t\t}},
\t}}
}}

'''

GROUP_FN = '''func group{group_ident}(r *shell.Runner, w *audit.Writer) *cli.Command {{
\treturn &cli.Command{{
\t\tName:  "{name}",
\t\tUsage: "{usage}",
\t\tCommands: []*cli.Command{{
{leaves}
\t\t}},
\t}}
}}

'''

LEAF_FN = '''func op{op_ident}(r *shell.Runner, w *audit.Writer) *cli.Command {{
\treturn &cli.Command{{
\t\tName:  "{name}",
\t\tUsage: "{usage}",
{args_usage}{flags_block}\t\tAction: verb.Wrap(
\t\t\tverb.Spec{{
\t\t\t\tName: "{verb_name}",
\t\t\t\tArgsFunc: func(c *cli.Command) (map[string]string, []string) {{
{argsfunc_body}\t\t\t\t}},
\t\t\t\tAction: func(ctx context.Context, c *cli.Command) error {{
{action_body}\t\t\t\t}},
\t\t\t}},
\t\t\tw,
\t\t),
\t}}
}}

'''

HELPERS = r'''
// readBody resolves the --body flag: literal JSON, '@path' for file,
// '-' for stdin. Empty input returns nil.
func readBody(spec string) ([]byte, error) {
	switch {
	case spec == "":
		return nil, nil
	case spec == "-":
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("{pkg}: read stdin: %w", err)
		}
		return b, nil
	case strings.HasPrefix(spec, "@"):
		b, err := os.ReadFile(spec[1:])
		if err != nil {
			return nil, fmt.Errorf("{pkg}: read %s: %w", spec[1:], err)
		}
		return b, nil
	default:
		return []byte(spec), nil
	}
}

func doRequestAndStream(ctx context.Context, r *shell.Runner, method, path string, q url.Values, body []byte, authNeeded bool, projection, output string) error {
	target := apiBase + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, rdr)
	if err != nil {
		return fmt.Errorf("{pkg}: build request: %w", err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
{auth_apply}
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("{pkg}: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("{pkg}: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("{pkg}: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	outFmt, outSet, err := restfmt.ParseOutput(output)
	if err != nil {
		return err
	}
	if restfmt.NeedsRender(projection, outSet) {
		// --query alone defaults to JSON, preserving the surface's existing
		// output format; --output flips it to yaml/text. Either routes through
		// restfmt instead of the byte-identical pretty-print path below.
		if !outSet {
			outFmt = restfmt.OutputJSON
		}
		rendered, rErr := restfmt.Render(projection, outFmt, raw)
		if rErr != nil {
			return rErr
		}
		_, err = os.Stdout.Write(rendered)
		return err
	}
	var pretty json.RawMessage
	if err := json.Unmarshal(raw, &pretty); err != nil {
		_, _ = os.Stdout.Write(raw)
		return nil
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return fmt.Errorf("{pkg}: re-encode response: %w", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, string(out)); err != nil {
		return err
	}
	return nil
}

func fetchSSM(ctx context.Context, r *shell.Runner, name string) (string, error) {
	out, err := r.Capture(ctx, "aws",
		"ssm", "get-parameter",
		"--name", name,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("{pkg}: fetch %s: %w", name, err)
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("{pkg}: %s resolved to empty value", name)
	}
	return v, nil
}
'''


AUTH_APPLY_BEARER = '''\tif authNeeded {
\t\tkey, err := fetchSSM(ctx, r, ssmAPIKey)
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\treq.Header.Set("Authorization", "Bearer "+key)
\t}
'''

AUTH_APPLY_DISCORD_BOT = '''\tif authNeeded {
\t\tkey, err := fetchSSM(ctx, r, ssmAPIKey)
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\treq.Header.Set("Authorization", "Bot "+key)
\t}
'''

AUTH_APPLY_TRELLO = '''\tif authNeeded {
\t\tk, err := fetchSSM(ctx, r, ssmAPIKey)
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\tt, err := fetchSSM(ctx, r, ssmAPIToken)
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\tu := req.URL
\t\tqv := u.Query()
\t\tqv.Set("key", k)
\t\tqv.Set("token", t)
\t\tu.RawQuery = qv.Encode()
\t}
'''

AUTH_APPLY_NONE = ""


def emit_leaf(
    pkg: str,
    method: str,
    path: str,
    op: dict,
    params: list[dict],
    auth_mode: str,
    spec_security: list,
    group_slug: str,
) -> tuple[str, str]:
    op_slug = derive_op_name(method, path, op)
    op_ident = go_ident(f"{group_slug}-{op_slug}")
    leaf_name = op_slug
    summary = go_str(
        op.get("summary") or op.get("description") or f"{method.upper()} {path}",
        120,
    )
    verb_name = f"ops.{pkg}.{group_slug}.{op_slug}"

    path_p, query_p = split_params(params)

    # Collision handling for the persistent --query/--output projection rail
    # (coily#46). An op that already owns an API query parameter of the same
    # name shadows the persistent flag, so the coily projection moves to --jq
    # (mirroring the gh-rest --jq rail) and the output formatter is suppressed
    # rather than mis-reading the API value as a format name.
    query_names = {qp["name"] for qp in query_p}
    proj_collides = "query" in query_names
    out_collides = "output" in query_names
    proj_expr = 'c.String("jq")' if proj_collides else 'c.String("query")'
    out_expr = '""' if out_collides else 'c.String("output")'

    has_body = method in ("post", "put", "patch", "delete") and (
        op.get("requestBody") is not None
    )

    flag_lines = []
    args_kv = []
    for qp in query_p:
        n = qp["name"]
        desc = go_str(qp.get("description", ""), 80)
        flag_lines.append(
            f'\t\t\t&cli.StringFlag{{Name: "{n}", Usage: "query: {desc}"}},'
        )
        args_kv.append(
            f'\t\t\t\t\tif c.IsSet("{n}") {{ m["--{n}"] = c.String("{n}") }}'
        )
    if has_body:
        flag_lines.append(
            '\t\t\t&cli.StringFlag{Name: "body", Usage: "JSON body, \'@file\', or \'-\' for stdin"},'
        )
        args_kv.append(
            '\t\t\t\t\tif c.IsSet("body") { m["--body"] = c.String("body") }'
        )
    if proj_collides:
        # This op owns an API --query; expose the coily JMESPath projection as
        # --jq so the persistent root --query stays usable (coily#46).
        flag_lines.append(
            '\t\t\t&cli.StringFlag{Name: "jq", Usage: "JMESPath projection applied to the JSON response; --jq here because this op owns an API --query (coily#46)"},'
        )

    flags_block = ""
    if flag_lines:
        flags_block = (
            "\t\tFlags: []cli.Flag{\n" + "\n".join(flag_lines) + "\n\t\t},\n"
        )

    pos_args = [p["name"] for p in path_p]
    args_usage = ""
    if pos_args:
        args_usage = f'\t\tArgsUsage: "{" ".join("<" + n + ">" for n in pos_args)}",\n'

    argsfunc_lines = ["\t\t\t\t\tm := map[string]string{}"]
    argsfunc_lines += args_kv
    argsfunc_lines.append("\t\t\t\t\treturn m, c.Args().Slice()")
    argsfunc_body = "\n".join(argsfunc_lines) + "\n"

    action_lines = []
    if pos_args:
        action_lines.append(f"\t\t\t\t\targs := c.Args().Slice()")
        action_lines.append(
            f'\t\t\t\t\tif len(args) != {len(pos_args)} {{ return fmt.Errorf("{pkg}: need {len(pos_args)} positional args ({" ".join(pos_args)}), got %d", len(args)) }}'
        )
        for i, n in enumerate(pos_args):
            action_lines.append(
                f'\t\t\t\t\tp{i} := url.PathEscape(args[{i}])'
            )

    path_literal = path
    for i, n in enumerate(pos_args):
        path_literal = path_literal.replace("{" + n + "}", f'"+p{i}+"')
    path_literal = '"' + path_literal + '"'
    if path_literal.endswith('+""'):
        path_literal = path_literal[:-3]
    if path_literal.startswith('""+'):
        path_literal = path_literal[3:]

    action_lines.append(f"\t\t\t\t\tpath := {path_literal}")
    action_lines.append("\t\t\t\t\tq := url.Values{}")
    for qp in query_p:
        n = qp["name"]
        action_lines.append(
            f'\t\t\t\t\tif c.IsSet("{n}") {{ q.Set("{n}", c.String("{n}")) }}'
        )

    if has_body:
        action_lines.append(
            '\t\t\t\t\tbody, err := readBody(c.String("body"))'
        )
        action_lines.append("\t\t\t\t\tif err != nil { return err }")
    else:
        action_lines.append("\t\t\t\t\tvar body []byte")

    op_security_required = bool(op.get("security", spec_security))
    auth_needed = (
        "true"
        if (
            auth_mode in ("bearer-required", "trello-keytoken", "discord-bot")
            or (auth_mode == "bearer-when-required" and op_security_required)
        )
        else "false"
    )

    action_lines.append(
        f'\t\t\t\t\treturn doRequestAndStream(ctx, r, "{method.upper()}", path, q, body, {auth_needed}, {proj_expr}, {out_expr})'
    )

    action_body = "\n".join(action_lines) + "\n"

    leaf_fn = LEAF_FN.format(
        op_ident=op_ident,
        name=leaf_name,
        usage=summary,
        args_usage=args_usage,
        flags_block=flags_block,
        verb_name=verb_name,
        argsfunc_body=argsfunc_body,
        action_body=action_body,
    )
    return op_ident, leaf_fn


# ---------- main ----------


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--spec", required=True)
    ap.add_argument("--pkg", required=True)
    ap.add_argument("--usage", required=True)
    ap.add_argument("--base-url", required=True)
    ap.add_argument(
        "--auth-mode",
        choices=[
            "none",
            "bearer-required",
            "bearer-when-required",
            "discord-bot",
            "trello-keytoken",
        ],
        default="bearer-when-required",
    )
    ap.add_argument(
        "--ssm-param",
        help="primary SSM param (bearer key, Trello key)",
        default="",
    )
    ap.add_argument(
        "--ssm-param-aux",
        help="secondary SSM param (Trello token)",
        default="",
    )
    ap.add_argument("--out", required=True)
    ap.add_argument(
        "--mount",
        default="ops",
        help="top-level coily group the wrapper mounts under (e.g. `ops` or `pkg`); affects only the doc comment.",
    )
    args = ap.parse_args()

    spec = load_spec(args.spec)
    title = spec.get("info", {}).get("title", args.pkg)
    spec_security = spec.get("security", []) or []

    auth_doc = {
        "none": "No authentication required.",
        "bearer-required": f"Bearer auth from AWS SSM at {args.ssm_param} on every call.",
        "bearer-when-required": f"Bearer auth from AWS SSM at {args.ssm_param}, fetched only for endpoints that declare it.",
        "discord-bot": f"Discord Bot auth from AWS SSM at {args.ssm_param} on every call.",
        "trello-keytoken": f"Trello key+token auth from AWS SSM at {args.ssm_param} (key) and {args.ssm_param_aux} (token), appended as query parameters.",
    }[args.auth_mode]

    ssm_consts_parts = []
    if args.ssm_param:
        ssm_consts_parts.append(
            f'\tssmAPIKey = "{args.ssm_param}" //nolint:gosec // SSM path, not a credential'
        )
    if args.ssm_param_aux:
        ssm_consts_parts.append(
            f'\tssmAPIToken = "{args.ssm_param_aux}" //nolint:gosec // SSM path, not a credential'
        )
    ssm_consts = "\n".join(ssm_consts_parts)

    auth_apply = {
        "none": AUTH_APPLY_NONE,
        "bearer-required": AUTH_APPLY_BEARER,
        "bearer-when-required": AUTH_APPLY_BEARER,
        "discord-bot": AUTH_APPLY_DISCORD_BOT,
        "trello-keytoken": AUTH_APPLY_TRELLO,
    }[args.auth_mode]

    # Group operations by first tag.
    groups: dict[str, list[tuple[str, str, dict, list]]] = {}
    for method, path, op, params in iter_operations(spec):
        tags = op.get("tags") or []
        if tags:
            g = slug(tags[0])
        else:
            # Fallback: first non-empty, non-template path segment.
            # Skips /api, /v1, /v10 prefixes by preferring the first
            # alphabetic segment longer than 2 chars.
            segs = [s for s in path.split("/") if s and not s.startswith("{")]
            chosen = next(
                (s for s in segs if len(s) > 2 and not re.fullmatch(r"v\d+", s) and s != "api"),
                segs[0] if segs else "default",
            )
            g = slug(chosen)
        groups.setdefault(g, []).append((method, path, op, params))

    # Emit each group's leaves.
    group_fns = []
    leaf_fns = []
    root_subs = []
    seen_op_idents = set()
    seen_leaf_names_per_group: dict[str, set[str]] = {}

    for g, ops in sorted(groups.items()):
        group_ident = go_ident(g)
        leaf_calls = []
        seen_leaves = seen_leaf_names_per_group.setdefault(g, set())
        for method, path, op, params in ops:
            op_ident, fn = emit_leaf(
                args.pkg, method, path, op, params, args.auth_mode, spec_security, g
            )
            # Dedup: if two ops produce the same name within a group, suffix
            # with method to disambiguate.
            base_name = op_ident
            i = 2
            while op_ident in seen_op_idents:
                new_ident = f"{base_name}{i}"
                fn = fn.replace(f"op{op_ident}(", f"op{new_ident}(", 1)
                op_ident = new_ident
                i += 1
            seen_op_idents.add(op_ident)

            # Also dedup leaf names within a group.
            leaf_name_match = re.search(r'Name:  "([^"]+)",', fn)
            if leaf_name_match:
                leaf_name = leaf_name_match.group(1)
                base_leaf = leaf_name
                j = 2
                while leaf_name in seen_leaves:
                    leaf_name = f"{base_leaf}-{method}{j if j > 2 else ''}"
                    j += 1
                if leaf_name != base_leaf:
                    fn = fn.replace(f'Name:  "{base_leaf}",', f'Name:  "{leaf_name}",', 1)
                seen_leaves.add(leaf_name)

            leaf_fns.append(fn)
            leaf_calls.append(f"\t\t\top{op_ident}(r, w),")
        group_fns.append(
            GROUP_FN.format(
                group_ident=group_ident,
                name=g,
                usage=f"{title} endpoints tagged {g}.",
                leaves="\n".join(leaf_calls),
            )
        )
        root_subs.append(f"\t\t\tgroup{group_ident}(r, w),")

    out = HEADER.format(
        pkg=args.pkg,
        mount=args.mount,
        title=title,
        base_url=args.base_url,
        spec_url=args.spec,
        auth_doc=auth_doc,
        ssm_consts=ssm_consts,
        usage=args.usage,
        root_subs="\n".join(root_subs),
    )
    out += "".join(group_fns)
    out += "".join(leaf_fns)
    out += HELPERS.replace("{pkg}", args.pkg).replace(
        "{auth_apply}", auth_apply
    )

    os.makedirs(os.path.dirname(args.out), exist_ok=True)
    with open(args.out, "w", encoding="utf-8") as f:
        f.write(out)
    # gofmt the output. Best-effort: if gofmt isn't on PATH, leave the file
    # as-is and let the linter complain.
    import subprocess

    try:
        subprocess.run(["gofmt", "-w", args.out], check=False)
    except FileNotFoundError:
        pass
    print(
        f"wrote {args.out}: {sum(len(ops) for ops in groups.values())} operations across {len(groups)} groups",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
