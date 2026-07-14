package bookmarks

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func ExportMarkdown(store *Store) string {
	store = Normalize(store)
	var b strings.Builder
	b.WriteString("# Branchmark bookmark tree v1\n\n")
	b.WriteString("<!--\n")
	b.WriteString("Format:\n")
	b.WriteString("- space: Name {tags=tag1,tag2}\n")
	b.WriteString("  - folder: Folder\n")
	b.WriteString("    - [Title](https://example.com) {tags=tag1,tag2}\n")
	b.WriteString("-->\n\n")
	for _, child := range store.Root.Children {
		writeNodeMarkdown(&b, child, 0)
	}
	return b.String()
}

func writeNodeMarkdown(b *strings.Builder, n *Node, depth int) {
	indent := strings.Repeat("  ", depth)
	switch n.Type {
	case NodeFolder:
		kind := "folder"
		if depth == 0 {
			kind = "space"
		}
		fmt.Fprintf(b, "%s- %s: %s%s\n", indent, kind, n.Title, formatMeta(n))
		for _, child := range n.Children {
			writeNodeMarkdown(b, child, depth+1)
		}
	case NodeBookmark:
		fmt.Fprintf(b, "%s- [%s](%s)%s\n", indent, n.Title, n.URL, formatMeta(n))
	}
}

func ImportMarkdown(markdown string) (*Store, error) {
	store := NewStore()
	stack := []*Node{store.Root}
	lines := strings.Split(markdown, "\n")
	inComment := false
	for lineNo, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "<!--") {
			inComment = true
		}
		if inComment {
			if strings.Contains(trimmed, "-->") {
				inComment = false
			}
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		depth, content, ok := parseListLine(raw)
		if !ok {
			continue
		}
		if depth+1 > len(stack) {
			return nil, fmt.Errorf("line %d: indentation jumps more than one level", lineNo+1)
		}
		stack = stack[:depth+1]
		parent := stack[len(stack)-1]
		if strings.HasPrefix(content, "folder:") || strings.HasPrefix(content, "space:") {
			_, titleRaw, _ := strings.Cut(content, ":")
			title, tags := splitMeta(strings.TrimSpace(titleRaw))
			folder := NewFolder(title)
			folder.Tags = tags
			parent.Children = append(parent.Children, folder)
			stack = append(stack, folder)
			continue
		}
		if strings.HasPrefix(content, "[") {
			title, bookmarkURL, rest, err := parseMarkdownLink(content)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
			}
			if _, err := url.ParseRequestURI(bookmarkURL); err != nil {
				return nil, fmt.Errorf("line %d: invalid URL %q", lineNo+1, bookmarkURL)
			}
			_, tags := splitMeta(rest)
			node := NewBookmark(title, bookmarkURL, tags)
			parent.Children = append(parent.Children, node)
		}
	}
	return Normalize(store), nil
}

func parseListLine(raw string) (int, string, bool) {
	spaces := 0
	for _, r := range raw {
		if r != ' ' {
			break
		}
		spaces++
	}
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "- ") {
		return 0, "", false
	}
	return spaces / 2, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), true
}

func parseMarkdownLink(content string) (string, string, string, error) {
	endTitle := strings.Index(content, "](")
	endURL := strings.LastIndex(content, ")")
	if !strings.HasPrefix(content, "[") || endTitle < 0 || endURL < endTitle {
		return "", "", "", fmt.Errorf("invalid markdown link")
	}
	title := content[1:endTitle]
	link := content[endTitle+2 : endURL]
	rest := strings.TrimSpace(content[endURL+1:])
	return title, link, rest, nil
}

func splitMeta(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	if !strings.HasSuffix(raw, "}") {
		return raw, nil
	}
	start := strings.LastIndex(raw, " {")
	if start < 0 {
		if !strings.HasPrefix(raw, "{") {
			return raw, nil
		}
		start = -1
	}
	body := ""
	metaStart := 1
	if start >= 0 {
		body = strings.TrimSpace(raw[:start])
		metaStart = start + 2
	}
	meta := strings.TrimSuffix(raw[metaStart:], "}")
	var tags []string
	for _, field := range strings.Fields(meta) {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "tags":
			tags = SplitTags(value)
		}
	}
	return body, tags
}

func formatMeta(n *Node) string {
	parts := []string{}
	if len(n.Tags) > 0 {
		tags := append([]string(nil), n.Tags...)
		sort.Strings(tags)
		parts = append(parts, "tags="+strings.Join(tags, ","))
	}
	if len(parts) == 0 {
		return ""
	}
	return " {" + strings.Join(parts, " ") + "}"
}
