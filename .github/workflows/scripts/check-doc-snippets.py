#!/usr/bin/env python3
"""Check that the configuration snippets of Markdown guides match their source files.

A snippet declares its source with one or more HTML comments on the lines right before its code fence:

    <!-- snippet-source: file=<path> path=<yq path> [snippet-path=<yq path>] mode=<exact|subset|adapted> sha=<hash> -->

- file: the source file, relative to the repository root.
- path: the node of the source file the snippet quotes, in a yq-like syntax: `.a.b`, `.a["c/d"]`, `.a[0]`, and
  `.env[name=X]` for the list item whose `name` is X. `.` is the whole document. Point to the smallest node the
  snippet quotes, because any change to that node asks for a review of the snippet.
- snippet-path: the node of the snippet to compare, `.` by default.
- mode:
  - exact: the snippet node equals the source node.
  - subset: every key, and every list item, of the snippet node is in the source node.
  - adapted: the snippet is a rewrite of the source node, which this script can't compare.
- sha: a hash of the source node when the snippet was last reviewed. When the source node changes, even in a way
  that the snippet still matches, the check fails until someone reviews the snippet and runs this script with
  --update-hashes, which stores the new hashes. It doesn't update the hash of a snippet that doesn't match its source.

In exact and subset modes, a snippet string such as `<CLUSTER_NAME>` is a placeholder and matches any value.
Comments and formatting are ignored: the script compares the parsed YAML.

Usage:
  check-doc-snippets.py [--update-hashes] GUIDE.md...   check the snippets; exits with status 1 on a problem
  check-doc-snippets.py --quoted FILE GUIDE.md...       list the snippets that quote FILE, before changing it
"""
import argparse
import hashlib
import json
import os
import pathlib
import re
import subprocess
import sys

import yaml

MARKER = re.compile(r"^\s*<!--\s*snippet-source:\s*(?P<attrs>.*?)\s*-->\s*$")
FENCE_OPEN = re.compile(r"^(\s*)```(\w*)\s*$")
HEADING_RULE_ID = re.compile(r"^#+\s+([A-Z]+-\d+):")
PLACEHOLDER = re.compile(r"^<[^<>]+>$")
PATH_TOKEN = re.compile(r'\.([A-Za-z_][\w-]*)|\["((?:[^"\\]|\\.)*)"\]|\[(\d+)\]|\[name=([^\]]+)\]|\.')


def repo_root():
    out = subprocess.run(["git", "rev-parse", "--show-toplevel"], capture_output=True, text=True, check=True)
    return pathlib.Path(out.stdout.strip())


def parse_path(expr):
    """Parse a yq-like path into a list of keys and indexes."""
    if not expr.startswith("."):
        raise ValueError(f"path must start with '.': {expr}")
    tokens, pos = [], 0
    while pos < len(expr):
        m = PATH_TOKEN.match(expr, pos)
        if not m or m.end() == pos:
            raise ValueError(f"invalid path {expr!r} at {expr[pos:]!r}")
        if m.group(1) is not None:
            tokens.append(m.group(1))
        elif m.group(2) is not None:
            tokens.append(json.loads(f'"{m.group(2)}"'))
        elif m.group(3) is not None:
            tokens.append(int(m.group(3)))
        elif m.group(4) is not None:
            tokens.append(("name", m.group(4)))
        pos = m.end()
    return tokens


def select(node, expr):
    for token in parse_path(expr):
        if isinstance(token, tuple):
            matches = [item for item in node if isinstance(item, dict) and item.get("name") == token[1]] \
                if isinstance(node, list) else []
            if len(matches) != 1:
                raise KeyError(f"{expr}: {len(matches)} list items named {token[1]!r}")
            node = matches[0]
            continue
        if isinstance(token, int):
            if not isinstance(node, list) or token >= len(node):
                raise KeyError(f"{expr}: no item {token}")
        elif not isinstance(node, dict) or token not in node:
            raise KeyError(f"{expr}: no key {token!r}")
        node = node[token]
    return node


def is_placeholder(value):
    return isinstance(value, str) and bool(PLACEHOLDER.match(value))


def compare(snippet, source, exact, where="."):
    """Return the differences between the snippet node and the source node."""
    if is_placeholder(snippet):
        return []
    if isinstance(snippet, dict):
        if not isinstance(source, dict):
            return [f"{where}: the snippet has a map, the source has {source!r}"]
        problems = []
        for key, value in snippet.items():
            if key not in source:
                problems.append(f"{where}: key {key!r} isn't in the source")
            else:
                problems += compare(value, source[key], exact, f"{where}.{key}" if where != "." else f".{key}")
        if exact:
            problems += [f"{where}: the source also has key {key!r}" for key in source if key not in snippet]
        return problems
    if isinstance(snippet, list):
        if not isinstance(source, list):
            return [f"{where}: the snippet has a list, the source has {source!r}"]
        if exact:
            if len(snippet) != len(source):
                return [f"{where}: the snippet has {len(snippet)} items, the source has {len(source)}"]
            problems = []
            for i, (s, o) in enumerate(zip(snippet, source)):
                problems += compare(s, o, exact, f"{where}[{i}]")
            return problems
        return [
            f"{where}: item {item!r} isn't in the source"
            for item in snippet
            if not any(not compare(item, candidate, exact) for candidate in source)
        ]
    if snippet != source:
        return [f"{where}: the snippet has {snippet!r}, the source has {source!r}"]
    return []


def source_hash(node):
    canonical = json.dumps(node, sort_keys=True, separators=(",", ":"), default=str)
    return hashlib.sha256(canonical.encode()).hexdigest()[:12]


def parse_attrs(text):
    attrs = {}
    for m in re.finditer(r'([\w-]+)=("([^"]*)"|\S+)', text):
        attrs[m.group(1)] = m.group(3) if m.group(3) is not None else m.group(2)
    return attrs


def check_guide(root, guide, update_hashes, quoted=None):
    """Check the snippets of a guide, or, with quoted, list the snippets that quote one of these files."""
    lines = guide.read_text().split("\n")
    guide_ref = guide.resolve().relative_to(root)
    problems, checked, rule_id = [], 0, None
    i = 0
    while i < len(lines):
        heading = HEADING_RULE_ID.match(lines[i])
        if heading:
            rule_id = heading.group(1)
        elif lines[i].startswith("#"):
            rule_id = None
        fence = FENCE_OPEN.match(lines[i])
        if not fence:
            i += 1
            continue
        end = i + 1
        while end < len(lines) and lines[end].strip() != "```":
            end += 1
        markers = []
        k = i - 1
        while k >= 0 and MARKER.match(lines[k]):
            markers.insert(0, k)
            k -= 1
        if markers:
            indent = len(fence.group(1))
            body = "\n".join(line[indent:] for line in lines[i + 1 : end])
            location = f"{guide_ref}:{i + 1}" + (f" ({rule_id})" if rule_id else "")
            try:
                snippet_doc = yaml.safe_load(body)
            except yaml.YAMLError as err:
                problems.append(f"{location}: the snippet isn't valid YAML: {err}")
                snippet_doc = None
            for k in markers:
                attrs = parse_attrs(MARKER.match(lines[k]).group("attrs"))
                missing = [a for a in ("file", "path", "mode") if a not in attrs]
                if missing or attrs["mode"] not in ("exact", "subset", "adapted"):
                    problems.append(f"{location}: invalid marker, missing {missing} or bad mode: {lines[k].strip()}")
                    continue
                source_file = root / attrs["file"]
                label = f"{location}, source {attrs['file']} {attrs['path']}"
                if quoted is not None:
                    if attrs["file"] in quoted:
                        problems.append(f"{location}: {attrs['path']} ({attrs['mode']})")
                    continue
                if not source_file.is_file():
                    problems.append(f"{label}: the source file doesn't exist")
                    continue
                source_text = source_file.read_text()
                try:
                    source_node = select(yaml.safe_load(source_text), attrs["path"])
                except KeyError as err:
                    problems.append(f"{label}: the source path doesn't exist: {err}")
                    continue
                checked += 1
                mismatches = []
                if attrs["mode"] != "adapted":
                    if snippet_doc is None:
                        continue
                    try:
                        snippet_node = select(snippet_doc, attrs.get("snippet-path", "."))
                    except KeyError as err:
                        problems.append(f"{label}: the snippet path doesn't exist: {err}")
                        continue
                    mismatches = compare(snippet_node, source_node, attrs["mode"] == "exact")
                    problems += [f"{label}: {p}" for p in mismatches]
                current = source_hash(source_node)
                if attrs.get("sha") != current:
                    if update_hashes and not mismatches:
                        lines[k] = re.sub(r"\s*sha=\S+", "", lines[k]).replace(" -->", f" sha={current} -->")
                    elif not mismatches:
                        problems.append(
                            f"{label}: the source changed since the snippet was last reviewed "
                            f"(sha {attrs.get('sha')} -> {current}). Review the snippet, then run with --update-hashes"
                        )
        i = end + 1
    if update_hashes:
        guide.write_text("\n".join(lines))
    return problems, checked


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("--update-hashes", action="store_true", help="store the current source hashes after a review")
    parser.add_argument("--quoted", action="append", type=pathlib.Path, metavar="FILE",
                        help="list the snippets that quote FILE instead of checking them; can be repeated")
    parser.add_argument("guides", nargs="+", type=pathlib.Path)
    args = parser.parse_args()
    root = repo_root()
    if args.quoted:
        quoted = {str(f.resolve().relative_to(root)) for f in args.quoted}
        found = []
        for guide in args.guides:
            found += check_guide(root, guide, False, quoted)[0]
        print("\n".join(found) if found else f"No snippet quotes {', '.join(sorted(quoted))}.")
        return 0
    all_problems, total = [], 0
    for guide in args.guides:
        problems, checked = check_guide(root, guide, args.update_hashes)
        all_problems += problems
        total += checked
    for problem in all_problems:
        print(f"::error::{problem}" if "GITHUB_ACTIONS" in os.environ else problem)
    print(f"Checked {total} snippet sources: {len(all_problems)} problems.")
    return 1 if all_problems else 0


if __name__ == "__main__":
    sys.exit(main())
