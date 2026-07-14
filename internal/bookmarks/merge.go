package bookmarks

import "strings"

func MergeStore(dst, src *Store) *Store {
	dst = Normalize(dst)
	src = Normalize(src)
	mergeChildren(dst.Root, src.Root.Children)
	return Normalize(dst)
}

func mergeChildren(dstParent *Node, srcChildren []*Node) {
	for _, child := range srcChildren {
		if child == nil {
			continue
		}
		if child.IsFolder() {
			if existing := findChildFolderByTitle(dstParent, child.Title); existing != nil {
				existing.Tags = mergeTags(existing.Tags, child.Tags)
				mergeChildren(existing, child.Children)
				existing.Expanded = true
				existing.Touch()
				continue
			}
		}
		dstParent.Children = append(dstParent.Children, child)
		dstParent.Touch()
	}
}

func findChildFolderByTitle(parent *Node, title string) *Node {
	if parent == nil {
		return nil
	}
	for _, child := range parent.Children {
		if child.IsFolder() && strings.EqualFold(child.Title, title) {
			return child
		}
	}
	return nil
}

func mergeTags(dst, src []string) []string {
	merged := append([]string(nil), dst...)
	merged = append(merged, src...)
	return CleanTags(merged)
}
