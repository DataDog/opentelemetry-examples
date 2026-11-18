#!/usr/bin/env python3
"""Check that the configuration snippets of Markdown guides match their source files.

A snippet declares its source with one or more HTML comments on the lines right before its code fence:

    <!-- snippet-source: file=<path> path=<yq path> [snippet-path=<yq path>] mode=<exact|subset|adapted> [sha=<hash>] -->

- file: the source file, relative to the repository root.
- path: the node of the source file the snippet quotes, in a yq-like syntax: `.a.b`, `.a["c/d"]`, `.a[0]`.
  `.` is the whole document.
- snippet-path: the node of the snippet to compare, `.` by default.
- mode:
  - exact: the snippet node equals the source node.
  - subset: every key, and every list item, of the snippet node is in the source node.
  - adapted: the snippet is a rewrite of the source node, which this script can't compare. The marker stores a hash
    of the source node: when the source node changes, the hash no longer matches and the snippet needs a review.
    After reviewing the snippet, run this script with --update-hashes to store the new hash.

In exact and subset modes, a snippet string such as `<CLUSTER_NAME>` is a placeholder and matches any value.
Comments and formatting are ignored: the script compares the parsed YAML.

Each source file must also point back to the guide, with a comment such as
`# Quoted in ddot/ADAPT-EXISTING-CONFIG.md (S-01, S-08)` that names the guide and the rule ID of the snippet, so that
whoever changes the source knows which snippets to update.

Usage: check-doc-snippets.py [--update-hashes] GUIDE.md...
Exits with status 1 when a snippet doesn't match its source.
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
PATH_TOKEN = re.compile(r'\.([A-Za-z_][\w-]*)|\["((?:[^"\\]|\\.)*)"\]|\[(\d+)\]|\.')


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
        pos = m.end()
    return tokens


def select(node, expr):
    for token in parse_path(expr):
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


def back_reference_found(source_text, guide_name, rule_id):
    for line in source_text.splitlines():
        if "#" in line and guide_name in line and "Quoted in" in line:
            if rule_id is None or re.search(rf"\b{re.escape(rule_id)}\b", line):
                return True
    return False


def check_guide(root, guide, update_hashes):
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
                if not source_file.is_file():
                    problems.append(f"{label}: the source file doesn't exist")
                    continue
                source_text = source_file.read_text()
                try:
                    source_node = select(yaml.safe_load(source_text), attrs["path"])
                except KeyError as err:
                    problems.append(f"{label}: the source path doesn't exist: {err}")
                    continue
                if not back_reference_found(source_text, guide.name, rule_id):
                    problems.append(
                        f"{label}: the source file has no back-reference comment naming {guide.name}"
                        + (f" and {rule_id}" if rule_id else "")
                        + f", such as '# Quoted in {guide.name} ({rule_id or '<rule ID>'})'"
                    )
                checked += 1
                if attrs["mode"] == "adapted":
                    current = source_hash(source_node)
                    if attrs.get("sha") != current:
                        if update_hashes:
                            lines[k] = re.sub(r"\s*sha=\S+", "", lines[k]).replace(" -->", f" sha={current} -->")
                        else:
                            problems.append(
                                f"{label}: the source changed since the snippet was last reviewed "
                                f"(sha {attrs.get('sha')} -> {current}). Review the snippet, then run with --update-hashes"
                            )
                    continue
                if snippet_doc is None:
                    continue
                try:
                    snippet_node = select(snippet_doc, attrs.get("snippet-path", "."))
                except KeyError as err:
                    problems.append(f"{label}: the snippet path doesn't exist: {err}")
                    continue
                problems += [f"{label}: {p}" for p in compare(snippet_node, source_node, attrs["mode"] == "exact")]
        i = end + 1
    if update_hashes:
        guide.write_text("\n".join(lines))
    return problems, checked


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("--update-hashes", action="store_true", help="store the current source hash in adapted markers")
    parser.add_argument("guides", nargs="+", type=pathlib.Path)
    args = parser.parse_args()
    root = repo_root()
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
