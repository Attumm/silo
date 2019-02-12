package main

import (
	"strings"
	"sort"
)

// Response structs
type ErrorMsg struct {
	Error      string
	Reason     string
	HTTPStatus int
}

type UploadSuccesResponse struct {
	Message     string
	Filename    string
	ContentURL  string
	Directories []string
}


func filter(c *CacheFiles, filters []string) []ListFile {
	newItems := []ListFile{}
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	for _, file := range c.Items {
		if notContains(file.Name, filters) {
			newItems = append(newItems, file.ListFile())
		}
	}
	return newItems
}

func notContains(item string, searchTerms []string) bool {
	for _, searchTerm := range searchTerms {
		if !strings.Contains(item, searchTerm) {
			return false
		}
	}
	return true
}

func contains(item string, searchTerms []string) bool {
	for _, searchTerm := range searchTerms {
		if strings.Contains(item, searchTerm) {
			return true
		}
	}
	return false
}

func exclude(items []ListFile, filters []string) []ListFile {
	newItems := []ListFile{}
	for _, file := range items {
		if !contains(file.Name, filters) {
			newItems = append(newItems, file)
		}
	}
	return newItems
}

func list(c *CacheFiles) []ListFile {
	newItems := []ListFile{}
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	for _, file := range c.Items {
		newItems = append(newItems, file.ListFile())
	}
	return newItems
}

func listTypeAhead(c *CacheFiles, prefix string) []ListFile {
	newItems := []ListFile{}
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	for _, file := range c.Items {
		if strings.HasPrefix(file.Name, prefix) {
			newItems = append(newItems, file.ListFile())
		}
	}
	return newItems
}

func subDir(full, sub []string) bool {
	if len(full) < len(sub) {
		return false
	}
	for i, v := range sub {
		if v != full[i] {
			return false
		}
	}
	return true
}

func filterDirs(items []ListFile, filters []string) []ListFile {
	newItems := []ListFile{}
	for _, item := range items {
		if subDir(item.Directories, filters) {
			newItems = append(newItems, item)
		}
	}
	return newItems
}

func sortBy(items []ListFile, attr string) {
	sortFuncs := map[string]func(int, int) bool{
		"name":  func(i, j int) bool { return items[i].Name < items[j].Name },
		"-name": func(i, j int) bool { return items[i].Name > items[j].Name },
		"-date": func(i, j int) bool { return items[i].ModDate < items[j].ModDate },
		"date":  func(i, j int) bool { return items[i].ModDate > items[j].ModDate },
	}
	if sortFunc, found := sortFuncs[attr]; found {
		sort.Slice(items, sortFunc)
	}
}
