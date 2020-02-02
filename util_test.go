package main

import (
	"reflect"
	"testing"
)

func TestRemoveEmpty(t *testing.T) {
	testcases := []struct {
		input    []string
		expected []string
	}{
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"", "b", "c"}, []string{"b", "c"}},
		{[]string{"", "", "c"}, []string{"c"}},
		{[]string{"", "", ""}, []string{}},
		{[]string{"abc", "", ""}, []string{"abc"}},
		{[]string{"abc", "def", ""}, []string{"abc", "def"}},
		{[]string{"abc", "def", "ghi"}, []string{"abc", "def", "ghi"}},
	}

	for tcNumber, testcase := range testcases {
		result := removeEmpty(testcase.input)
		if !reflect.DeepEqual(result, testcase.expected) {
			t.Error("testcase", tcNumber, "expected", testcase.expected, "!=", result)
		}
	}
}

func TestStripTrailingSlash(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/", ""},
		{"//", "/"},
		{"a/b/c", "a/b/c"},
		{"a/b/c/", "a/b/c"},
		{"/a/b/c", "/a/b/c"},
	}

	for tcNumber, testcase := range testcases {
		result := stripTrailingSlash(testcase.input)
		if result != testcase.expected {
			t.Error("testcase", tcNumber, "expected", testcase.expected, "!=", result)
		}
	}
}
