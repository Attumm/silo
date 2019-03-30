package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

func syncFiles(path string) {
	var start time.Time
	var updateCache bool
	for {
		start = time.Now()
		fileChan := make(chan *File, 100)
		go DirWalk(path, fileChan, true)
		items := make(map[string]*File)

		updateCache = false
		for file := range fileChan {
			filePath := file.rellFullPath()
			cachedFile, found := Cache.Get(filePath)
			if !found || cachedFile.ModDate != file.ModDate {
				updateCache = true
				file.SetContentType()
				items[filePath] = file
			} else {
				items[filePath] = cachedFile
			}
		}
		if updateCache || len(items) != Cache.Length() {
			fmt.Println("update items", updateCache, len(items), Cache.Length())
			Cache.Update(items)
		}
		fmt.Println("ingestion took:", time.Now().Sub(start))
		time.Sleep(time.Second * time.Duration(SETTINGS.GetInt("sync")))
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
	basePath := SETTINGS.Get("base")

	for _, file := range files {
		// TODO check mod time, to skip unchanged files.
		fileChan <- &File{
			Name:    file.Name(),
			ModDate: file.ModTime().Unix(),
			Size:    file.Size(),
			AbsPath: absPath,
			RelPath: path[len(basePath):] + string(filepath.Separator),
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
