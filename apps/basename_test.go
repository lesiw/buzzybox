package apps_test

import (
	"reflect"
	"strings"
	"testing"

	"lesiw.io/gobox/apps"
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
		result := run(t, apps.Basename, tc.path)
		if result != tc.want {
			t.Errorf("Basename(%s) = '%s', want %s", tc.path, result, tc.want)
		}
	}
}

func TestBasenameAllNames(t *testing.T) {
	testCases := []struct {
		argv []string
		want []string
	}{{
		argv: []string{"-a", "/path/to/file.txt"},
		want: []string{"file.txt"},
	}, {
		argv: []string{"-a", "/path/to/file.txt", "/path/to/file2.txt"},
		want: []string{"file.txt", "file2.txt"},
	}, {
		argv: []string{"-a", "/path/to/file.txt", "/path/to/file2.txt",
			"/path/to/file3.txt"},
		want: []string{"file.txt", "file2.txt", "file3.txt"},
	}}

	for _, tc := range testCases {
		result := runN(t, apps.Basename, tc.argv...)
		if !reflect.DeepEqual(result, tc.want) {
			t.Errorf("Basename(%s) = '%s', want '%s'", tc.argv, result, tc.want)
		}
	}
}

func TestBasenameSuffixes(t *testing.T) {
	testCases := []struct {
		argv []string
		want []string
	}{{
		argv: []string{"/path/to/file.txt", ".txt"},
		want: []string{"file"},
	}, {
		argv: []string{"-s", ".txt", "/path/to/file.txt", "/path/to/file2.txt"},
		want: []string{"file", "file2"},
	}, {
		argv: []string{"-s", ".txt", "/path/to/file.txt", "/path/to/file2.txt",
			"/path/to/file3.txt"},
		want: []string{"file", "file2", "file3"},
	}}

	for _, tc := range testCases {
		result := runN(t, apps.Basename, tc.argv...)
		if !reflect.DeepEqual(result, tc.want) {
			t.Errorf("Basename(%s) = '%s', want '%s'", tc.argv, result, tc.want)
		}
	}
}

func TestBasenameInvalidInput(t *testing.T) {
	testCases := []struct {
		argv []string
		msg  string
	}{{
		argv: []string{"foo", "bar", "baz"},
		msg:  "too many arguments",
	}, {
		argv: []string{"-s"},
		msg:  "flag needs an argument: -s",
	}, {
		argv: []string{"-s", ".txt"},
		msg:  "needs 1 argument",
	}}

	for _, tc := range testCases {
		result := failN(t, apps.Basename, tc.argv...)
		if !strings.Contains(result[0], tc.msg) {
			t.Errorf("Basename(%s) = '%s', want '%s'", tc.argv, result[0], tc.msg)
		}
	}
}
