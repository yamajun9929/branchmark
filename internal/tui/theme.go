package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
)

type palette struct {
	Background          string
	SelectionBackground string
	Foreground          string
	Muted               string
	Accent              string
	Highlight           string
	Success             string
	Border              string
}

type theme struct {
	name       string
	background string
	selection  string
	foreground string
	muted      string
	accent     string
	highlight  string
	success    string
	border     string
}

var builtInThemes = map[string]palette{
	"catppuccin-mocha": {
		Background: "#1d202f", SelectionBackground: "#363b53", Foreground: "#cdd6f4", Muted: "#7f889f",
		Accent: "#89b4fa", Highlight: "#74c7ec", Success: "#a6e3a1", Border: "#89b4fa",
	},
	"tokyonight": {
		Background: "#1a1b26", SelectionBackground: "#283457", Foreground: "#c0caf5", Muted: "#565f89",
		Accent: "#7aa2f7", Highlight: "#7dcfff", Success: "#9ece6a", Border: "#7aa2f7",
	},
	"dracula": {
		Background: "#282a36", SelectionBackground: "#44475a", Foreground: "#f8f8f2", Muted: "#6272a4",
		Accent: "#bd93f9", Highlight: "#8be9fd", Success: "#50fa7b", Border: "#bd93f9",
	},
	"nord": {
		Background: "#2e3440", SelectionBackground: "#3b4252", Foreground: "#eceff4", Muted: "#81a1c1",
		Accent: "#88c0d0", Highlight: "#8fbcbb", Success: "#a3be8c", Border: "#88c0d0",
	},
	"gruvbox-dark": {
		Background: "#282828", SelectionBackground: "#3c3836", Foreground: "#ebdbb2", Muted: "#a89984",
		Accent: "#83a598", Highlight: "#8ec07c", Success: "#b8bb26", Border: "#83a598",
	},
	"gruvbox-light": {
		Background: "#fbf1c7", SelectionBackground: "#ebdbb2", Foreground: "#3c3836", Muted: "#7c6f64",
		Accent: "#076678", Highlight: "#427b58", Success: "#79740e", Border: "#076678",
	},
	"monochrome": {
		Background: "#1e1e1e", SelectionBackground: "#444444", Foreground: "#e0e0e0", Muted: "#a0a0a0",
		Accent: "#e0e0e0", Highlight: "#d0d0d0", Success: "#f0f0f0", Border: "#e0e0e0",
	},
}

func ThemeNames(cfg *bookmarks.Config) []string {
	names := make([]string, 0, len(builtInThemes)+1)
	for name := range builtInThemes {
		names = append(names, name)
	}
	names = append(names, "terminal")
	if cfg != nil {
		for name := range cfg.Themes {
			name = strings.ToLower(strings.TrimSpace(name))
			if name != "" {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return uniqueStrings(names)
}

func CanonicalThemeName(cfg *bookmarks.Config, name string) (string, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range ThemeNames(cfg) {
		if candidate == name {
			return candidate, true
		}
	}
	return "", false
}

func resolveTheme(cfg *bookmarks.Config) theme {
	name := bookmarks.DefaultTheme
	if cfg != nil && strings.TrimSpace(cfg.Theme) != "" {
		name = strings.ToLower(strings.TrimSpace(cfg.Theme))
	}
	if name == "terminal" {
		return theme{name: name, selection: "\x1b[7m", accent: "\x1b[1m", highlight: "\x1b[1m", success: "\x1b[1m", border: "\x1b[1m"}
	}
	p, ok := builtInThemes[name]
	if !ok && cfg != nil {
		if custom, found := cfg.Themes[name]; found {
			p = mergePalette(builtInThemes[bookmarks.DefaultTheme], custom)
			ok = true
		}
	}
	if !ok {
		name = bookmarks.DefaultTheme
		p = builtInThemes[name]
	}
	selected := theme{
		name: name, background: backgroundColor(p.Background), selection: backgroundColor(p.SelectionBackground),
		foreground: foregroundColor(p.Foreground), muted: foregroundColor(p.Muted), accent: foregroundColor(p.Accent),
		highlight: foregroundColor(p.Highlight), success: foregroundColor(p.Success), border: foregroundColor(p.Border),
	}
	if cfg != nil && cfg.TransparentBackground {
		selected.background = ""
	}
	return selected
}

func mergePalette(base palette, custom bookmarks.Theme) palette {
	if custom.Background != "" {
		base.Background = custom.Background
	}
	if custom.SelectionBackground != "" {
		base.SelectionBackground = custom.SelectionBackground
	}
	if custom.Foreground != "" {
		base.Foreground = custom.Foreground
	}
	if custom.Muted != "" {
		base.Muted = custom.Muted
	}
	if custom.Accent != "" {
		base.Accent = custom.Accent
	}
	if custom.Highlight != "" {
		base.Highlight = custom.Highlight
	}
	if custom.Success != "" {
		base.Success = custom.Success
	}
	if custom.Border != "" {
		base.Border = custom.Border
	}
	return base
}

func backgroundColor(value string) string { return colorSequence(value, 48) }
func foregroundColor(value string) string { return colorSequence(value, 38) }

func colorSequence(value string, code int) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return ""
	}
	rgb, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", code, (rgb>>16)&0xff, (rgb>>8)&0xff, rgb&0xff)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	unique := values[:0]
	for _, value := range values {
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	return unique
}

func applyTheme(cfg *bookmarks.Config) {
	selected := resolveTheme(cfg)
	ansiBg = selected.background
	ansiSelectBg = selected.selection
	ansiFg = selected.foreground
	ansiMuted = selected.muted
	ansiBlue = selected.accent
	ansiCyan = selected.highlight
	ansiGreen = selected.success
	ansiBorder = selected.border
}
