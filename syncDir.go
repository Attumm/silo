package main

import (
	"fmt"
	"time"
	"io/ioutil"
	"path/filepath"
	"log"
)

func syncFiles(path string) {
	var start time.Time
	var changeFound bool
	for {
		start = time.Now()
		fileChan := make(chan *File, 100)
		go DirWalk(path, fileChan, true)
		items := make(map[string]*File)

		changeFound = false
		for file := range fileChan {
			filePath := file.rellFullPath()
			if _, found := Cache.Get(filePath); !found {
				changeFound = true
			}
			items[filePath] = file
		}
		if changeFound || len(items) != Cache.Length() {
			fmt.Println("update items", changeFound, len(items), Cache.Length())
			Cache.Update(items)
		}
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
			RelPath: path[len(SETTINGS.Base):] + string(filepath.Separator),
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
