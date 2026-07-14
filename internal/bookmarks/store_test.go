package bookmarks

import (
	"path/filepath"
	"testing"
)

func TestDefaultStorePathUsesBRMKData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.json")
	t.Setenv("BRMK_DATA", path)

	if got := DefaultStorePath(); got != path {
		t.Fatalf("path=%q, want %q", got, path)
	}
}

func TestDefaultStorePathUsesXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRMK_DATA", "")
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := filepath.Join(dir, "brmk", "bookmarks.json")
	if got := DefaultStorePath(); got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}

func TestDefaultStorePathUsesHomeDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("BRMK_DATA", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".config", "brmk", "bookmarks.json")
	if got := DefaultStorePath(); got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}

func TestSaveCreatesBackupOfPreviousStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.json")

	store := NewStore()
	store.Root.Children = append(store.Root.Children, NewBookmark("One", "https://one.example", nil))
	if err := Save(path, store); err != nil {
		t.Fatal(err)
	}

	store.Root.Children = append(store.Root.Children, NewBookmark("Two", "https://two.example", nil))
	if err := Save(path, store); err != nil {
		t.Fatal(err)
	}

	backups, err := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "bookmarks-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups=%d, want 1: %v", len(backups), backups)
	}

	backupStore, err := Load(backups[0])
	if err != nil {
		t.Fatal(err)
	}
	if got := CountBookmarks(backupStore.Root); got != 1 {
		t.Fatalf("backup bookmarks=%d, want 1", got)
	}

	currentStore, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := CountBookmarks(currentStore.Root); got != 2 {
		t.Fatalf("current bookmarks=%d, want 2", got)
	}
}
