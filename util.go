package main
// utils

import (
	"strings"
	"path/filepath"
	"strconv"
)


type Settings struct {
	CORSSet		bool
	CORSDomains	string
	Base		string
	Host		string
	SyncPauze	int
}

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
		} else if r >= 48 && r <= 57 {
			f = false
			n = append(n, r)
		} else if r == 45 || r == 95 {
			f = false
			n = append(n, r) 
		} else if r == 46 && !f {
			f = true
			n = append(n, r)
		}
	}
	return string(n)

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
