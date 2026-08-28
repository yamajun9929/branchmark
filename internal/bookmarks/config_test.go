package bookmarks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg, err := EnsureConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProfile != "default" {
		t.Fatalf("active_profile=%q", cfg.ActiveProfile)
	}
	if cfg.WebBrowser != DefaultWebBrowser {
		t.Fatalf("web_browser=%q", cfg.WebBrowser)
	}
	if cfg.DefaultSpace != DefaultSpace {
		t.Fatalf("default_space=%q", cfg.DefaultSpace)
	}
	if cfg.Theme != DefaultTheme {
		t.Fatalf("theme=%q, want %q", cfg.Theme, DefaultTheme)
	}
	if _, ok := FindBrowserProfile(cfg.BrowserProfiles, "default"); !ok {
		t.Fatal("default profile was not created")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeConfigNormalizesCustomThemeNames(t *testing.T) {
	cfg := NormalizeConfig(&Config{
		Theme: "  My Theme  ",
		Themes: map[string]Theme{
			" My Theme ": {Background: " #101418 "},
		},
	})
	if cfg.Theme != "my theme" {
		t.Fatalf("theme=%q", cfg.Theme)
	}
	if got := cfg.Themes["my theme"].Background; got != "#101418" {
		t.Fatalf("background=%q", got)
	}
}

func TestConfigPreservesTransparentBackground(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := DefaultConfig()
	cfg.TransparentBackground = true

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.TransparentBackground {
		t.Fatal("transparent_background was not preserved")
	}
}

func TestDefaultConfigPathUsesBRMKConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.json")
	t.Setenv("BRMK_CONFIG", path)

	if got := DefaultConfigPath(); got != path {
		t.Fatalf("path=%q, want %q", got, path)
	}
}

func TestDefaultConfigPathUsesXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRMK_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := filepath.Join(dir, "brmk", "config.json")
	if got := DefaultConfigPath(); got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}

func TestDefaultConfigPathUsesHomeDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("BRMK_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".config", "brmk", "config.json")
	if got := DefaultConfigPath(); got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}

func TestBrowserProfileConfig(t *testing.T) {
	cfg := UpsertBrowserProfile(DefaultConfig(), BrowserProfile{
		Name:    "work",
		Browser: "chrome",
		Kind:    "managed",
	})
	var ok bool
	cfg, ok = SelectBrowserProfile(cfg, "work")
	if !ok {
		t.Fatal("profile was not selected")
	}
	if cfg.ActiveProfile != "work" {
		t.Fatalf("active_profile=%q", cfg.ActiveProfile)
	}
	profile, ok := FindBrowserProfile(cfg.BrowserProfiles, "work")
	if !ok {
		t.Fatal("profile was not found")
	}
	if profile.Kind != "managed" || profile.Browser != "chrome" {
		t.Fatalf("profile=%+v", profile)
	}
}
