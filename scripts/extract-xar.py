#!/usr/bin/env python3
"""Minimal xar archive extractor.

Used by .github/workflows/release-tools.yml to crack the macOS aws-cli
.pkg (a xar archive) on Linux runners. Ubuntu dropped the `xar` package
from apt and bsdtar's xar support on Ubuntu mangles output paths, so
extraction goes through this reader instead.

Only supports what the aws-cli .pkg uses today: gzip-encoded file data
and plain directory entries. No signature verification (the aws-cli
download is already SHA-pinned downstream).

Usage: extract-xar.py <archive.pkg> <dest-dir>
"""

from __future__ import annotations

import struct
import sys
import zlib
from pathlib import Path
from xml.etree import ElementTree as ET


HEADER_STRUCT = ">4sHHQQL"  # magic, size, version, toc_comp, toc_uncomp, cksum_alg
HEADER_LEN = struct.calcsize(HEADER_STRUCT)


def extract(pkg_path: Path, dest: Path) -> None:
    dest.mkdir(parents=True, exist_ok=True)
    with pkg_path.open("rb") as fh:
        header = fh.read(HEADER_LEN)
        magic, _size, _version, toc_comp, _toc_uncomp, _cksum = struct.unpack(
            HEADER_STRUCT, header
        )
        if magic != b"xar!":
            raise SystemExit(f"not a xar archive: {pkg_path}")
        toc_xml = zlib.decompress(fh.read(toc_comp))
        heap_offset = fh.tell()
        root = ET.fromstring(toc_xml)
        toc = root.find("toc")
        if toc is None:
            raise SystemExit("xar: missing <toc>")

        def walk(elem: ET.Element, prefix: Path) -> None:
            for file_elem in elem.findall("file"):
                name_elem = file_elem.find("name")
                if name_elem is None or name_elem.text is None:
                    continue
                target = prefix / name_elem.text
                ftype = file_elem.findtext("type", default="file")
                data_elem = file_elem.find("data")
                if ftype == "directory":
                    target.mkdir(parents=True, exist_ok=True)
                elif data_elem is not None:
                    offset = int(data_elem.findtext("offset", "0"))
                    length = int(data_elem.findtext("length", "0"))
                    encoding_elem = data_elem.find("encoding")
                    style = (
                        encoding_elem.attrib.get("style", "")
                        if encoding_elem is not None
                        else ""
                    )
                    fh.seek(heap_offset + offset)
                    raw = fh.read(length)
                    if "gzip" in style or "deflate" in style:
                        payload = zlib.decompress(raw)
                    elif style in ("", "application/octet-stream"):
                        payload = raw
                    else:
                        raise SystemExit(f"xar: unsupported encoding {style!r}")
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_bytes(payload)
                walk(file_elem, target)

        walk(toc, dest)


def main(argv: list[str]) -> int:
    if len(argv) != 3:
        print(__doc__, file=sys.stderr)
        return 2
    extract(Path(argv[1]), Path(argv[2]))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
