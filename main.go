package main

/* README
To make use of this application, create new user, without login and homedir.
Create new directory and make new user owner of the directory.
Set BASEDIR to absolute path of directory.
Run this application only with the new user.

This will prevent many security issues.
*/

// TODO list
// ADD Context package, for lifecycle and cancellation of requests
// ADD something of users, or inbox

// Maybe ADD META through Shadow files with meta data

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Globals
//var BASEDIR = "/home/meneer/Downloads"
var BASEDIR = "/home/meneer/Development/silo/files"

// Global Cache Readonly for user input.
//Sync goroutine is responsible for updating the cache
type CacheMap map[string]*File
type CacheFiles struct {
	Cycle int64
	Items CacheMap
	Mu    sync.RWMutex
}

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
	now := time.Now()
	defer c.Mu.Unlock()
	c.Items = newItems
	c.Cycle = now.UnixNano()
}

var Cache = &CacheFiles{Items: make(CacheMap)}

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

// syncFiles responsible for updating the cache

func syncFiles(path string) {
	var start time.Time
	for {
		start = time.Now()
		fileChan := make(chan *File, 100)
		go DirWalk(path, fileChan, true)
		items := make(map[string]*File)
		for file := range fileChan {
			items[file.rellFullPath()] = file
		}
		Cache.Update(items)
		fmt.Println("ingestion took:", time.Now().Sub(start))

		time.Sleep(time.Second * 1)
	}
}

func DirWalk(path string, fileChan chan *File, toplevel bool) {
	var absPath string
	if filepath.IsAbs(path) {
		absPath = path
	} else {
		var err error
		absPath, err = filepath.Abs(path)
		if err != nil {
			log.Fatal(err)
		}
	}
	files, err := ioutil.ReadDir(absPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		// TODO check mod time, to skip unchanged files.
		fileChan <- &File{
			Name:    file.Name(),
			ModDate: file.ModTime().Unix(),
			Size:    file.Size(),
			AbsPath: absPath,
			RelPath: path[len(BASEDIR):] + string(filepath.Separator),
			IsDir:   file.IsDir(),
		}
		if file.IsDir() {
			DirWalk(filepath.Join(path, file.Name()), fileChan, false)
		}
	}
	if toplevel {
		close(fileChan)
	}
}

// Response structs
type ErrorMsg struct {
	Error      string
	Reason     string
	HttpStatus int
}

type UploadSuccesResponse struct {
	Message     string
	Filename    string
	ContentURL  string
	Directories []string
}

func ErrorResponse(w http.ResponseWriter, reason string, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(ErrorMsg{Error: "Error", Reason: reason, HttpStatus: httpStatus})
}

// Util Functions

func removeEmpty(l []string) []string {
	nonEmpty := []string{}
	for _, v := range l {
		if len(v) > 0 {
			nonEmpty = append(nonEmpty, v)
		}
	}
	return nonEmpty
}

func FilenameFromAbsPath(absPath string) string {
	items := strings.Split(absPath, string(filepath.Separator))
	return items[len(items)-1]
}

func intMoreDefault(s string, defaultN int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	if n < defaultN {
		return defaultN
	}
	return n
}

func handleParameters(w http.ResponseWriter, r *http.Request) []ListFile {
	filters, filterGiven := r.URL.Query()["filter"]
	dirs, dirsGiven := r.URL.Query()["dirs"]
	orderby, orderByGiven := r.URL.Query()["orderby"]
	excludes, excludeGiven := r.URL.Query()["exclude"]
	limitStr, limitGiven := r.URL.Query()["limit"]
	pageStr, pageGiven := r.URL.Query()["page"]
	pageSizeStr, pageSizeGiven := r.URL.Query()["pagesize"]
	typeAhead, typeAheadGiven := r.URL.Query()["typeahead"]

	var listItems []ListFile

	//TODO make generalist filter function that can take many filter option and loops one
	if filterGiven {
		listItems = filter(Cache, filters)
	} else if typeAheadGiven {
		listItems = listTypeAhead(Cache, typeAhead[0])
		sortBy(listItems, "name")
	} else {
		listItems = list(Cache)
	}

	if excludeGiven {
		listItems = exclude(listItems, excludes)
	}

	if dirsGiven {
		listItems = filterDirs(listItems, dirs)
	}

	if orderByGiven {
		sortBy(listItems, orderby[0])
	}

	if !limitGiven && !pageGiven {
		return listItems
	}

	limit := len(listItems)
	if limitGiven {
		limit = intMoreDefault(limitStr[0], 1)
	}

	pageSize := 10
	if pageSizeGiven {
		pageSize = intMoreDefault(pageSizeStr[0], 1)
	}

	page := 1
	if pageGiven {
		page = intMoreDefault(pageStr[0], 1)
	}
	w.Header().Set("Page", strconv.Itoa(page))
	w.Header().Set("Page-Size", strconv.Itoa(pageSize))
	//TODO fix below
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(listItems) {
		end = len(listItems)
	}
	if len(listItems) <= limit {
		return listItems[start:end]
	} else {
		listItems = listItems[start:end]
		if len(listItems) < limit {
			return listItems
		}
		return listItems[:limit]
	}
}

// REST API functions
func listRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	items := handleParameters(w, r)

	w.Header().Set("Total-Items", strconv.Itoa(len(items)))
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(items)
}

func listGroupedRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	items := handleParameters(w, r)

	w.Header().Set("Total-Items", strconv.Itoa(len(items)))
	w.WriteHeader(http.StatusOK)

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
	for i, _ := range items {
		itemGrouped := items[i].ListFileGrouped()
		groupedItems = append(groupedItems, itemGrouped)

		if itemGrouped.IsDir {
			tmp := strings.Join(itemGrouped.Directories, "") + itemGrouped.Name
			topLevel.Grouped = append(topLevel.Grouped, itemGrouped)
			dirs[tmp] = itemGrouped
		}
	}

	dirs["<---->"] = topLevel

	for i, _ := range groupedItems {
		if groupedItems[i].IsDir {
			continue
		}
		a := dirs[strings.Join(groupedItems[i].Directories, "")]
		if a == nil {
			a = dirs["<---->"]
		}
		a.Grouped = append(a.Grouped, groupedItems[i])
	}
	json.NewEncoder(w).Encode(topLevel)
}

func detailRest(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/detail"):]
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	w.WriteHeader(http.StatusOK)
	// TODO missing handling of 404
	file, ok := Cache.Get(filename)
	if !ok {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		return
	} else {
		json.NewEncoder(w).Encode(file.ListFile())
	}
}

func contentRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	filename, err := url.PathUnescape(r.URL.Path[len("/content"):])
	if err != nil {
		ErrorResponse(w, "Unable to parse URL", http.StatusBadRequest)
		return
	}
	// To run without CacheFiles, and your own peril.
	// You'll be ungaurded against path traversal attacks are possible
	// http.ServeFile(w, r, filepath.Join(BASEDIR, filename))
	// uncomment the line above, comment out or remove everything below

	file, found := Cache.Get(filename)
	if !found {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		return

	}
	switch r.Method {
	case http.MethodGet:
		http.ServeFile(w, r, file.fullPath())
		return
	case http.MethodDelete:
		if err := os.Remove(file.fullPath()); err != nil {
			ErrorResponse(w, "File not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		http.ServeFile(w, r, file.fullPath())
		return
	}
}

func deleteRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	filename, err := url.PathUnescape(r.URL.Path[len("/delete"):])
	if err != nil {
		ErrorResponse(w, "Unable to parse URL", http.StatusBadRequest)
		return
	}
	file, found := Cache.Get(filename)
	if !found {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		return
	}
	if err := os.Remove(file.fullPath()); err != nil {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func uploadRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		ErrorResponse(w, "Unable to handle Multipart form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := r.ParseForm(); err != nil {
		ErrorResponse(w, "Unable to handle form", http.StatusBadRequest)
		return
	}

	dirs := []string{}
	if val, ok := r.PostForm["dirs[]"]; ok {
		dirs = val
	}

	// TODO add validation on cleaned filename
	clFilename := cleanFilename(handler.Filename)
	fp := filepath.Join(BASEDIR, strings.Join(dirs, string(filepath.Separator)), clFilename)
	f, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		ErrorResponse(w, "Unable to store file", http.StatusBadRequest)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	absPath := f.Name()
	filename := FilenameFromAbsPath(absPath)
	rellFullPath := absPath[len(BASEDIR):]

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UploadSuccesResponse{
		Message:     "Upload succeeded",
		Filename:    filename,
		ContentURL:  "/" + "content/" + url.PathEscape(rellFullPath),
		Directories: removeEmpty(strings.Split(rellFullPath[:len(rellFullPath)-len(filename)], "/")),
	})
}

// HTML VIEWS functions

func indexView(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func itemsView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	listItems := handleParameters(w, r)
	items := []string{}
	for _, item := range listItems {
		items = append(items, fmt.Sprintf("<a href=\"%s\">%s</a>", item.ViewURL, item.Name))
	}

	// TODO add template
	fmt.Fprint(w, fmt.Sprintf(`
	<html>
	  <body>
		%s
	  </body>
	</html>`, strings.Join(items, "<br>")))
}

func viewView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	filename, err := url.PathUnescape(r.URL.Path[len("/view"):])
	if err != nil {
		ErrorResponse(w, "Unable to parse URL", http.StatusBadRequest)
		return
	}
	if file, found := Cache.Get(filename); !found {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		return
	} else {
		// TODO add template
		body := fmt.Sprintf(`
		<h1>%s</h1>
		<p>Video <a href="%s">%s</a></p>
		<p>Content <a href="%s" download>download</a></p>
		<p>Image <img src="%s"></p>
		`,
			file.Name,
			("/" + "video" + file.urlEncoded()), file.Name,
			("/" + "content" + file.urlEncoded()), ("/" + "content" + file.urlEncoded()))

		response := fmt.Sprintf(`
	<html>
	<body>
		%s
	</body>
	</html>`, body)
		fmt.Fprint(w, response)
	}
}

func videoView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	filename, err := url.PathUnescape(r.URL.Path[len("/video"):])
	if err != nil {
		ErrorResponse(w, "Unable to parse URL", http.StatusBadRequest)
		return
	}
	if file, found := Cache.Get(filename); !found {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		return
	} else {
		t := template.New("page")
		// TODO store templates in a map
		t, err := t.Parse(`
	<html>
	<body>
		<video width="320" heigt="240"  autoplay controls>
		<source src="{{ .ContentURL }}">
	</body>
	</html>`)
		if err != nil {
			log.Fatal(err)
		}
		t.Execute(w, file.ListFile())
	}
}

func addView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Last-Update", strconv.FormatInt((Cache.Cycle/1000000), 10))
	// TODO add template
	response := fmt.Sprintf(`
		<html>
			<body>
				<form enctype="multipart/form-data" action="/upload/" method="post">
				<input type="file" name="uploadfile" />
				<input type="text" name="dirs[]" value=""/>
				<input type="submit" value="upload" />
			</form>
			</body>
			</html>
	`)
	fmt.Fprint(w, response)
}

func main() {
	// TODO refactor code around BASEDIR
	var base string
	var host string
	flag.StringVar(&base, "base", "/home/meneer/Development/silo/files", "set the basedir")
	flag.StringVar(&host, "host", "0.0.0.0:8000", "enter host with port")
	if base != BASEDIR {
		BASEDIR = base
	}
	flag.Parse()

	go syncFiles(base)

	//Rest Api
	http.HandleFunc("/list/group/", listGroupedRest)
	http.HandleFunc("/list/", listRest)

	http.HandleFunc("/detail/", detailRest)
	http.HandleFunc("/content/", contentRest)
	http.HandleFunc("/upload/", uploadRest)
	http.HandleFunc("/delete/", deleteRest)

	// Vue view
	http.HandleFunc("/", indexView)

	//HTML Views
	http.HandleFunc("/items", itemsView)
	http.HandleFunc("/view/", viewView)
	http.HandleFunc("/video/", videoView)
	http.HandleFunc("/add/", addView)

	log.Fatal(http.ListenAndServe(host, nil))
}

// utils

func cleanFilename(s string) string {
	// ord 65 to 90 uppercase alphabet
	// ord 97 to 122 lowercase alphabet
	// ord 46 == '.'
	rr := []rune(s)
	n := []rune{}
	f := false
	for _, r := range rr {
		if r >= 65 && r <= 90 {
			f = false
			n = append(n, r)
		} else if r >= 97 && r <= 122 {
			f = false
			n = append(n, r)
		} else if r == 46 && !f {
			f = true
			n = append(n, r)
		}
	}
	return string(n)
}
