package main

import (
	"sync"
	"time"
	"strings"
	"path/filepath"
	"net/url"
	"strconv"
)

// Global Cache Readonly for user input.
//Sync goroutine is responsible for updating the cache
type CacheMap map[string]*File
type CacheFiles struct {
	Cycle int64
	Items CacheMap
	Mu    sync.RWMutex
}
type ListFiles []ListFile

func (c *CacheFiles) Set(k string, f *File) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Items[k] = f
}

func (c *CacheFiles) Get(k string) (*File, bool) {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	found, ok := c.Items[k]
	return found, ok
}

func (c *CacheFiles) Update(newItems CacheMap) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Cycle = time.Now().UnixNano()
	c.Items = newItems
}

func (c CacheFiles) Length() int {
	return len(c.Items)
}

func (c *CacheFiles) LastCycle() int64 {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return c.Cycle
}

func (c *CacheFiles) LastCycleSec() int {
	return int(c.LastCycle()/1000000) 
}

// File And ListFile Structs and functions
type File struct {
	Name    string
	Size    int64
	AbsPath string
	RelPath string
	IsDir   bool
	ModDate int64
}

type ListFile struct {
	Name        string
	ModDate     int64
	SizeBytes   string
	IsDir       bool
	Directories []string
	DetailURL   string
	ContentURL  string
	VideoURL    string
	ViewURL     string
}

type ListFileGrouped struct {
	Name        string
	ModDate     int64
	SizeBytes   string
	IsDir       bool
	Directories []string
	DetailURL   string
	ContentURL  string
	VideoURL    string
	ViewURL     string
	Grouped     []*ListFileGrouped
}

func ListFileToGrouped(items []ListFile) *ListFileGrouped {
	topLevel := &ListFileGrouped{
		Name:        "topLevel",
		ModDate:     1,
		SizeBytes:   "1",
		IsDir:       true,
		DetailURL:   "",
		ContentURL:  "",
		VideoURL:    "",
		ViewURL:     "",
		Directories: []string{},
		Grouped:     []*ListFileGrouped{},
	}

	dirs := make(map[string]*ListFileGrouped)
	groupedItems := []*ListFileGrouped{}
	for i := range items {
		itemGrouped := items[i].ListFileGrouped()
		groupedItems = append(groupedItems, itemGrouped)

		if itemGrouped.IsDir {
			tmp := strings.Join(itemGrouped.Directories, "") + itemGrouped.Name
			topLevel.Grouped = append(topLevel.Grouped, itemGrouped)
			dirs[tmp] = itemGrouped
		}
	}

	dirs["<---->"] = topLevel

	for i := range groupedItems {
		if groupedItems[i].IsDir {
			continue
		}
		a := dirs[strings.Join(groupedItems[i].Directories, "")]
		if a == nil {
			a = dirs["<---->"]
		}
		a.Grouped = append(a.Grouped, groupedItems[i])
	}
	return topLevel
}

func (f File) fullPath() string {
	return filepath.Join(f.AbsPath, f.Name)
}

func (f File) rellFullPath() string {
	return f.RelPath + f.Name
}

func (f File) urlEncoded() string {
	return url.PathEscape(f.rellFullPath())
}

func (f File) urlFor(s string) string {
	return "/" + s + f.urlEncoded()
}

func (f File) ListFile() ListFile {
	return ListFile{
		Name:        f.Name,
		ModDate:     f.ModDate,
		SizeBytes:   strconv.FormatInt(f.Size, 10),
		IsDir:       f.IsDir,
		DetailURL:   f.urlFor("detail"),
		ContentURL:  f.urlFor("content"),
		VideoURL:    f.urlFor("video"),
		ViewURL:     f.urlFor("view"),
		Directories: removeEmpty(strings.Split(f.RelPath, string(filepath.Separator))),
	}
}

func (f ListFile) ListFileGrouped() *ListFileGrouped {
	return &ListFileGrouped{
		Name:        f.Name,
		ModDate:     f.ModDate,
		SizeBytes:   f.SizeBytes,
		IsDir:       f.IsDir,
		DetailURL:   f.DetailURL,
		ContentURL:  f.ContentURL,
		VideoURL:    f.VideoURL,
		ViewURL:     f.ViewURL,
		Directories: f.Directories,
		Grouped:     []*ListFileGrouped{},
	}
}
