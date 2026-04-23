#!/usr/bin/env python3
# Verify pkg/shell/tools.yaml is internally coherent and, where possible,
# cross-check entries against upstream-published checksums.
#
# Runs in two places:
#   1. pre-commit (local + CI) when pkg/shell/tools.yaml is staged.
#   2. .github/workflows/release-tools.yml, after the workflow rewrites
#      the manifest and before it commits the result.
#
# Checks performed per (tool, platform) entry:
#
#   Schema:
#     - sha256 is 64 hex chars and not a PLACEHOLDER_* sentinel
#     - url starts with release_url_base
#     - url basename matches "<tool>-<goos>-<goarch>"
#     - version is non-empty
#
#   Upstream cross-check:
#     - kubectl: fetch "<upstream_url>.sha256" (k8s.io publishes this
#       alongside every binary) and assert it equals entry.sha256.
#       Independent authoritative source, catches a CDN that serves
#       tampered bytes without also editing the checksum file.
#     - gh: not yet cross-checked. cli/cli publishes a signed
#       checksums.txt per release but it lists archive hashes, not the
#       extracted binary. Wiring that in requires re-fetching and
#       extracting the archive, which duplicates the release workflow's
#       own download step. Left as a TODO, optionally to be paired with
#       cosign verification of checksums.txt.
#     - aws: no upstream per-binary checksum published. No cross-check.

import argparse
import pathlib
import re
import sys
import urllib.error
import urllib.request

import yaml

MANIFEST_PATH = pathlib.Path("pkg/shell/tools.yaml")
HEX64 = re.compile(r"^[0-9a-f]{64}$")


def fail(msg: str, errors: list[str]) -> None:
    errors.append(msg)


def http_get(url: str, timeout: int = 30) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": "coily-verify-tools/1"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read()


def check_schema(
    tool: str,
    platform: str,
    entry: dict,
    base_url: str,
    errors: list[str],
    allow_placeholders: bool = False,
) -> bool:
    """Validate one entry. Returns True iff the entry is eligible for
    upstream cross-check (all schema rules pass AND sha is a real hex
    string)."""
    sha = entry.get("sha256", "")
    sha_ok = True
    if sha.startswith("PLACEHOLDER_"):
        if not allow_placeholders:
            fail(f"{tool} {platform}: sha256 is still a placeholder", errors)
        sha_ok = False
    elif not HEX64.match(sha):
        fail(f"{tool} {platform}: sha256 is not 64 hex chars: {sha!r}", errors)
        sha_ok = False

    url = entry.get("url", "")
    if not url.startswith(base_url):
        fail(f"{tool} {platform}: url not under release_url_base: {url!r}", errors)

    base_name = f"{tool}-{platform.replace('/', '-')}"
    archive = entry.get("archive", "")
    if archive:
        if archive != "tar.gz":
            fail(f"{tool} {platform}: unsupported archive format {archive!r}", errors)
        expected_basename = f"{base_name}.{archive}"
        if not entry.get("entry"):
            fail(f"{tool} {platform}: archive is set but `entry` path is empty", errors)
    else:
        expected_basename = base_name
    if not url.endswith("/" + expected_basename):
        fail(f"{tool} {platform}: url basename should be {expected_basename!r}: {url!r}", errors)

    if not entry.get("version"):
        fail(f"{tool} {platform}: version is empty", errors)

    if not entry.get("upstream_url"):
        fail(f"{tool} {platform}: upstream_url is empty", errors)

    return sha_ok


def cross_check_kubectl(platform: str, entry: dict, errors: list[str]) -> bool:
    sha_url = entry["upstream_url"] + ".sha256"
    try:
        body = http_get(sha_url).decode("ascii").strip()
    except urllib.error.URLError as e:
        fail(f"kubectl {platform}: fetching {sha_url}: {e}", errors)
        return False
    # k8s.io returns just the hex hash, sometimes with a trailing newline.
    upstream_sha = body.split()[0].lower()
    if not HEX64.match(upstream_sha):
        fail(f"kubectl {platform}: upstream .sha256 is not 64 hex chars: {body!r}", errors)
        return False
    if upstream_sha != entry["sha256"].lower():
        fail(
            f"kubectl {platform}: upstream sha256 {upstream_sha} does not match "
            f"manifest sha256 {entry['sha256']}",
            errors,
        )
        return False
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--allow-placeholders",
        action="store_true",
        help=(
            "Don't error on PLACEHOLDER_* sha256 values. Used by "
            "pre-commit when a schema change is committed ahead of the "
            "release-tools workflow rewriting real SHAs. The workflow "
            "itself always runs without this flag so release-time "
            "manifests cannot slip through with placeholders."
        ),
    )
    args = parser.parse_args()

    if not MANIFEST_PATH.exists():
        print(f"error: {MANIFEST_PATH} not found (run from repo root)", file=sys.stderr)
        return 2
    try:
        manifest = yaml.safe_load(MANIFEST_PATH.read_text())
    except yaml.YAMLError as e:
        print(f"error: {MANIFEST_PATH} is not valid YAML: {e}", file=sys.stderr)
        return 2

    base_url = manifest.get("release_url_base", "")
    tools = manifest.get("tools", {})
    if not tools:
        print("error: manifest has no tools", file=sys.stderr)
        return 2

    errors: list[str] = []
    cross_checked = 0
    skipped = 0

    for tool, platforms in tools.items():
        for platform, entry in platforms.items():
            sha_concrete = check_schema(
                tool,
                platform,
                entry,
                base_url,
                errors,
                allow_placeholders=args.allow_placeholders,
            )
            if not sha_concrete:
                continue
            if tool == "kubectl":
                if cross_check_kubectl(platform, entry, errors):
                    cross_checked += 1
            else:
                # gh and aws: no upstream cross-check today.
                skipped += 1

    if errors:
        print("tools.yaml verification FAILED:", file=sys.stderr)
        for e in errors:
            print(f"  - {e}", file=sys.stderr)
        return 1

    print(f"tools.yaml OK ({cross_checked} cross-checked, {skipped} schema-only)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
