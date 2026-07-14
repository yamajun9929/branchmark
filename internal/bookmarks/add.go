package bookmarks

import "strings"

func AddBookmark(dataPath, space, title, rawURL string, tags []string) (*Node, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, ErrURLRequired
	}
	store, err := Load(dataPath)
	if err != nil {
		return nil, err
	}
	parent := FindOrCreateFolderPath(store, space)
	node := NewBookmark(title, rawURL, tags)
	parent.Children = append(parent.Children, node)
	parent.Expanded = true
	parent.Touch()
	if err := Save(dataPath, store); err != nil {
		return nil, err
	}
	return node, nil
}
