#!/usr/bin/env python3
"""Validate scanner output against the devpilot-scanning-repos Finding schema.

Usage:
    cat findings.json | python3 scripts/check-findings.py
    python3 scripts/check-findings.py findings.json

Exits 0 if every finding is well-formed, non-zero otherwise. On failure, prints
the index of the bad finding, the field that's wrong, and the offending object.
The agent should re-emit (or drop) the failing finding before scoring.
"""

from __future__ import annotations

import json
import sys
from typing import Any

REQUIRED_FIELDS = (
    "category",
    "title",
    "severity",
    "file",
    "line_range",
    "evidence",
    "why_it_matters",
    "suggested_fix",
)
VALID_CATEGORIES = {"security", "edge-case", "coverage"}
VALID_SEVERITIES = {"high", "medium", "low"}
MAX_TITLE_LEN = 80


def check(finding: Any, idx: int) -> list[str]:
    errs: list[str] = []
    if not isinstance(finding, dict):
        return [f"[{idx}] not a JSON object"]

    for field in REQUIRED_FIELDS:
        if field not in finding:
            errs.append(f"[{idx}] missing required field '{field}'")

    cat = finding.get("category")
    if cat is not None and cat not in VALID_CATEGORIES:
        errs.append(f"[{idx}] category='{cat}' not in {sorted(VALID_CATEGORIES)}")

    sev = finding.get("severity")
    if sev is not None and sev not in VALID_SEVERITIES:
        errs.append(f"[{idx}] severity='{sev}' not in {sorted(VALID_SEVERITIES)}")

    title = finding.get("title")
    if isinstance(title, str):
        if not title.strip():
            errs.append(f"[{idx}] title is empty")
        elif len(title) > MAX_TITLE_LEN:
            errs.append(f"[{idx}] title length {len(title)} > {MAX_TITLE_LEN}")

    evidence = finding.get("evidence")
    if evidence is not None and (not isinstance(evidence, str) or not evidence.strip()):
        errs.append(f"[{idx}] evidence must be a non-empty string (speculation without code = drop)")

    file_ = finding.get("file")
    if isinstance(file_, str) and file_.startswith("/"):
        errs.append(f"[{idx}] file must be repo-relative, got absolute path '{file_}'")

    lr = finding.get("line_range")
    if isinstance(lr, str) and not lr.startswith("L"):
        errs.append(f"[{idx}] line_range should look like 'L12-L34', got '{lr}'")

    suggested = finding.get("suggested_fix")
    if "suggested_fix" in finding and suggested is not None and not isinstance(suggested, str):
        errs.append(f"[{idx}] suggested_fix must be a string or null")

    return errs


def main() -> int:
    if len(sys.argv) > 1 and sys.argv[1] != "-":
        with open(sys.argv[1], encoding="utf-8") as fh:
            raw = fh.read()
    else:
        raw = sys.stdin.read()

    try:
        findings = json.loads(raw)
    except json.JSONDecodeError as e:
        print(f"ERROR: input is not valid JSON: {e}", file=sys.stderr)
        return 2

    if not isinstance(findings, list):
        print("ERROR: top-level JSON must be an array of Finding objects", file=sys.stderr)
        return 2

    all_errs: list[str] = []
    for i, f in enumerate(findings):
        all_errs.extend(check(f, i))

    if all_errs:
        print("INVALID findings:", file=sys.stderr)
        for e in all_errs:
            print(f"  - {e}", file=sys.stderr)
        print(f"\n{len(all_errs)} error(s) across {len(findings)} finding(s)", file=sys.stderr)
        return 1

    print(f"OK: {len(findings)} finding(s) validated")
    return 0


if __name__ == "__main__":
    sys.exit(main())
