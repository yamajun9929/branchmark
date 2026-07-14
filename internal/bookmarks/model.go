package bookmarks

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

const (
	NodeFolder   = "folder"
	NodeBookmark = "bookmark"
)

type Store struct {
	Version int   `json:"version"`
	Root    *Node `json:"root"`
}

type Node struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	URL       string    `json:"url,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Expanded  bool      `json:"expanded"`
	Children  []*Node   `json:"children,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewStore() *Store {
	root := NewFolder("Bookmarks")
	root.ID = "root"
	root.Expanded = true
	return &Store{Version: 1, Root: root}
}

func NewFolder(title string) *Node {
	now := time.Now().UTC()
	return &Node{
		ID:        newID(),
		Type:      NodeFolder,
		Title:     strings.TrimSpace(title),
		Expanded:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewBookmark(title, url string, tags []string) *Node {
	now := time.Now().UTC()
	title = strings.TrimSpace(title)
	if title == "" {
		title = strings.TrimSpace(url)
	}
	return &Node{
		ID:        newID(),
		Type:      NodeBookmark,
		Title:     title,
		URL:       strings.TrimSpace(url),
		Tags:      CleanTags(tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (n *Node) IsFolder() bool {
	return n != nil && n.Type == NodeFolder
}

func (n *Node) IsBookmark() bool {
	return n != nil && n.Type == NodeBookmark
}

func (n *Node) Touch() {
	if n != nil {
		n.UpdatedAt = time.Now().UTC()
	}
}

func Normalize(s *Store) *Store {
	if s == nil {
		return NewStore()
	}
	if s.Version == 0 {
		s.Version = 1
	}
	if s.Root == nil {
		s.Root = NewStore().Root
	}
	s.Root.ID = "root"
	s.Root.Type = NodeFolder
	if strings.TrimSpace(s.Root.Title) == "" {
		s.Root.Title = "Bookmarks"
	}
	seen := map[string]bool{}
	normalizeNode(s.Root, seen)
	return s
}

func normalizeNode(n *Node, seen map[string]bool) {
	if n == nil {
		return
	}
	if n.Type != NodeFolder && n.Type != NodeBookmark {
		if n.URL == "" {
			n.Type = NodeFolder
		} else {
			n.Type = NodeBookmark
		}
	}
	if strings.TrimSpace(n.ID) == "" || seen[n.ID] {
		n.ID = newID()
	}
	seen[n.ID] = true
	n.Title = strings.TrimSpace(n.Title)
	n.URL = strings.TrimSpace(n.URL)
	n.Tags = CleanTags(n.Tags)
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}
	if n.Type == NodeBookmark {
		n.Children = nil
		return
	}
	for _, child := range n.Children {
		normalizeNode(child, seen)
	}
}

func CleanTags(tags []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.TrimPrefix(tag, "#"))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		cleaned = append(cleaned, tag)
	}
	return cleaned
}

func SplitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return CleanTags(strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	}))
}

func Find(s *Store, id string) (*Node, *Node, int) {
	if s == nil || s.Root == nil {
		return nil, nil, -1
	}
	if id == s.Root.ID {
		return s.Root, nil, -1
	}
	return findIn(s.Root, id)
}

func findIn(parent *Node, id string) (*Node, *Node, int) {
	for i, child := range parent.Children {
		if child.ID == id {
			return child, parent, i
		}
		if child.IsFolder() {
			if found, foundParent, foundIndex := findIn(child, id); found != nil {
				return found, foundParent, foundIndex
			}
		}
	}
	return nil, nil, -1
}

func Remove(s *Store, id string) (*Node, bool) {
	node, parent, index := Find(s, id)
	if node == nil || parent == nil || index < 0 {
		return nil, false
	}
	parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
	parent.Touch()
	return node, true
}

func FindOrCreateFolderPath(s *Store, rawPath string) *Node {
	Normalize(s)
	current := s.Root
	for _, part := range strings.Split(rawPath, "/") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var next *Node
		for _, child := range current.Children {
			if child.IsFolder() && strings.EqualFold(child.Title, part) {
				next = child
				break
			}
		}
		if next == nil {
			next = NewFolder(part)
			current.Children = append(current.Children, next)
			current.Touch()
		}
		next.Expanded = true
		current = next
	}
	return current
}

func CountBookmarks(n *Node) int {
	if n == nil {
		return 0
	}
	if n.IsBookmark() {
		return 1
	}
	total := 0
	for _, child := range n.Children {
		total += CountBookmarks(child)
	}
	return total
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}
