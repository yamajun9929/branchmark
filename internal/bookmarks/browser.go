package bookmarks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func OpenURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL is empty")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("opening URLs is unsupported on %s", runtime.GOOS)
	}
	return cmd.Start()
}

func OpenURLWithProfile(rawURL string, profile BrowserProfile) error {
	profile, ok := normalizeBrowserProfile(profile)
	if !ok || profile.Kind == "default" {
		return OpenURL(rawURL)
	}
	if rawURL == "" {
		return fmt.Errorf("URL is empty")
	}
	if runtime.GOOS != "darwin" {
		return OpenURL(rawURL)
	}
	browser := browserAppName(profile.Browser)
	if browser == "" || browser == "default" {
		return OpenURL(rawURL)
	}
	browserArgs, profileDir, err := profileBrowserArgs(rawURL, profile, browser)
	if err != nil {
		return err
	}
	if profileDir != "" {
		if err := os.MkdirAll(profileDir, 0o700); err != nil {
			return err
		}
	}
	if executable, ok := browserExecutablePath(browser); ok {
		return exec.Command(executable, browserArgs...).Start()
	}
	args := []string{"-na", browser, "--args"}
	args = append(args, browserArgs...)
	return exec.Command("open", args...).Start()
}

func profileBrowserArgs(rawURL string, profile BrowserProfile, browser string) ([]string, string, error) {
	args := []string{}
	profileDir := ""
	switch profile.Kind {
	case "managed":
		path := profile.Path
		if path == "" {
			path = DefaultBrowserProfilePath(profile.Name)
		}
		profileDir = path
		if isFirefoxBrowser(browser) {
			args = append(args, "--new-instance", "--profile", path)
		} else if isArcBrowser(browser) {
			return nil, "", fmt.Errorf("managed profiles are not supported for Arc; use default profile or Chrome/Brave/Edge/Firefox")
		} else {
			args = append(args, "--user-data-dir="+path)
		}
	case "existing":
		if profile.Directory != "" && !isFirefoxBrowser(browser) {
			args = append(args, "--profile-directory="+profile.Directory)
		}
		if profile.Path != "" && isFirefoxBrowser(browser) {
			args = append(args, "--profile", profile.Path)
		}
	}
	args = append(args, profile.Args...)
	args = append(args, "--new-window", rawURL)
	return args, profileDir, nil
}

func DefaultBrowserProfilePath(name string) string {
	dir := defaultConfigDir()
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "brmk", "browser-profiles", sanitizeProfileName(name))
}

func browserExecutablePath(browser string) (string, bool) {
	for _, path := range browserExecutableCandidates(browser) {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func browserExecutableCandidates(browser string) []string {
	executable := browserExecutableName(browser)
	if executable == "" {
		return nil
	}
	appName := browserAppName(browser)
	if appName == "" || appName == "default" {
		return nil
	}
	appDirs := []string{"/Applications"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		appDirs = append(appDirs, filepath.Join(home, "Applications"))
	}
	candidates := make([]string, 0, len(appDirs))
	for _, dir := range appDirs {
		candidates = append(candidates, filepath.Join(dir, appName+".app", "Contents", "MacOS", executable))
	}
	return candidates
}

func browserExecutableName(browser string) string {
	switch browserAppName(browser) {
	case "", "default":
		return ""
	case "Firefox":
		return "firefox"
	default:
		return browserAppName(browser)
	}
}

func browserAppName(browser string) string {
	switch strings.ToLower(strings.TrimSpace(browser)) {
	case "", "default":
		return "default"
	case "chrome", "google chrome":
		return "Google Chrome"
	case "brave", "brave browser":
		return "Brave Browser"
	case "edge", "microsoft edge":
		return "Microsoft Edge"
	case "firefox", "mozilla firefox":
		return "Firefox"
	case "arc":
		return "Arc"
	default:
		return strings.TrimSpace(browser)
	}
}

func isFirefoxBrowser(browser string) bool {
	return strings.EqualFold(browser, "Firefox") || strings.Contains(strings.ToLower(browser), "firefox")
}

func isArcBrowser(browser string) bool {
	return strings.EqualFold(browser, "Arc")
}

func sanitizeProfileName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "profile"
	}
	return b.String()
}
