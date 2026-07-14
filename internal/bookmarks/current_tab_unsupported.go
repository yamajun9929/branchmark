//go:build !darwin

package bookmarks

import "fmt"

type BrowserTab struct {
	Title string
	URL   string
}

func CurrentTab(browser string) (BrowserTab, error) {
	return BrowserTab{}, fmt.Errorf("current browser tab capture is only supported on macOS")
}
