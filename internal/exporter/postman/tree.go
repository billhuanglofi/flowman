package postman

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

type requestEntry struct {
	path    string
	folders []string
	item    collectionItem
}

type treeNode struct {
	name     string
	request  *collectionItem
	children map[string]*treeNode
	order    []string
}

func buildEntries(workspace model.Workspace) []requestEntry {
	entries := make([]requestEntry, 0, len(workspace.Requests))
	for index, request := range workspace.Requests {
		path := filepath.ToSlash(workspace.Project.Requests[index])
		entries = append(entries, requestEntry{
			path:    path,
			folders: requestFolders(path),
			item:    collectionItem{Name: request.Name, Request: exportRequest(request)},
		})
	}
	sort.Slice(entries, func(left int, right int) bool {
		return entries[left].path < entries[right].path
	})
	return entries
}

func requestFolders(path string) []string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return nil
	}
	return append([]string(nil), parts[1:len(parts)-1]...)
}

func nestEntries(entries []requestEntry) []collectionItem {
	root := &treeNode{name: ""}
	for _, entry := range entries {
		root.add(entry)
	}
	return root.items()
}

func (node *treeNode) add(entry requestEntry) {
	current := node
	for _, folder := range entry.folders {
		if current.children == nil {
			current.children = map[string]*treeNode{}
		}
		child, ok := current.children[folder]
		if !ok {
			child = &treeNode{name: folder}
			current.children[folder] = child
			current.order = append(current.order, folder)
		}
		current = child
	}
	if current.children == nil {
		current.children = map[string]*treeNode{}
	}
	leaf := entry.item
	current.children[entry.item.Name] = &treeNode{name: entry.item.Name, request: &leaf}
	current.order = append(current.order, entry.item.Name)
}

func (node *treeNode) items() []collectionItem {
	items := make([]collectionItem, 0, len(node.order))
	seen := map[string]bool{}
	for _, name := range node.order {
		if seen[name] {
			continue
		}
		seen[name] = true
		child := node.children[name]
		if child.request != nil {
			items = append(items, *child.request)
			continue
		}
		items = append(items, collectionItem{Name: child.name, Item: child.items()})
	}
	sort.Slice(items, func(left int, right int) bool {
		return items[left].Name < items[right].Name
	})
	return items
}
