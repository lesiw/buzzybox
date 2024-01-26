package hive_test

import (
	"reflect"
	"strings"
	"testing"
)

func TestBasename(t *testing.T) {
	testCases := []struct {
		path string
		want string
	}{{
		path: "file.txt",
		want: "file.txt",
	}, {
		path: "/path/to/file.txt",
		want: "file.txt",
	}, {
		path: "/path/to/subdirectory/file.txt",
		want: "file.txt",
	}, {
		path: "/path/to/directory/",
		want: "directory",
	}}

	for _, tc := range testCases {
		result := run(t, "basename", tc.path)
		if result != tc.want {
			t.Errorf("%s = '%s', want %s", tc.path, result, tc.want)
		}
	}
}

func TestBasenameAllNames(t *testing.T) {
	testCases := []struct {
		argv []string
		want []string
	}{{
		argv: []string{"basename", "-a", "/path/to/file.txt"},
		want: []string{"file.txt"},
	}, {
		argv: []string{"basename", "-a", "/path/to/file.txt", "/path/to/file2.txt"},
		want: []string{"file.txt", "file2.txt"},
	}, {
		argv: []string{"basename", "-a", "/path/to/file.txt", "/path/to/file2.txt",
			"/path/to/file3.txt"},
		want: []string{"file.txt", "file2.txt", "file3.txt"},
	}}

	for _, tc := range testCases {
		result := runN(t, tc.argv...)
		if !reflect.DeepEqual(result, tc.want) {
			t.Errorf("%s = '%s', want '%s'", tc.argv, result, tc.want)
		}
	}
}

func TestBasenameSuffixes(t *testing.T) {
	testCases := []struct {
		argv []string
		want []string
	}{{
		argv: []string{"basename", "/path/to/file.txt", ".txt"},
		want: []string{"file"},
	}, {
		argv: []string{"basename", "-s.txt", "/path/to/file.txt", "/path/to/file2.txt"},
		want: []string{"file", "file2"},
	}, {
		argv: []string{"basename", "-s", ".txt", "/path/to/file.txt", "/path/to/file2.txt"},
		want: []string{"file", "file2"},
	}, {
		argv: []string{"basename", "-s", ".txt", "/path/to/file.txt", "/path/to/file2.txt",
			"/path/to/file3.txt"},
		want: []string{"file", "file2", "file3"},
	}}

	for _, tc := range testCases {
		result := runN(t, tc.argv...)
		if !reflect.DeepEqual(result, tc.want) {
			t.Errorf("%s = '%s', want '%s'", tc.argv, result, tc.want)
		}
	}
}

func TestBasenameInvalidInput(t *testing.T) {
	testCases := []struct {
		argv []string
		msg  string
	}{{
		argv: []string{"basename", "foo", "bar", "baz"},
		msg:  "too many arguments",
	}, {
		argv: []string{"basename", "-s"},
		msg:  "bad flag: need value: -s",
	}, {
		argv: []string{"basename", "-s", ".txt"},
		msg:  "needs 1 argument",
	}}

	for _, tc := range testCases {
		result := failN(t, tc.argv...)
		if !strings.Contains(result[0], tc.msg) {
			t.Errorf("%s = '%s', want '%s'", tc.argv, result[0], tc.msg)
		}
	}
}
