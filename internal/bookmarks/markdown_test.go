package bookmarks

import (
	"strings"
	"testing"
)

func TestMarkdownRoundTrip(t *testing.T) {
	store := NewStore()
	work := NewFolder("Work")
	work.Tags = []string{"team"}
	bookmark := NewBookmark("Go", "https://go.dev", []string{"docs", "lang"})
	work.Children = append(work.Children, bookmark)
	store.Root.Children = append(store.Root.Children, work)

	exported := ExportMarkdown(store)
	if !strings.HasPrefix(exported, "# Branchmark bookmark tree v1\n") {
		t.Fatalf("export header=%q", strings.SplitN(exported, "\n", 2)[0])
	}
	if !strings.Contains(exported, "- space: Work") {
		t.Fatalf("export missing space: %s", exported)
	}
	if !strings.Contains(exported, "[Go](https://go.dev) {tags=docs,lang}") {
		t.Fatalf("export missing bookmark metadata: %s", exported)
	}

	imported, err := ImportMarkdown(exported)
	if err != nil {
		t.Fatal(err)
	}
	if got := CountBookmarks(imported.Root); got != 1 {
		t.Fatalf("bookmarks=%d, want 1", got)
	}
	if imported.Root.Children[0].Title != "Work" {
		t.Fatalf("folder title=%q", imported.Root.Children[0].Title)
	}
	importedBookmark := imported.Root.Children[0].Children[0]
	if importedBookmark.Title != "Go" || importedBookmark.URL != "https://go.dev" {
		t.Fatalf("unexpected bookmark: %+v", importedBookmark)
	}
}

func TestImportRejectsIndentJump(t *testing.T) {
	_, err := ImportMarkdown("- folder: A\n      - [B](https://example.com)\n")
	if err == nil {
		t.Fatal("expected indentation error")
	}
}

func TestImportSpaceSyntax(t *testing.T) {
	store, err := ImportMarkdown("- space: Work\n  - folder: CMS\n    - [Manage](https://example.com)\n")
	if err != nil {
		t.Fatal(err)
	}
	if store.Root.Children[0].Title != "Work" {
		t.Fatalf("space title=%q", store.Root.Children[0].Title)
	}
	if got := CountBookmarks(store.Root); got != 1 {
		t.Fatalf("bookmarks=%d, want 1", got)
	}
}

func TestMergeStoreMergesMatchingFolders(t *testing.T) {
	current := NewStore()
	work := NewFolder("Work")
	work.Tags = []string{"existing"}
	tools := NewFolder("Tools")
	tools.Children = append(tools.Children, NewBookmark("Existing Tool", "https://tool.example", nil))
	work.Children = append(work.Children, NewBookmark("Existing", "https://existing.example", nil), tools)
	personal := NewFolder("Personal")
	current.Root.Children = append(current.Root.Children, work, personal)

	imported, err := ImportMarkdown(strings.TrimSpace(`
- folder: Work {tags=docs}
  - [Go](https://go.dev)
  - folder: Tools
    - [Example](https://example.com)
- folder: Personal
  - [Home](https://home.example)
`))
	if err != nil {
		t.Fatal(err)
	}

	merged := MergeStore(current, imported)
	if len(merged.Root.Children) != 2 {
		t.Fatalf("top-level children=%d, want 2", len(merged.Root.Children))
	}
	mergedWork := findChildFolderByTitle(merged.Root, "Work")
	if mergedWork == nil {
		t.Fatal("Work folder was not found")
	}
	if strings.Join(mergedWork.Tags, ",") != "existing,docs" {
		t.Fatalf("work tags=%v, want existing,docs", mergedWork.Tags)
	}
	mergedTools := findChildFolderByTitle(mergedWork, "Tools")
	if mergedTools == nil {
		t.Fatal("Tools folder was not merged")
	}
	if got := CountBookmarks(merged.Root); got != 5 {
		t.Fatalf("bookmarks=%d, want 5", got)
	}
}

func TestNormalizeKeepsRootID(t *testing.T) {
	store := Normalize(NewStore())
	if store.Root.ID != "root" {
		t.Fatalf("root id=%q, want root", store.Root.ID)
	}
}
