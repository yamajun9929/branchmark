package bookmarks

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ActiveProfile   string           `json:"active_profile"`
	WebBrowser      string           `json:"web_browser,omitempty"`
	DefaultSpace    string           `json:"default_space,omitempty"`
	Theme           string           `json:"theme,omitempty"`
	Themes          map[string]Theme `json:"themes,omitempty"`
	BrowserProfiles []BrowserProfile `json:"browser_profiles"`
}

// Theme defines optional color overrides for a custom TUI theme. Colors use
// the #RRGGBB format. Omitted colors inherit from catppuccin-mocha.
type Theme struct {
	Background          string `json:"background,omitempty"`
	SelectionBackground string `json:"selection_background,omitempty"`
	Foreground          string `json:"foreground,omitempty"`
	Muted               string `json:"muted,omitempty"`
	Accent              string `json:"accent,omitempty"`
	Highlight           string `json:"highlight,omitempty"`
	Success             string `json:"success,omitempty"`
	Border              string `json:"border,omitempty"`
}

type BrowserProfile struct {
	Name      string   `json:"name"`
	Browser   string   `json:"browser,omitempty"`
	Kind      string   `json:"kind,omitempty"`
	Path      string   `json:"path,omitempty"`
	Directory string   `json:"directory,omitempty"`
	Args      []string `json:"args,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		ActiveProfile: "default",
		WebBrowser:    DefaultWebBrowser,
		DefaultSpace:  DefaultSpace,
		Theme:         DefaultTheme,
		BrowserProfiles: []BrowserProfile{
			{Name: "default", Kind: "default"},
		},
	}
}

func DefaultConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("BRMK_CONFIG")); path != "" {
		return path
	}
	if dir := defaultConfigDir(); dir != "" {
		return filepath.Join(dir, "brmk", "config.json")
	}
	return "config.json"
}

func EnsureConfig(path string) (*Config, error) {
	cfg, err := LoadConfig(path)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg = DefaultConfig()
	if err := SaveConfig(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultConfigDir() string {
	if dir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); dir != "" {
		return dir
	}
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		return filepath.Join(home, ".config")
	}
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return dir
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return DefaultConfig(), nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return NormalizeConfig(&cfg), nil
}

func SaveConfig(path string, cfg *Config) error {
	if path == "" {
		path = DefaultConfigPath()
	}
	cfg = NormalizeConfig(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func NormalizeConfig(cfg *Config) *Config {
	if cfg == nil {
		return DefaultConfig()
	}
	cfg.ActiveProfile = strings.TrimSpace(cfg.ActiveProfile)
	cfg.WebBrowser = strings.TrimSpace(cfg.WebBrowser)
	cfg.DefaultSpace = strings.TrimSpace(cfg.DefaultSpace)
	cfg.Theme = strings.ToLower(strings.TrimSpace(cfg.Theme))
	if cfg.Theme == "" {
		cfg.Theme = DefaultTheme
	}
	cfg.Themes = normalizeThemes(cfg.Themes)
	cfg.BrowserProfiles = normalizeBrowserProfiles(cfg.BrowserProfiles)
	if cfg.ActiveProfile == "" || !hasBrowserProfile(cfg.BrowserProfiles, cfg.ActiveProfile) {
		cfg.ActiveProfile = "default"
	}
	return cfg
}

const (
	DefaultWebBrowser = "Google Chrome"
	DefaultSpace      = "Inbox"
	DefaultTheme      = "catppuccin-mocha"
)

func normalizeThemes(themes map[string]Theme) map[string]Theme {
	if len(themes) == 0 {
		return nil
	}
	normalized := make(map[string]Theme, len(themes))
	for name, theme := range themes {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		theme.Background = strings.TrimSpace(theme.Background)
		theme.SelectionBackground = strings.TrimSpace(theme.SelectionBackground)
		theme.Foreground = strings.TrimSpace(theme.Foreground)
		theme.Muted = strings.TrimSpace(theme.Muted)
		theme.Accent = strings.TrimSpace(theme.Accent)
		theme.Highlight = strings.TrimSpace(theme.Highlight)
		theme.Success = strings.TrimSpace(theme.Success)
		theme.Border = strings.TrimSpace(theme.Border)
		normalized[name] = theme
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func AddBrowser(cfg *Config, override string) string {
	override = strings.TrimSpace(override)
	if override != "" {
		return override
	}
	if cfg != nil && strings.TrimSpace(cfg.WebBrowser) != "" {
		return strings.TrimSpace(cfg.WebBrowser)
	}
	return DefaultWebBrowser
}

func AddSpace(cfg *Config) string {
	if cfg != nil && strings.TrimSpace(cfg.DefaultSpace) != "" {
		return strings.TrimSpace(cfg.DefaultSpace)
	}
	return DefaultSpace
}

func normalizeBrowserProfiles(profiles []BrowserProfile) []BrowserProfile {
	normalized := make([]BrowserProfile, 0, len(profiles)+1)
	seen := map[string]bool{}
	hasDefault := false
	for _, profile := range profiles {
		profile, ok := normalizeBrowserProfile(profile)
		if !ok || seen[strings.ToLower(profile.Name)] {
			continue
		}
		if strings.EqualFold(profile.Name, "default") {
			hasDefault = true
		}
		seen[strings.ToLower(profile.Name)] = true
		normalized = append(normalized, profile)
	}
	if !hasDefault {
		normalized = append([]BrowserProfile{{Name: "default", Kind: "default"}}, normalized...)
	}
	return normalized
}

func normalizeBrowserProfile(profile BrowserProfile) (BrowserProfile, bool) {
	profile.Name = strings.TrimSpace(profile.Name)
	if profile.Name == "" {
		return BrowserProfile{}, false
	}
	profile.Browser = strings.TrimSpace(profile.Browser)
	profile.Kind = strings.ToLower(strings.TrimSpace(profile.Kind))
	switch profile.Kind {
	case "", "default":
		profile.Kind = "default"
	case "managed", "existing":
	default:
		profile.Kind = "managed"
	}
	profile.Path = strings.TrimSpace(profile.Path)
	profile.Directory = strings.TrimSpace(profile.Directory)
	if strings.EqualFold(profile.Name, "default") {
		profile.Name = "default"
		profile.Kind = "default"
	}
	return profile, true
}

func hasBrowserProfile(profiles []BrowserProfile, name string) bool {
	_, ok := FindBrowserProfile(profiles, name)
	return ok
}

func FindBrowserProfile(profiles []BrowserProfile, name string) (BrowserProfile, bool) {
	name = strings.TrimSpace(name)
	for _, profile := range profiles {
		if strings.EqualFold(profile.Name, name) {
			return profile, true
		}
	}
	return BrowserProfile{}, false
}

func UpsertBrowserProfile(cfg *Config, profile BrowserProfile) *Config {
	cfg = NormalizeConfig(cfg)
	var ok bool
	profile, ok = normalizeBrowserProfile(profile)
	if !ok {
		return cfg
	}
	for i, existing := range cfg.BrowserProfiles {
		if strings.EqualFold(existing.Name, profile.Name) {
			cfg.BrowserProfiles[i] = profile
			return NormalizeConfig(cfg)
		}
	}
	cfg.BrowserProfiles = append(cfg.BrowserProfiles, profile)
	return NormalizeConfig(cfg)
}

func SelectBrowserProfile(cfg *Config, name string) (*Config, bool) {
	cfg = NormalizeConfig(cfg)
	if !hasBrowserProfile(cfg.BrowserProfiles, name) {
		return cfg, false
	}
	cfg.ActiveProfile = strings.TrimSpace(name)
	return NormalizeConfig(cfg), true
}
