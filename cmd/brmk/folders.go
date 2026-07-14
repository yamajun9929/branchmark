package main

import (
	"sort"
	"strings"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
)

// folderPaths returns every folder below the root as slash-separated paths.
// The first path component is a Space.
func folderPaths(store *bookmarks.Store) []string {
	if store == nil || store.Root == nil {
		return nil
	}

	paths := make([]string, 0)
	var walk func(*bookmarks.Node, []string)
	walk = func(parent *bookmarks.Node, prefix []string) {
		for _, child := range parent.Children {
			if child == nil || !child.IsFolder() {
				continue
			}
			title := strings.TrimSpace(child.Title)
			if title == "" {
				continue
			}
			path := append(append([]string(nil), prefix...), title)
			paths = append(paths, strings.Join(path, "/"))
			walk(child, path)
		}
	}
	walk(store.Root, nil)
	sort.Strings(paths)
	return paths
}
