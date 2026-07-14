package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
)

func TestRunAddUsesSpaceFlagWithoutPrompt(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "bookmarks.json")
	configPath := filepath.Join(dir, "config.json")

	err := runAdd([]string{
		"https://example.com",
		"--data", dataPath,
		"--config", configPath,
		"--space", "Work",
		"--title", "Example",
		"--tags", "docs,web",
	})
	if err != nil {
		t.Fatal(err)
	}

	store, err := bookmarks.Load(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	work := findChildFolder(store.Root, "Work")
	if work == nil {
		t.Fatal("Work space was not created")
	}
	if len(work.Children) != 1 {
		t.Fatalf("bookmarks=%d, want 1", len(work.Children))
	}
	bookmark := work.Children[0]
	if bookmark.Title != "Example" || bookmark.URL != "https://example.com" {
		t.Fatalf("bookmark=%+v", bookmark)
	}
	if strings.Join(bookmark.Tags, ",") != "docs,web" {
		t.Fatalf("tags=%v", bookmark.Tags)
	}
}

func TestRunAddUsesConfiguredDefaultSpaceWithNoPrompt(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "bookmarks.json")
	configPath := filepath.Join(dir, "config.json")
	if err := bookmarks.SaveConfig(configPath, &bookmarks.Config{
		ActiveProfile: "default",
		WebBrowser:    "Google Chrome",
		DefaultSpace:  "Read Later",
		BrowserProfiles: []bookmarks.BrowserProfile{
			{Name: "default", Kind: "default"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	err := runAdd([]string{
		"https://example.com/article",
		"--data", dataPath,
		"--config", configPath,
		"--title", "Article",
		"--tags", "readlater",
		"--no-prompt",
	})
	if err != nil {
		t.Fatal(err)
	}

	store, err := bookmarks.Load(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	space := findChildFolder(store.Root, "Read Later")
	if space == nil {
		t.Fatal("configured default space was not used")
	}
	if len(space.Children) != 1 || space.Children[0].Title != "Article" {
		t.Fatalf("space children=%+v", space.Children)
	}
}

func TestRunAddAcceptsYesAlias(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "bookmarks.json")
	configPath := filepath.Join(dir, "config.json")

	err := runAdd([]string{
		"https://example.com/alias",
		"--data", dataPath,
		"--config", configPath,
		"--space", "Inbox",
		"--title", "Alias",
		"--yes",
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := bookmarks.Load(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	inbox := findChildFolder(store.Root, "Inbox")
	if inbox == nil || len(inbox.Children) != 1 {
		t.Fatalf("inbox=%+v", inbox)
	}
}

func TestRunAddCreatesDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "bookmarks.json")
	configPath := filepath.Join(dir, "config.json")

	err := runAdd([]string{
		"https://example.com/new",
		"--data", dataPath,
		"--config", configPath,
		"--title", "New",
		"--no-prompt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
	cfg, err := bookmarks.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultSpace != bookmarks.DefaultSpace {
		t.Fatalf("default_space=%q", cfg.DefaultSpace)
	}
}

func TestRunAddDryRunDoesNotWriteData(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "bookmarks.json")
	configPath := filepath.Join(dir, "config.json")

	err := runAdd([]string{
		"https://example.com/dry",
		"--data", dataPath,
		"--config", configPath,
		"--space", "Work",
		"--title", "Dry",
		"--tags", "docs",
		"--no-prompt",
		"--dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dataPath); !os.IsNotExist(err) {
		t.Fatalf("data file exists after dry-run: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config file exists after dry-run: %v", err)
	}
}

func TestRunAddSpaceQuestionPromptsForSpace(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "bookmarks.json")
	configPath := filepath.Join(dir, "config.json")
	store := bookmarks.NewStore()
	store.Root.Children = append(store.Root.Children, bookmarks.NewFolder("Work"), bookmarks.NewFolder("Personal"))
	if err := bookmarks.Save(dataPath, store); err != nil {
		t.Fatal(err)
	}

	withStdin(t, "Work\n\n", func() {
		err := runAdd([]string{
			"https://example.com/choice",
			"--data", dataPath,
			"--config", configPath,
			"--space", "?",
			"--title", "Choice",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	store, err := bookmarks.Load(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	work := findChildFolder(store.Root, "Work")
	if work == nil || len(work.Children) != 1 || work.Children[0].Title != "Choice" {
		t.Fatalf("work=%+v", work)
	}
}

func TestReadPromptWithDefault(t *testing.T) {
	var output bytes.Buffer
	got, err := readPromptWithDefault(bufio.NewReader(strings.NewReader("\n")), &output, "Space", "Inbox")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Inbox" {
		t.Fatalf("value=%q, want Inbox", got)
	}
	if !strings.Contains(output.String(), "Space [Inbox]:") {
		t.Fatalf("prompt output=%q", output.String())
	}
}

func TestRunCompletionZshPrintsCompletionScript(t *testing.T) {
	output := captureRunStdout(t, func() {
		if err := runCompletion([]string{"zsh"}); err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{"#compdef brmk", "profile:manage browser profiles", "theme:manage color themes", "__complete profiles", "__complete spaces", "__complete themes"} {
		if !strings.Contains(output, want) {
			t.Fatalf("completion output missing %q:\n%s", want, output)
		}
	}
}

func TestRunThemeSetUpdatesConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := runThemeSet([]string{"gruvbox-light", "--config", configPath}); err != nil {
		t.Fatal(err)
	}
	cfg, err := bookmarks.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "gruvbox-light" {
		t.Fatalf("theme=%q", cfg.Theme)
	}
}

func TestRunThemeSetRejectsUnknownTheme(t *testing.T) {
	err := runThemeSet([]string{"not-a-theme", "--config", filepath.Join(t.TempDir(), "config.json")})
	if err == nil || !strings.Contains(err.Error(), "unknown theme") {
		t.Fatalf("error=%v", err)
	}
}

func TestInternalCompleteProfilesReadsConfigWithoutCreatingIt(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := bookmarks.SaveConfig(configPath, &bookmarks.Config{
		ActiveProfile: "work",
		BrowserProfiles: []bookmarks.BrowserProfile{
			{Name: "work", Browser: "Google Chrome", Kind: "managed"},
			{Name: "private", Browser: "Brave Browser", Kind: "managed"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	output := captureRunStdout(t, func() {
		if err := runInternalComplete([]string{"profiles", "--config", configPath}); err != nil {
			t.Fatal(err)
		}
	})
	if got, want := output, "default\nprivate\nwork\n"; got != want {
		t.Fatalf("profiles=%q, want %q", got, want)
	}
}

func TestInternalCompleteSpacesReadsData(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "bookmarks.json")
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	work.Children = append(work.Children, bookmarks.NewFolder("Engineering"))
	store.Root.Children = append(store.Root.Children, work, bookmarks.NewFolder("Personal"))
	if err := bookmarks.Save(dataPath, store); err != nil {
		t.Fatal(err)
	}

	output := captureRunStdout(t, func() {
		if err := runInternalComplete([]string{"spaces", "--data", dataPath}); err != nil {
			t.Fatal(err)
		}
	})
	if got, want := output, "Personal\nWork\nWork/Engineering\n"; got != want {
		t.Fatalf("spaces=%q, want %q", got, want)
	}
}

func TestFolderPathsIncludesNestedFolders(t *testing.T) {
	store := bookmarks.NewStore()
	work := bookmarks.NewFolder("Work")
	engineering := bookmarks.NewFolder("Engineering")
	engineering.Children = append(engineering.Children, bookmarks.NewFolder("Infra"))
	work.Children = append(work.Children, engineering)
	store.Root.Children = append(store.Root.Children, work, bookmarks.NewFolder("Personal"))

	got := strings.Join(folderPaths(store), "\n")
	want := "Personal\nWork\nWork/Engineering\nWork/Engineering/Infra"
	if got != want {
		t.Fatalf("folder paths=%q, want %q", got, want)
	}
}

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	old := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString(input); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdin = reader
	defer func() {
		os.Stdin = old
		_ = reader.Close()
	}()
	fn()
}

func captureRunStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = old }()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func findChildFolder(parent *bookmarks.Node, title string) *bookmarks.Node {
	for _, child := range parent.Children {
		if child.IsFolder() && strings.EqualFold(child.Title, title) {
			return child
		}
	}
	return nil
}
