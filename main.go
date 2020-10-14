package main

/* README
To make use of this application, create new user, without login and homedir.
Create new directory and make new user owner of the directory.
Set SETTINGS.Base to absolute path of directory.
Run this application only with the new user.

This will prevent many security issues.
*/

// TODO list
// ADD Context package, for lifecycle and cancellation of requests
// ADD something of users, or inbox

// Maybe ADD META through Shadow files with meta data

import (
	"fmt"
	"log"
	"net/http"
)

var Cache = &CacheFiles{Items: make(CacheMap)}

func init() {

	SETTINGS.SetParsed("base", "/app/files", "set the basedir", BasePathParser)
	SETTINGS.Set("host", "0.0.0.0:8000", "enter host with port")
	SETTINGS.Set("cors", "not-set", "Domains whitelisted under cors")
	SETTINGS.SetInt("sync", 600, "Pauze between directory cache syncs, in seconds")

	SETTINGS.Parse()
}

func main() {

	fmt.Println("Start server:", SETTINGS.Get("host"))
	fmt.Println("File path:", SETTINGS.Get("base"))

	fmt.Println("Sync pauze, seconds:", SETTINGS.GetInt("sync"))

	go syncFiles(SETTINGS.Get("base"))

	//Rest Api
	http.HandleFunc("/list/group/", listGroupedRest)
	http.HandleFunc("/list/", listRest)

	http.HandleFunc("/detail/", detailRest)
	http.HandleFunc("/content/", contentRest)
	http.HandleFunc("/upload/", uploadRest)
	http.HandleFunc("/delete/", deleteRest)

	http.HandleFunc("/cycle/", cycleRest)

	// Vue view
	http.HandleFunc("/", indexView)

	//HTML Views
	http.HandleFunc("/items", itemsView)
	http.HandleFunc("/view/", viewView)
	http.HandleFunc("/video/", videoView)
	http.HandleFunc("/add/", addView)

	log.Fatal(http.ListenAndServe(SETTINGS.Get("host"), nil))
}
