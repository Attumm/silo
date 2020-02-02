package main

// utils

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Remove all chars that are not lower or uppercase alphabet
// And allow for one dot in succesion.
// ord 65 to 90 uppercase alphabet
// ord 97 to 122 lowercase alphabet
// ord 46 == '.'
func cleanFilename(s string) string {

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

// Return slice with strings that are not empty
func removeEmpty(l []string) []string {
	nonEmpty := []string{}
	for _, v := range l {
		if len(v) > 0 {
			nonEmpty = append(nonEmpty, v)
		}
	}
	return nonEmpty
}

// Remove the path, and return the filename
func FilenameFromAbsPath(absPath string) string {
	items := strings.Split(absPath, string(filepath.Separator))
	return items[len(items)-1]
}

// Parse int from string
// if parsed int or error return default value
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

// Parses input for basepath
func BasePathParser(s string) string {
	return stripTrailingSlash(s)
}

// strips Tralling Slash from string
func stripTrailingSlash(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
