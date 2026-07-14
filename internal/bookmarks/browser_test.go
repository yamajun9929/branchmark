package bookmarks

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultBrowserProfilePathSanitizesName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := DefaultBrowserProfilePath("Work Profile!")
	if got, want := path[len(path)-len("work-profile"):], "work-profile"; got != want {
		t.Fatalf("suffix=%q, want %q; path=%q", got, want, path)
	}
}

func TestManagedChromiumProfileArgsUseDedicatedDirAndNewWindow(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "chrome-work")

	args, profileDir, err := profileBrowserArgs("https://example.com", BrowserProfile{
		Name:    "Work Work",
		Browser: "Google Chrome",
		Kind:    "managed",
		Path:    profilePath,
	}, "Google Chrome")
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"--user-data-dir=" + profilePath, "--new-window", "https://example.com"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args=%v, want %v", args, wantArgs)
	}
	if profileDir != profilePath {
		t.Fatalf("profileDir=%q, want %q", profileDir, profilePath)
	}
}

func TestExistingChromiumProfileArgsUseProfileDirectoryAndNewWindow(t *testing.T) {
	args, profileDir, err := profileBrowserArgs("https://example.com", BrowserProfile{
		Name:      "work",
		Browser:   "Google Chrome",
		Kind:      "existing",
		Directory: "Profile 3",
	}, "Google Chrome")
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"--profile-directory=Profile 3", "--new-window", "https://example.com"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args=%v, want %v", args, wantArgs)
	}
	if profileDir != "" {
		t.Fatalf("profileDir=%q, want empty", profileDir)
	}
}

func TestManagedFirefoxProfileArgsUseProfilePathAndNewWindow(t *testing.T) {
	args, profileDir, err := profileBrowserArgs("https://example.com", BrowserProfile{
		Name:    "work",
		Browser: "Firefox",
		Kind:    "managed",
		Path:    "/tmp/brmk-firefox-work",
	}, "Firefox")
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"--new-instance", "--profile", "/tmp/brmk-firefox-work", "--new-window", "https://example.com"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args=%v, want %v", args, wantArgs)
	}
	if profileDir != "/tmp/brmk-firefox-work" {
		t.Fatalf("profileDir=%q, want /tmp/brmk-firefox-work", profileDir)
	}
}
