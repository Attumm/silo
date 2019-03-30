package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ErrorResponse(w http.ResponseWriter, reason string, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(ErrorMsg{Error: "Error", Reason: reason, HTTPStatus: httpStatus})
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
	}
	listItems = listItems[start:end]
	if len(listItems) < limit {
		return listItems
	}
	return listItems[:limit]
}

// REST API functions
func listRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	setHeader(w)
	items := handleParameters(w, r)

	w.Header().Set("Total-Items", strconv.Itoa(len(items)))
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(items)
}

func listGroupedRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	setHeader(w)
	items := handleParameters(w, r)

	w.Header().Set("Total-Items", strconv.Itoa(len(items)))
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(ListFileToGrouped(items))
}

func detailRest(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/detail"):]
	w.Header().Set("Content-Type", "application/json")
	setHeader(w)
	w.WriteHeader(http.StatusOK)
	// TODO missing handling of 404
	file, ok := Cache.Get(filename)
	if !ok {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(file.ListFile())
}

func contentRest(w http.ResponseWriter, r *http.Request) {
	setHeader(w)
	filename, err := url.PathUnescape(r.URL.Path[len("/content"):])
	if err != nil {
		ErrorResponse(w, "Unable to parse URL", http.StatusBadRequest)
		return
	}
	// To run without CacheFiles, and your own peril.
	// You'll be ungaurded against path traversal attacks are possible
	// http.ServeFile(w, r, filepath.Join(SETTINGS.Base, filename))
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
	setHeader(w)
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
	setHeader(w)
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
	fp := filepath.Join(SETTINGS.Get("base"), strings.Join(dirs, string(filepath.Separator)), clFilename)
	f, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		ErrorResponse(w, "Unable to store file", http.StatusBadRequest)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	absPath := f.Name()
	filename := FilenameFromAbsPath(absPath)
	rellFullPath := absPath[len(SETTINGS.Get("base")):]

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UploadSuccesResponse{
		Message:     "Upload succeeded",
		Filename:    filename,
		ContentURL:  "/" + "content/" + url.PathEscape(rellFullPath),
		Directories: removeEmpty(strings.Split(rellFullPath[:len(rellFullPath)-len(filename)], "/")),
	})
}

func cycleRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	setHeader(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(strconv.Itoa(Cache.LastCycleSec()))
}

// HTML VIEWS functions
func indexView(w http.ResponseWriter, r *http.Request) {
	setHeader(w)
	http.ServeFile(w, r, "index.html")
}

func itemsView(w http.ResponseWriter, r *http.Request) {
	setHeader(w)
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
	setHeader(w)
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
	setHeader(w)
	filename, err := url.PathUnescape(r.URL.Path[len("/video"):])
	if err != nil {
		ErrorResponse(w, "Unable to parse URL", http.StatusBadRequest)
		log.Fatal("Unable to Parse URL")
	}
	file, found := Cache.Get(filename)

	if !found {
		ErrorResponse(w, "File not found", http.StatusNotFound)
		log.Fatal("File not found")
	}
	tmpl := template.New("page")
	// TODO store templates in a map
	t, err := tmpl.Parse(`
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

func addView(w http.ResponseWriter, r *http.Request) {
	setHeader(w)
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
