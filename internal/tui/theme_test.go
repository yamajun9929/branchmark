package tui

import (
	"testing"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
)

func TestThemeNamesIncludeBuiltinsAndCustomThemes(t *testing.T) {
	cfg := bookmarks.DefaultConfig()
	cfg.Themes = map[string]bookmarks.Theme{"my-theme": {Accent: "#ff00aa"}}
	got := ThemeNames(cfg)
	for _, want := range []string{"catppuccin-mocha", "dracula", "gruvbox-dark", "gruvbox-light", "monochrome", "my-theme", "nord", "terminal", "tokyonight"} {
		found := false
		for _, name := range got {
			if name == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("themes=%v, missing %q", got, want)
		}
	}
}

func TestResolveThemeUsesCustomOverrides(t *testing.T) {
	cfg := bookmarks.DefaultConfig()
	cfg.Theme = "custom"
	cfg.Themes = map[string]bookmarks.Theme{
		"custom": {Background: "#101418", Accent: "#ff00aa"},
	}
	got := resolveTheme(cfg)
	if got.name != "custom" {
		t.Fatalf("name=%q", got.name)
	}
	if got.background != "\x1b[48;2;16;20;24m" {
		t.Fatalf("background=%q", got.background)
	}
	if got.accent != "\x1b[38;2;255;0;170m" {
		t.Fatalf("accent=%q", got.accent)
	}
	if got.foreground == "" {
		t.Fatal("foreground should inherit from the default theme")
	}
}

func TestResolveTerminalThemeUsesTerminalColors(t *testing.T) {
	got := resolveTheme(&bookmarks.Config{Theme: "terminal"})
	if got.background != "" || got.foreground != "" {
		t.Fatalf("terminal theme sets colors: %+v", got)
	}
	if got.selection != "\x1b[7m" || got.border != "\x1b[1m" {
		t.Fatalf("terminal theme=%+v", got)
	}
}

func TestCanonicalThemeName(t *testing.T) {
	cfg := bookmarks.DefaultConfig()
	cfg.Themes = map[string]bookmarks.Theme{"custom": {}}
	if got, ok := CanonicalThemeName(cfg, "  DRACULA "); !ok || got != "dracula" {
		t.Fatalf("theme=%q, ok=%v", got, ok)
	}
	if _, ok := CanonicalThemeName(cfg, "missing"); ok {
		t.Fatal("missing theme was accepted")
	}
}
