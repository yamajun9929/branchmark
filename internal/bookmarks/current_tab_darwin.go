//go:build darwin

package bookmarks

import (
	"fmt"
	"os/exec"
	"strings"
)

type BrowserTab struct {
	Title string
	URL   string
}

func CurrentTab(browser string) (BrowserTab, error) {
	app := currentTabAppName(browser)
	if app == "" {
		return BrowserTab{}, fmt.Errorf("browser is required")
	}

	script := chromiumCurrentTabScript(app)
	if strings.EqualFold(app, "Safari") {
		script = safariCurrentTabScript(app)
	}

	output, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			detail := strings.TrimSpace(string(exitErr.Stderr))
			if detail != "" {
				return BrowserTab{}, fmt.Errorf("get current tab from %s: %s", app, detail)
			}
		}
		return BrowserTab{}, fmt.Errorf("get current tab from %s: %w", app, err)
	}

	parts := strings.SplitN(strings.TrimRight(string(output), "\r\n"), "\x1f", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return BrowserTab{}, fmt.Errorf("get current tab from %s: unexpected response", app)
	}
	return BrowserTab{
		Title: strings.TrimSpace(parts[0]),
		URL:   strings.TrimSpace(parts[1]),
	}, nil
}

func currentTabAppName(browser string) string {
	switch strings.ToLower(strings.TrimSpace(browser)) {
	case "", "chrome", "google chrome":
		return "Google Chrome"
	case "brave", "brave browser":
		return "Brave Browser"
	case "edge", "microsoft edge":
		return "Microsoft Edge"
	case "safari":
		return "Safari"
	case "arc":
		return "Arc"
	default:
		return strings.TrimSpace(browser)
	}
}

func chromiumCurrentTabScript(app string) string {
	quotedApp := appleScriptQuote(app)
	return fmt.Sprintf(`
tell application %s
	if not (exists front window) then error "no browser window"
	set currentTab to active tab of front window
	return (title of currentTab) & (ASCII character 31) & (URL of currentTab)
end tell
`, quotedApp)
}

func safariCurrentTabScript(app string) string {
	quotedApp := appleScriptQuote(app)
	return fmt.Sprintf(`
tell application %s
	if not (exists front window) then error "no browser window"
	set currentTab to current tab of front window
	return (name of currentTab) & (ASCII character 31) & (URL of currentTab)
end tell
`, quotedApp)
}

func appleScriptQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
