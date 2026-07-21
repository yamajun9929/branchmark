package tui

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
)

func TestRowsRenderAsTree(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	goBookmark := bookmarks.NewBookmark("Go", "https://go.dev", []string{"docs"})
	work.Children = append(work.Children,
		goBookmark,
		bookmarks.NewBookmark("Example", "https://example.com", nil),
	)
	personal := bookmarks.NewFolder("Personal")
	store.Root.Children = append(store.Root.Children, work, personal)

	a := &app{store: store}
	a.rebuildRows()

	lines := formattedRows(a.treeRows)
	want := []string{
		"▾  Work",
		"├──  Go",
		"└── 󰖟 Example",
	}
	for i, prefix := range want {
		if i >= len(lines) || !strings.HasPrefix(lines[i], prefix) {
			t.Fatalf("line %d=%q, want prefix %q\nall lines:\n%s", i, lineAt(lines, i), prefix, strings.Join(lines, "\n"))
		}
	}
}

func TestFilteredRowsKeepAncestors(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	tools := bookmarks.NewFolder("Tools")
	tools.Children = append(tools.Children, bookmarks.NewBookmark("Go", "https://go.dev", []string{"docs"}))
	work.Children = append(work.Children, tools)
	store.Root.Children = append(store.Root.Children, work)

	a := &app{store: store, filter: "go"}
	a.rebuildRows()

	lines := formattedRows(a.treeRows)
	want := []string{
		"▾  Work",
		"└── ▾  Tools",
		"    └──  Go",
	}
	if len(lines) != len(want) {
		t.Fatalf("lines=%d, want %d\n%s", len(lines), len(want), strings.Join(lines, "\n"))
	}
	for i, prefix := range want {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Fatalf("line %d=%q, want prefix %q\nall lines:\n%s", i, lines[i], prefix, strings.Join(lines, "\n"))
		}
	}
}

func TestDisplayWidthHandlesJapaneseAndFullWidthCharacters(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"日本語", 6},
		{"タイトル（仮）", 14},
		{"ＡＢＣ１２３", 12},
		{"。、！？", 8},
		{"半角ｶﾅ", 6},
		{"Go🚀", 4},
		{"e\u0301", 1},
	}
	for _, tt := range cases {
		if got := displayWidth(tt.text); got != tt.want {
			t.Fatalf("displayWidth(%q)=%d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestJapaneseTitleRowKeepsRequestedWidth(t *testing.T) {
	bookmark := bookmarks.NewBookmark("日本語タイトル（仮）", "https://example.com", nil)
	item := treeRow{node: bookmark, last: true}
	line := "  " + formatRow(item)
	padded := padRight(truncateWidth(line, 24), 24)

	if got := displayWidth(padded); got != 24 {
		t.Fatalf("displayWidth(treeRow)=%d, want 24: %q", got, padded)
	}
	if displayWidth(truncateWidth(line, 12)) > 12 {
		t.Fatalf("truncateWidth exceeded target: %q", truncateWidth(line, 12))
	}
}

func TestDrawKeepsHelpLineAtBottom(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WEZTERM_PANE", "")
	t.Setenv("WEZTERM_EXECUTABLE", "")

	store := bookmarks.NewStore()
	a := &app{dataPath: filepath.Join(t.TempDir(), "bookmarks.json"), store: store}
	a.rebuildRows()

	output := captureStdout(t, func() {
		a.draw(80, 12, ">", "saved", normalHelpLine)
	})
	lines := strings.Split(output, "\r\n")
	if len(lines) == 0 || !strings.Contains(lines[len(lines)-1], normalHelpLine) {
		t.Fatalf("last line does not contain help line:\n%s", output)
	}
	if !strings.Contains(output, "status: saved") {
		t.Fatalf("output missing status line:\n%s", output)
	}
}

func TestHelpPaneRendersAndToggles(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WEZTERM_PANE", "")
	t.Setenv("WEZTERM_EXECUTABLE", "")

	store := bookmarks.NewStore()
	a := &app{dataPath: filepath.Join(t.TempDir(), "bookmarks.json"), store: store}
	a.rebuildRows()

	a.handleKey(key{name: "rune", r: '?'})
	if !a.showHelp {
		t.Fatal("help was not shown")
	}
	output := captureStdout(t, func() {
		a.draw(120, 24, ">", "help shown", normalHelpLine)
	})
	if !strings.Contains(output, "Help  (?/esc close)") || !strings.Contains(output, "R               reload from disk") {
		t.Fatalf("output missing help pane:\n%s", output)
	}

	a.handleKey(key{name: "esc"})
	if a.showHelp {
		t.Fatal("help was not hidden")
	}
}

func TestPromptEditorMovesCursorAndEditsInMiddle(t *testing.T) {
	editor := newPromptEditor("abc")
	editor.apply(key{name: "left"})
	editor.apply(key{name: "left"})
	editor.apply(key{name: "rune", r: 'X'})
	if got := editor.value(); got != "aXbc" {
		t.Fatalf("value=%q, want aXbc", got)
	}
	if editor.cursor != 2 {
		t.Fatalf("cursor=%d, want 2", editor.cursor)
	}
	editor.apply(key{name: "delete"})
	if got := editor.value(); got != "aXc" {
		t.Fatalf("value after delete=%q, want aXc", got)
	}
	editor.apply(key{name: "backspace"})
	if got := editor.value(); got != "ac" {
		t.Fatalf("value after backspace=%q, want ac", got)
	}
}

func TestPromptEditorSelectAllReplacesInput(t *testing.T) {
	editor := newPromptEditor("日本語 title")
	editor.apply(key{name: "ctrl-1"})
	if !editor.allSelected {
		t.Fatal("input was not selected")
	}
	editor.apply(key{name: "rune", r: 'X'})
	if got := editor.value(); got != "X" {
		t.Fatalf("value=%q, want X", got)
	}
	if editor.allSelected {
		t.Fatal("selection was not cleared after replacement")
	}
}

func TestPromptEditorHomeEndAndClear(t *testing.T) {
	editor := newPromptEditor("abc")
	editor.apply(key{name: "home"})
	editor.apply(key{name: "rune", r: 'X'})
	editor.apply(key{name: "end"})
	editor.apply(key{name: "rune", r: 'Y'})
	if got := editor.value(); got != "XabcY" {
		t.Fatalf("value=%q, want XabcY", got)
	}
	editor.apply(key{name: "ctrl-21"})
	if got := editor.value(); got != "" {
		t.Fatalf("value after clear=%q, want empty", got)
	}
}

func TestPromptEditorTabCompletesAndCycles(t *testing.T) {
	candidates := normalizeCompletions([]string{"private", "prod", "default"})
	editor := newPromptEditor("pr")

	if !editor.complete(candidates, 1) {
		t.Fatal("completion failed")
	}
	if got := editor.value(); got != "private" {
		t.Fatalf("value=%q, want private", got)
	}
	if !editor.complete(candidates, 1) {
		t.Fatal("second completion failed")
	}
	if got := editor.value(); got != "prod" {
		t.Fatalf("value=%q, want prod", got)
	}
	if !editor.complete(candidates, -1) {
		t.Fatal("reverse completion failed")
	}
	if got := editor.value(); got != "private" {
		t.Fatalf("value=%q, want private", got)
	}
}

func TestProfileNamesAreCompletionCandidates(t *testing.T) {
	a := &app{config: &bookmarks.Config{
		ActiveProfile: "default",
		BrowserProfiles: []bookmarks.BrowserProfile{
			{Name: "default", Kind: "default"},
			{Name: "work", Kind: "managed"},
		},
	}}
	if got := strings.Join(normalizeCompletions(a.profileNames()), ","); got != "default,work" {
		t.Fatalf("profile names=%q", got)
	}
}

func TestTabsRepresentTopLevelSpaces(t *testing.T) {
	store := bookmarks.NewStore()
	store.Root.Children = append(store.Root.Children, bookmarks.NewFolder("Work"), bookmarks.NewFolder("Personal"))
	a := &app{store: store, config: bookmarks.DefaultConfig()}
	a.rebuildRows()

	line := a.renderTabsLine(80)
	if !strings.Contains(line, "[Work]") || !strings.Contains(line, "Personal") || !strings.Contains(line, "profile=default") {
		t.Fatalf("unexpected tabs line: %q", line)
	}
	if _, ok := a.spaceIndexAtX(a.tabRanges[1].start); !ok {
		t.Fatal("space tab range was not recorded")
	}
}

func TestCreateSpaceCreatesTopLevelTabAndSelectsIt(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	store.Root.Children = append(store.Root.Children, work)
	dataPath := filepath.Join(t.TempDir(), "bookmarks.json")
	configPath := filepath.Join(t.TempDir(), "config.json")
	a := &app{
		dataPath:   dataPath,
		configPath: configPath,
		store:      store,
		config:     bookmarks.DefaultConfig(),
	}
	a.rebuildRows()

	a.createSpace(" Personal ")

	if len(store.Root.Children) != 2 || store.Root.Children[1].Title != "Personal" {
		t.Fatalf("top-level spaces=%v, want Work and Personal", childTitles(store.Root))
	}
	if active := a.activeSpace(); active == nil || active.Title != "Personal" {
		t.Fatalf("active space=%v, want Personal", active)
	}
	if selected := a.selected(); selected == nil || selected.node.Title != "Personal" {
		t.Fatalf("selected=%v, want Personal", selected)
	}
	saved, err := bookmarks.Load(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Root.Children) != 2 || saved.Root.Children[1].Title != "Personal" {
		t.Fatalf("saved top-level spaces=%v, want Work and Personal", childTitles(saved.Root))
	}
	config, err := bookmarks.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if config.DefaultSpace != "Personal" {
		t.Fatalf("default_space=%q, want Personal", config.DefaultSpace)
	}
}

func TestCreateSpaceRejectsDuplicateTopLevelName(t *testing.T) {
	store := bookmarks.NewStore()
	store.Root.Children = append(store.Root.Children, bookmarks.NewFolder("Work"))
	a := &app{store: store, dataPath: filepath.Join(t.TempDir(), "bookmarks.json")}
	a.rebuildRows()

	a.createSpace(" work ")

	if len(store.Root.Children) != 1 {
		t.Fatalf("top-level spaces=%d, want 1", len(store.Root.Children))
	}
	if a.status != "space already exists: Work" {
		t.Fatalf("status=%q, want duplicate-space message", a.status)
	}
}

func TestTabAndShiftTabSwitchSpaces(t *testing.T) {
	store := bookmarks.NewStore()
	one := bookmarks.NewFolder("One")
	two := bookmarks.NewFolder("Two")
	three := bookmarks.NewFolder("Three")
	store.Root.Children = append(store.Root.Children, one, two, three)
	a := &app{store: store, config: bookmarks.DefaultConfig()}
	a.rebuildRows()

	a.handleKey(key{name: "tab"})
	if active := a.activeSpace(); active == nil || active.ID != two.ID {
		t.Fatalf("active space=%v, want Two", active)
	}
	a.handleKey(key{name: "shift-tab"})
	if active := a.activeSpace(); active == nil || active.ID != one.ID {
		t.Fatalf("active space=%v, want One", active)
	}
	a.handleKey(key{name: "shift-tab"})
	if active := a.activeSpace(); active == nil || active.ID != three.ID {
		t.Fatalf("active space=%v, want Three", active)
	}
}

func TestJumpFolderMovesBetweenVisibleFolders(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	docs := bookmarks.NewFolder("Docs")
	tools := bookmarks.NewFolder("Tools")
	work.Children = append(work.Children,
		bookmarks.NewBookmark("Go", "https://go.dev", nil),
		docs,
		bookmarks.NewBookmark("Example", "https://example.com", nil),
		tools,
	)
	store.Root.Children = append(store.Root.Children, work)
	a := &app{store: store, config: bookmarks.DefaultConfig()}
	a.rebuildRows()

	a.cursor = 1
	a.jumpFolder(1)
	if selected := a.selected(); selected == nil || selected.node.ID != docs.ID {
		t.Fatalf("selected=%v, want Docs", selected)
	}
	a.jumpFolder(1)
	if selected := a.selected(); selected == nil || selected.node.ID != tools.ID {
		t.Fatalf("selected=%v, want Tools", selected)
	}
	a.jumpFolder(1)
	if selected := a.selected(); selected == nil || selected.node.ID != tools.ID {
		t.Fatalf("selected=%v, want Tools at bottom", selected)
	}
	if a.status != "last folder" {
		t.Fatalf("status=%q, want last folder", a.status)
	}
	a.jumpFolder(-1)
	if selected := a.selected(); selected == nil || selected.node.ID != docs.ID {
		t.Fatalf("selected=%v, want Docs", selected)
	}
	a.jumpFolder(-1)
	if selected := a.selected(); selected == nil || selected.node.ID != work.ID {
		t.Fatalf("selected=%v, want Work", selected)
	}
	a.jumpFolder(-1)
	if selected := a.selected(); selected == nil || selected.node.ID != work.ID {
		t.Fatalf("selected=%v, want Work at top", selected)
	}
	if a.status != "first folder" {
		t.Fatalf("status=%q, want first folder", a.status)
	}
}

func TestSelectConfiguredSpace(t *testing.T) {
	store := bookmarks.NewStore()
	one := bookmarks.NewFolder("One")
	two := bookmarks.NewFolder("Two")
	store.Root.Children = append(store.Root.Children, one, two)
	a := &app{
		store: store,
		config: &bookmarks.Config{
			ActiveProfile: "default",
			DefaultSpace:  "Two",
			BrowserProfiles: []bookmarks.BrowserProfile{
				{Name: "default", Kind: "default"},
			},
		},
	}

	a.selectConfiguredSpace()

	if active := a.activeSpace(); active == nil || active.ID != two.ID {
		t.Fatalf("active space=%v, want Two", active)
	}
}

func TestSwitchSpaceSavesDefaultSpace(t *testing.T) {
	store := bookmarks.NewStore()
	one := bookmarks.NewFolder("One")
	two := bookmarks.NewFolder("Two")
	store.Root.Children = append(store.Root.Children, one, two)
	configPath := filepath.Join(t.TempDir(), "config.json")
	a := &app{
		configPath: configPath,
		store:      store,
		config:     bookmarks.DefaultConfig(),
	}
	a.rebuildRows()

	a.handleKey(key{name: "tab"})

	cfg, err := bookmarks.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultSpace != "Two" {
		t.Fatalf("default_space=%q, want Two", cfg.DefaultSpace)
	}
}

func TestMouseClickOnTabSwitchesSpaceOnRelease(t *testing.T) {
	store := bookmarks.NewStore()
	one := bookmarks.NewFolder("Work")
	two := bookmarks.NewFolder("Personal")
	store.Root.Children = append(store.Root.Children, one, two)
	a := &app{store: store, config: bookmarks.DefaultConfig()}
	a.rebuildRows()
	a.renderTabsLine(80)
	x := a.tabRanges[1].start

	a.handleMouse(key{name: "mouse", mouseButton: 0, mouseX: x, mouseY: treeBodyStartLine - 1})
	if active := a.activeSpace(); active == nil || active.ID != one.ID {
		t.Fatalf("active space changed on press: %v", active)
	}
	a.handleMouse(key{name: "mouse", mouseButton: 0, mouseX: x, mouseY: treeBodyStartLine - 1, mouseRelease: true})

	if active := a.activeSpace(); active == nil || active.ID != two.ID {
		t.Fatalf("active space=%v, want Personal", active)
	}
}

func TestMouseWheelOnTabsDoesNotSwitchSpace(t *testing.T) {
	store := bookmarks.NewStore()
	one := bookmarks.NewFolder("Work")
	two := bookmarks.NewFolder("Personal")
	for i := 0; i < 30; i++ {
		one.Children = append(one.Children, bookmarks.NewBookmark("Item", "https://example.com", nil))
	}
	store.Root.Children = append(store.Root.Children, one, two)
	a := &app{store: store, config: bookmarks.DefaultConfig()}
	a.rebuildRows()
	a.renderTabsLine(80)

	a.handleMouse(key{name: "mouse", mouseButton: mouseWheelDown, mouseX: a.tabRanges[1].start, mouseY: treeBodyStartLine - 1})

	if active := a.activeSpace(); active == nil || active.ID != one.ID {
		t.Fatalf("active space changed to %v, want Work", active)
	}
	if a.cursor != mouseScrollLines {
		t.Fatalf("cursor=%d, want %d", a.cursor, mouseScrollLines)
	}
	if a.offset != mouseScrollLines {
		t.Fatalf("offset=%d, want %d", a.offset, mouseScrollLines)
	}
}

func TestMoveNodeToFolderByID(t *testing.T) {
	store := bookmarks.NewStore()
	inbox := bookmarks.NewFolder("Inbox")
	archive := bookmarks.NewFolder("Archive")
	bookmark := bookmarks.NewBookmark("Example", "https://example.com", nil)
	inbox.Children = append(inbox.Children, bookmark)
	store.Root.Children = append(store.Root.Children, inbox, archive)

	a := &app{store: store}
	message, moved := a.moveNodeToFolder(bookmark.ID, archive.ID)
	if !moved {
		t.Fatalf("move failed: %s", message)
	}
	if _, parent, _ := bookmarks.Find(store, bookmark.ID); parent == nil || parent.ID != archive.ID {
		t.Fatalf("bookmark parent=%v, want Archive", parent)
	}
	if len(inbox.Children) != 0 {
		t.Fatalf("inbox children=%d, want 0", len(inbox.Children))
	}
}

func TestBuildFolderChoicesExcludesMovingFolderSubtree(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	docs := bookmarks.NewFolder("Docs")
	work.Children = append(work.Children, docs)
	archive := bookmarks.NewFolder("Archive")
	store.Root.Children = append(store.Root.Children, work, archive)

	choices := buildFolderChoices(store, work)
	for _, choice := range choices {
		if choice.node.ID == work.ID || choice.node.ID == docs.ID {
			t.Fatalf("choice includes moving subtree: %+v", choice)
		}
	}
	if len(choices) != 2 {
		t.Fatalf("choices=%d, want root and Archive", len(choices))
	}
}

func TestFilterFolderChoicesMatchesPath(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	docs := bookmarks.NewFolder("Docs")
	work.Children = append(work.Children, docs)
	personal := bookmarks.NewFolder("Personal")
	store.Root.Children = append(store.Root.Children, work, personal)

	choices := buildFolderChoices(store, nil)
	filtered := filterFolderChoices(choices, "work/doc")
	if len(filtered) != 1 || filtered[0].node.ID != docs.ID {
		t.Fatalf("filtered=%v, want Docs", filtered)
	}
	if index := indexFolderChoice(filtered, docs.ID); index != 0 {
		t.Fatalf("index=%d, want 0", index)
	}
}

func TestMoveFolderCannotMoveIntoItself(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	docs := bookmarks.NewFolder("Docs")
	work.Children = append(work.Children, docs)
	store.Root.Children = append(store.Root.Children, work)

	a := &app{store: store}
	if message, moved := a.moveNodeToFolder(work.ID, docs.ID); moved {
		t.Fatalf("move succeeded unexpectedly: %s", message)
	}
	if len(work.Children) != 1 || work.Children[0].ID != docs.ID {
		t.Fatalf("folder children changed: %+v", work.Children)
	}
}

func TestReorderNodeSwapsSiblings(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	alpha := bookmarks.NewBookmark("Alpha", "https://alpha.example", nil)
	beta := bookmarks.NewBookmark("Beta", "https://beta.example", nil)
	gamma := bookmarks.NewBookmark("Gamma", "https://gamma.example", nil)
	work.Children = append(work.Children, alpha, beta, gamma)
	store.Root.Children = append(store.Root.Children, work)

	a := &app{store: store}
	if message, moved := a.reorderNode(beta.ID, -1); !moved {
		t.Fatalf("reorder up failed: %s", message)
	}
	if got := childTitles(work); strings.Join(got, ",") != "Beta,Alpha,Gamma" {
		t.Fatalf("order=%v", got)
	}
	if message, moved := a.reorderNode(beta.ID, 1); !moved {
		t.Fatalf("reorder down failed: %s", message)
	}
	if got := childTitles(work); strings.Join(got, ",") != "Alpha,Beta,Gamma" {
		t.Fatalf("order=%v", got)
	}
	if message, moved := a.reorderNode(alpha.ID, -1); moved {
		t.Fatalf("top item moved unexpectedly: %s", message)
	}
}

func TestReorderSelectedKeepsSpaceActive(t *testing.T) {
	store := bookmarks.NewStore()
	one := bookmarks.NewFolder("One")
	two := bookmarks.NewFolder("Two")
	three := bookmarks.NewFolder("Three")
	store.Root.Children = append(store.Root.Children, one, two, three)
	a := &app{
		dataPath: filepath.Join(t.TempDir(), "bookmarks.json"),
		store:    store,
		config:   bookmarks.DefaultConfig(),
	}
	a.spaceIndex = 1
	a.rebuildRows()

	a.reorderSelected(-1)

	if got := childTitles(store.Root); strings.Join(got, ",") != "Two,One,Three" {
		t.Fatalf("spaces=%v", got)
	}
	if active := a.activeSpace(); active == nil || active.ID != two.ID {
		t.Fatalf("active space=%v, want Two", active)
	}
	if selected := a.selected(); selected == nil || selected.node.ID != two.ID {
		t.Fatalf("selected=%v, want Two", selected)
	}
}

func TestUndoLastRestoresReorder(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	alpha := bookmarks.NewBookmark("Alpha", "https://alpha.example", nil)
	beta := bookmarks.NewBookmark("Beta", "https://beta.example", nil)
	work.Children = append(work.Children, alpha, beta)
	store.Root.Children = append(store.Root.Children, work)
	a := &app{
		dataPath: filepath.Join(t.TempDir(), "bookmarks.json"),
		store:    store,
		config:   bookmarks.DefaultConfig(),
	}
	a.rebuildRows()
	a.cursor = 1

	a.reorderSelected(1)
	if got := childTitles(work); strings.Join(got, ",") != "Beta,Alpha" {
		t.Fatalf("order after reorder=%v", got)
	}

	a.undoLast()
	restored := a.activeSpace()
	if got := childTitles(restored); strings.Join(got, ",") != "Alpha,Beta" {
		t.Fatalf("order after undo=%v", got)
	}
}

func TestUndoLastRestoresDelete(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	bookmark := bookmarks.NewBookmark("Example", "https://example.com", nil)
	work.Children = append(work.Children, bookmark)
	store.Root.Children = append(store.Root.Children, work)
	a := &app{
		dataPath: filepath.Join(t.TempDir(), "bookmarks.json"),
		store:    store,
		config:   bookmarks.DefaultConfig(),
	}
	a.rebuildRows()
	a.captureUndo("delete", bookmark.ID)
	if _, ok := bookmarks.Remove(store, bookmark.ID); !ok {
		t.Fatal("remove failed")
	}
	a.rebuildRows()

	a.undoLast()
	if restored := a.activeSpace(); bookmarks.CountBookmarks(restored) != 1 {
		t.Fatalf("bookmarks after undo=%d, want 1", bookmarks.CountBookmarks(restored))
	}
}

func TestReloadReadsStoreFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.json")
	initial := bookmarks.NewStore()
	initial.Root.Children = append(initial.Root.Children, bookmarks.NewFolder("Initial"))
	if err := bookmarks.Save(path, initial); err != nil {
		t.Fatal(err)
	}

	updated := bookmarks.NewStore()
	updated.Root.Children = append(updated.Root.Children, bookmarks.NewFolder("Updated"))
	if err := bookmarks.Save(path, updated); err != nil {
		t.Fatal(err)
	}

	a := &app{dataPath: path, store: initial, config: bookmarks.DefaultConfig()}
	a.rebuildRows()
	a.captureUndo("delete", "")
	a.reload()

	if a.undo != nil {
		t.Fatal("reload did not clear undo")
	}
	if active := a.activeSpace(); active == nil || active.Title != "Updated" {
		t.Fatalf("active space=%v, want Updated", active)
	}
	if a.status != "reloaded" {
		t.Fatalf("status=%q, want reloaded", a.status)
	}
}

func TestParseSGRMouse(t *testing.T) {
	mouse, ok := parseSGRMouse("[<0;12;5M")
	if !ok {
		t.Fatal("mouse sequence was not parsed")
	}
	if mouse.name != "mouse" || mouse.mouseButton != 0 || mouse.mouseX != 12 || mouse.mouseY != 5 || mouse.mouseRelease {
		t.Fatalf("unexpected mouse event: %+v", mouse)
	}
	release, ok := parseSGRMouse("[<0;12;5m")
	if !ok || !release.mouseRelease {
		t.Fatalf("mouse release was not parsed: %+v ok=%v", release, ok)
	}
	wheel, ok := parseSGRMouse("[<65;12;5M")
	if !ok || wheel.mouseButton != mouseWheelDown {
		t.Fatalf("mouse wheel was not parsed: %+v ok=%v", wheel, ok)
	}
}

func TestSplitEscapePayloadKeepsPendingMouseRelease(t *testing.T) {
	first, rest := splitEscapePayload([]byte("[<0;12;5M\x1b[<0;12;5m"))
	if string(first) != "[<0;12;5M" {
		t.Fatalf("first=%q", first)
	}
	if string(rest) != "\x1b[<0;12;5m" {
		t.Fatalf("rest=%q", rest)
	}
}

func TestReadKeyParsesBufferedEscapePayload(t *testing.T) {
	oldPending := pendingInput
	oldSuppress := suppressMouseTerminator
	defer func() {
		pendingInput = oldPending
		suppressMouseTerminator = oldSuppress
	}()

	pendingInput = []byte("\x1b[<65;12;5M")
	suppressMouseTerminator = false

	k := readKey()
	if k.name != "mouse" || k.mouseButton != mouseWheelDown || k.mouseX != 12 || k.mouseY != 5 {
		t.Fatalf("key=%+v, want wheel mouse event", k)
	}
	if len(pendingInput) != 0 {
		t.Fatalf("pendingInput=%q, want empty", pendingInput)
	}
}

func TestReadKeyParsesMultipleBufferedMouseEvents(t *testing.T) {
	oldPending := pendingInput
	oldSuppress := suppressMouseTerminator
	defer func() {
		pendingInput = oldPending
		suppressMouseTerminator = oldSuppress
	}()

	pendingInput = []byte("\x1b[<65;12;5M\x1b[<64;12;4M")
	suppressMouseTerminator = false

	down := readKey()
	if down.name != "mouse" || down.mouseButton != mouseWheelDown {
		t.Fatalf("first key=%+v, want wheel down", down)
	}
	up := readKey()
	if up.name != "mouse" || up.mouseButton != mouseWheelUp {
		t.Fatalf("second key=%+v, want wheel up", up)
	}
}

func TestIncompleteSGRMousePayloadDetected(t *testing.T) {
	if !isIncompleteSGRMousePayload([]byte("[<0;12;5")) {
		t.Fatal("incomplete mouse payload was not detected")
	}
	if isIncompleteSGRMousePayload([]byte("[<0;12;5m")) {
		t.Fatal("complete mouse release was treated as incomplete")
	}
	if isIncompleteSGRMousePayload([]byte("[6~")) {
		t.Fatal("page key was treated as incomplete mouse")
	}
}

func TestReadByteSuppressesDanglingMouseTerminator(t *testing.T) {
	oldPending := pendingInput
	oldSuppress := suppressMouseTerminator
	defer func() {
		pendingInput = oldPending
		suppressMouseTerminator = oldSuppress
	}()

	pendingInput = []byte{'m', 'x'}
	suppressMouseTerminator = true
	ch, err := readByte()
	if err != nil {
		t.Fatal(err)
	}
	if ch != 'x' {
		t.Fatalf("ch=%q, want x", ch)
	}
}

func TestSplitEscapePayloadKeepsPendingPageKey(t *testing.T) {
	first, rest := splitEscapePayload([]byte("[6~x"))
	if string(first) != "[6~" {
		t.Fatalf("first=%q", first)
	}
	if string(rest) != "x" {
		t.Fatalf("rest=%q", rest)
	}
}

func TestSplitEscapePayloadKeepsPendingShiftTab(t *testing.T) {
	first, rest := splitEscapePayload([]byte("[Zx"))
	if string(first) != "[Z" {
		t.Fatalf("first=%q", first)
	}
	if string(rest) != "x" {
		t.Fatalf("rest=%q", rest)
	}
}

func TestMouseClickSelectsRowAndSecondClickTogglesFolder(t *testing.T) {
	store := bookmarks.NewStore()
	inbox := bookmarks.NewFolder("Inbox")
	child := bookmarks.NewFolder("Child")
	child.Children = append(child.Children, bookmarks.NewBookmark("Example", "https://example.com", nil))
	inbox.Children = append(inbox.Children, child)
	store.Root.Children = append(store.Root.Children, inbox)

	a := &app{store: store}
	a.rebuildRows()
	a.handleMouse(key{name: "mouse", mouseButton: 0, mouseX: 6, mouseY: treeBodyStartLine + 1})
	if a.cursor != 1 {
		t.Fatalf("cursor=%d, want 1", a.cursor)
	}
	if a.status != "selected: Child" {
		t.Fatalf("status=%q", a.status)
	}

	a.handleMouse(key{name: "mouse", mouseButton: 0, mouseX: 6, mouseY: treeBodyStartLine + 1})
	if child.Expanded {
		t.Fatalf("folder was not toggled closed")
	}
}

func TestFolderToggleDoesNotSaveBookmarkFile(t *testing.T) {
	store := bookmarks.NewStore()
	inbox := bookmarks.NewFolder("Inbox")
	store.Root.Children = append(store.Root.Children, inbox)
	dataPath := filepath.Join(t.TempDir(), "bookmarks.json")
	a := &app{dataPath: dataPath, store: store, config: bookmarks.DefaultConfig()}
	a.rebuildRows()

	a.openOrToggle()

	if _, err := os.Stat(dataPath); !os.IsNotExist(err) {
		t.Fatalf("data file was written during folder toggle: %v", err)
	}
	if a.status != "toggled" {
		t.Fatalf("status=%q, want toggled", a.status)
	}
}

func TestMouseWheelScrollsRows(t *testing.T) {
	store := bookmarks.NewStore()
	for i := 0; i < 30; i++ {
		store.Root.Children = append(store.Root.Children, bookmarks.NewBookmark("Item", "https://example.com", nil))
	}
	a := &app{store: store}
	a.rebuildRows()
	a.handleMouse(key{name: "mouse", mouseButton: mouseWheelDown, mouseX: 6, mouseY: treeBodyStartLine + 1})
	if a.offset != mouseScrollLines {
		t.Fatalf("offset=%d, want %d", a.offset, mouseScrollLines)
	}
	if a.cursor != mouseScrollLines {
		t.Fatalf("cursor=%d, want %d", a.cursor, mouseScrollLines)
	}
	a.handleMouse(key{name: "mouse", mouseButton: mouseWheelUp, mouseX: 6, mouseY: treeBodyStartLine + 1})
	if a.offset != 0 {
		t.Fatalf("offset after wheel up=%d, want 0", a.offset)
	}
	if a.cursor != mouseScrollLines {
		t.Fatalf("cursor after wheel up=%d, want %d", a.cursor, mouseScrollLines)
	}
}

func TestMouseWheelScrollsViewportWithoutChangingVisibleSelection(t *testing.T) {
	store := bookmarks.NewStore()
	for i := 0; i < 30; i++ {
		store.Root.Children = append(store.Root.Children, bookmarks.NewBookmark("Item", "https://example.com", nil))
	}
	a := &app{store: store}
	a.rebuildRows()
	a.cursor = 10

	a.handleMouse(key{name: "mouse", mouseButton: mouseWheelDown, mouseX: 6, mouseY: treeBodyStartLine + 1})

	if a.offset != mouseScrollLines {
		t.Fatalf("offset=%d, want %d", a.offset, mouseScrollLines)
	}
	if a.cursor != 10 {
		t.Fatalf("cursor=%d, want unchanged 10", a.cursor)
	}
}

func TestScrollRowsKeepsVisibleCursor(t *testing.T) {
	store := bookmarks.NewStore()
	for i := 0; i < 12; i++ {
		store.Root.Children = append(store.Root.Children, bookmarks.NewBookmark("Item", "https://example.com", nil))
	}
	a := &app{store: store}
	a.rebuildRows()

	a.scrollRows(4, 5)
	if a.offset != 4 || a.cursor != 4 {
		t.Fatalf("offset=%d cursor=%d, want 4/4", a.offset, a.cursor)
	}
	a.scrollRows(100, 5)
	if a.offset != len(a.treeRows)-5 || a.cursor != len(a.treeRows)-5 {
		t.Fatalf("offset=%d cursor=%d, want %d", a.offset, a.cursor, len(a.treeRows)-5)
	}
	a.scrollRows(-2, 5)
	if a.offset != len(a.treeRows)-7 {
		t.Fatalf("offset=%d, want %d", a.offset, len(a.treeRows)-7)
	}
	if a.cursor < a.offset || a.cursor >= a.offset+5 {
		t.Fatalf("cursor=%d outside offset=%d height=5", a.cursor, a.offset)
	}
}

func TestScrollRowsStopsAtEdges(t *testing.T) {
	store := bookmarks.NewStore()
	for i := 0; i < 12; i++ {
		store.Root.Children = append(store.Root.Children, bookmarks.NewBookmark("Item", "https://example.com", nil))
	}
	a := &app{store: store}
	a.rebuildRows()

	a.scrollRows(-100, 5)
	if a.offset != 0 || a.cursor != 0 {
		t.Fatalf("offset=%d cursor=%d, want top 0/0", a.offset, a.cursor)
	}

	a.scrollRows(100, 5)
	if want := len(a.treeRows) - 5; a.offset != want || a.cursor != want {
		t.Fatalf("offset=%d cursor=%d, want bottom %d/%d", a.offset, a.cursor, want, want)
	}

	a.scrollRows(100, 5)
	if want := len(a.treeRows) - 5; a.offset != want || a.cursor != want {
		t.Fatalf("offset=%d cursor=%d, want still bottom %d/%d", a.offset, a.cursor, want, want)
	}
}

func TestBookmarkIconUsesNerdFontStyleDomainIcon(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://github.com/yamajun9929/branchmark", iconGitHub},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", iconYouTube},
		{"https://youtu.be/dQw4w9WgXcQ", iconYouTube},
		{"https://drive.google.com/drive/u/0/folders/example", iconGoogleDrive},
		{"https://docs.google.com/spreadsheets/d/example/edit", iconGoogleSpreadsheet},
		{"https://docs.google.com/document/d/example/edit", iconDocument},
		{"https://docs.google.com/presentation/d/example/edit", iconPresentation},
		{"https://docs.google.com/forms/d/example/edit", iconForms},
		{"https://mail.google.com/mail/u/0/#inbox", iconGmail},
		{"https://calendar.google.com/calendar/u/0/r", iconCalendar},
		{"https://meet.google.com/abc-defg-hij", iconVideo},
		{"https://www.google.com/maps/place/Tokyo", iconGoogleMaps},
		{"https://cloud.google.com/run", iconGoogleCloud},
		{"https://console.cloud.google.com/home/dashboard", iconGoogleCloud},
		{"https://analytics.google.com/analytics/web/", iconGoogleAnalytics},
		{"https://ads.google.com/aw/overview", iconGoogleAds},
		{"https://classroom.google.com/", iconGoogleClassroom},
		{"https://keep.google.com/", iconGoogleKeep},
		{"https://play.google.com/store/apps", iconGooglePlay},
		{"https://translate.google.com/", iconGoogleTranslate},
		{"https://earth.google.com/web/", iconGoogleEarth},
		{"https://console.firebase.google.com/project/example/overview", iconFirebase},
		{"https://chromewebstore.google.com/detail/example", iconGoogleChrome},
		{"https://www.google.com/search?q=brmk", iconGoogle},
		{"https://aws.amazon.com/console/", iconAws},
		{"https://console.aws.amazon.com/console/home", iconAws},
		{"https://s3.amazonaws.com/example", iconAws},
		{"https://my.salesforce.com/", iconSalesforce},
		{"https://example.lightning.force.com/lightning/page/home", iconSalesforce},
		{"https://www.microsoft.com/", iconMicrosoft},
		{"https://www.office.com/", iconMicrosoftOffice},
		{"https://teams.microsoft.com/l/channel/example", iconMicrosoftTeams},
		{"https://outlook.office.com/mail/", iconMicrosoftOutlook},
		{"https://onedrive.live.com/", iconMicrosoftOneDrive},
		{"https://contoso.sharepoint.com/sites/team", iconMicrosoftSharePoint},
		{"https://excel.office.com/spreadsheets/example", iconMicrosoftExcel},
		{"https://word.office.com/document/example", iconMicrosoftWord},
		{"https://powerpoint.office.com/presentation/example", iconMicrosoftPowerPoint},
		{"https://www.onenote.com/notebooks", iconMicrosoftOneNote},
		{"https://portal.azure.com/#home", iconMicrosoftAzure},
		{"https://www.bing.com/search?q=brmk", iconMicrosoftBing},
		{"https://app.powerbi.com/groups/me/list", iconStats},
		{"https://www.yahoo.co.jp/", iconYahoo},
		{"https://go.dev/doc/", iconGo},
		{"https://example.com", defaultBookmarkIcon},
		{"https://example.com/README.md", iconMarkdown},
	}
	for _, tt := range cases {
		got := bookmarkIcon(bookmarks.NewBookmark("test", tt.url, nil))
		if got != tt.want {
			t.Fatalf("url=%q got=%q want=%q", tt.url, got, tt.want)
		}
	}
}

func formattedRows(treeRows []treeRow) []string {
	lines := make([]string, 0, len(treeRows))
	for _, item := range treeRows {
		lines = append(lines, formatRow(item))
	}
	return lines
}

func lineAt(lines []string, i int) string {
	if i < 0 || i >= len(lines) {
		return ""
	}
	return lines[i]
}

func childTitles(parent *bookmarks.Node) []string {
	titles := make([]string, 0, len(parent.Children))
	for _, child := range parent.Children {
		titles = append(titles, child.Title)
	}
	return titles
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
