package hive_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lesiw.io/buzzybox/hive"
)

type catTest struct {
	name  string
	stdin []string
	files []string
	args  []string
	want  string
}

func TestCat(t *testing.T) {
	tests := []catTest{{
		name:  "stdin with newline",
		stdin: []string{"hello world\n"},
		want:  "hello world\n",
	}, {
		name:  "stdin without newline",
		stdin: []string{"hello world"},
		want:  "hello world",
	}, {
		name:  "one file",
		files: []string{"foo\n"},
		args:  []string{"f0"},
		want:  "foo\n",
	}, {
		name:  "two files",
		files: []string{"foo\n", "bar\n"},
		args:  []string{"f0", "f1"},
		want:  "foo\nbar\n",
	}, {
		name:  "same file",
		files: []string{"foo\n"},
		args:  []string{"f0", "f0"},
		want:  "foo\nfoo\n",
	}, {
		name:  "file then stdin",
		files: []string{"file contents\n"},
		stdin: []string{"stdin contents\n"},
		args:  []string{"f0", "-"},
		want:  "file contents\nstdin contents\n",
	}, {
		name:  "stdin then file",
		files: []string{"file contents\n"},
		stdin: []string{"stdin contents\n"},
		args:  []string{"-", "f0"},
		want:  "stdin contents\nfile contents\n",
	}, {
		name:  "multiple stdin",
		stdin: []string{"one\n", "two\n"},
		args:  []string{"-", "-"},
		want:  "one\ntwo\n",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCat(t, tt)
		})
	}
}

func testCat(t *testing.T, tt catTest) {
	stdin := newMultiStringReader(tt.stdin)
	stdout := &strings.Builder{}
	if len(tt.files) > 0 {
		dir := t.TempDir()
		for i, f := range tt.files {
			path := filepath.Join(dir, fmt.Sprintf("f%d", i))
			if err := os.WriteFile(path, []byte(f), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
	}
	args := append([]string{"cat"}, tt.args...)
	cmd := hive.Command(args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	if code := cmd.Run(); code != 0 {
		t.Errorf("exit status %d, want 0", code)
	}
	if got := stdout.String(); got != tt.want {
		t.Errorf("got %q, want %q", got, tt.want)
	}
}
