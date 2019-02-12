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
	"flag"
	"fmt"
	"log"
	"net/http"
)

var Cache = &CacheFiles{Items: make(CacheMap)}
var SETTINGS = Settings{}

func init() {
	var base string
	var host string
	var corsDomains string
	var syncPauze int

	flag.StringVar(&base, "base", "/app/files", "set the basedir")
	flag.StringVar(&host, "host", "0.0.0.0:8000", "enter host with port")
	flag.StringVar(&corsDomains, "cors", "", "Domains whitelisted under cors")
	flag.IntVar(&syncPauze, "sync", 1, "Pauze between directory cache syncs, in seconds")

	flag.Parse()

	SETTINGS.Base = base
	SETTINGS.Host = host
	SETTINGS.CORSSet = len(corsDomains) > 1
	SETTINGS.CORSDomains = corsDomains
	SETTINGS.SyncPauze = syncPauze
}

func main() {

	fmt.Println("Start server:", SETTINGS.Host)
	fmt.Println("File path:", SETTINGS.Host)
	fmt.Println("Sync pauze, seconds:", SETTINGS.SyncPauze)

	go syncFiles(SETTINGS.Base)

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

	log.Fatal(http.ListenAndServe(SETTINGS.Host, nil))
}
