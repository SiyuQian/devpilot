#!/usr/bin/env python3
"""Fetch RSS news feeds and output structured text by category."""

import json
import re
import sys
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime

import feedparser


ITEMS_PER_CATEGORY = 6
FETCH_TIMEOUT = 10
DEDUP_PREFIX_LEN = 50


def load_feeds(path):
    """Load feed configuration from JSON file."""
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)["categories"]


def fetch_single_feed(name, url):
    """Fetch and parse a single RSS feed. Returns (name, entries) or (name, [])."""
    try:
        feed = feedparser.parse(url, request_headers={"User-Agent": "DailyNewsBot/1.0"})
        if feed.bozo and not feed.entries:
            print(f"[WARN] Failed to parse {name}: {feed.bozo_exception}", file=sys.stderr)
            return name, []
        entries = []
        for entry in feed.entries:
            published = None
            for date_field in ("published_parsed", "updated_parsed"):
                parsed = getattr(entry, date_field, None)
                if parsed:
                    try:
                        published = datetime(*parsed[:6])
                    except Exception:
                        pass
                    break
            title = getattr(entry, "title", "").strip()
            raw_summary = re.sub(r"<[^>]+>", "", getattr(entry, "summary", ""))
            summary = re.sub(r"&\w+;", " ", raw_summary).strip()
            link = getattr(entry, "link", "")
            if title:
                entries.append({
                    "title": title,
                    "summary": summary[:500],
                    "link": link,
                    "published": published,
                    "source": name,
                })
        return name, entries
    except Exception as e:
        print(f"[WARN] Error fetching {name}: {e}", file=sys.stderr)
        return name, []


def deduplicate(entries):
    """Remove duplicate entries based on lowercase title prefix."""
    seen = set()
    result = []
    for entry in entries:
        key = entry["title"].lower().strip()[:DEDUP_PREFIX_LEN]
        if key not in seen:
            seen.add(key)
            result.append(entry)
    return result


def fetch_all(categories):
    """Fetch all feeds concurrently and return entries grouped by category."""
    tasks = []
    for category, feeds in categories.items():
        for feed in feeds:
            tasks.append((category, feed["name"], feed["url"]))

    raw = {cat: [] for cat in categories}
    with ThreadPoolExecutor(max_workers=10) as pool:
        futures = {}
        for category, name, url in tasks:
            future = pool.submit(fetch_single_feed, name, url)
            futures[future] = category

        for future in as_completed(futures):
            category = futures[future]
            try:
                _, entries = future.result(timeout=FETCH_TIMEOUT + 5)
                raw[category].extend(entries)
            except Exception as e:
                print(f"[WARN] Future failed for {category}: {e}", file=sys.stderr)

    result = {}
    for category, entries in raw.items():
        entries.sort(key=lambda e: e["published"] or datetime.min, reverse=True)
        entries = deduplicate(entries)
        result[category] = entries[:ITEMS_PER_CATEGORY]

    return result


def format_output(results, categories):
    """Format results as structured text, preserving category order from config."""
    lines = []
    for category in categories:
        entries = results.get(category, [])
        lines.append(f"## {category}")
        lines.append("")
        if not entries:
            lines.append("(No articles fetched for this category)")
            lines.append("")
            continue
        for i, entry in enumerate(entries, 1):
            date_str = entry["published"].strftime("%Y-%m-%d") if entry["published"] else "unknown"
            lines.append(f"{i}. [{entry['source']} | {date_str}] {entry['title']}")
            if entry["summary"]:
                lines.append(f"   {entry['summary']}")
            lines.append("")
    return "\n".join(lines)


def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    feeds_path = os.path.join(script_dir, "..", "feeds.json")

    categories = load_feeds(feeds_path)
    results = fetch_all(categories)

    total = sum(len(v) for v in results.values())
    if total == 0:
        print("ERROR: All feeds failed to fetch.", file=sys.stderr)
        sys.exit(1)

    print(format_output(results, categories))


if __name__ == "__main__":
    main()
