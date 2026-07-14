# Branchmark bookmark tree v1

<!--
Merge import template.

Usage:
./brmk import examples/merge-import.md --merge

Rules:
- Top-level folders are spaces. Matching spaces/folders at the same tree level are merged.
- Nested folders use "- folder: Name".
- Bookmarks use "- [Title](https://example.com)".
- Optional metadata uses "{tags=tag1,tag2}".
- Bookmark URLs are appended as-is; remove duplicate URLs from this file before importing.
-->

- space: Work {tags=team}
  - folder: Docs {tags=reference}
    - [Example Docs](https://example.com/docs) {tags=docs}
  - folder: Tools
    - [Example Tool](https://example.com/tools) {tags=tool}
- space: Personal
  - folder: Read Later
    - [Example Article](https://example.com/article) {tags=readlater}
