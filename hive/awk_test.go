package hive_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lesiw.io/buzzybox/hive"
	"lesiw.io/buzzybox/internal/posix"
)

type awkTest struct {
	t        *testing.T
	input    string
	name     string
	outfiles []string
}

func (t *awkTest) prog() string {
	return "testdata/awk/" + t.input + "." + t.name + ".awk"
}

func (t *awkTest) inputpath() string {
	return "testdata/awk/" + t.input
}

func (t *awkTest) out() string {
	s, err := os.ReadFile("testdata/awk/" + t.input + "." + t.name + ".out")
	if err != nil {
		return ""
	}
	return string(s)
}

func (t *awkTest) err() string {
	s, err := os.ReadFile("testdata/awk/" + t.input + "." + t.name + ".err")
	if err != nil {
		return ""
	}
	return string(s)
}

func (t *awkTest) run() (stdout string, stderr string, dir string) {
	var err error
	posix.ResetRandom()

	prog := t.prog()
	if len(t.outfiles) > 0 {
		prog, err = filepath.Abs(prog)
		if err != nil {
			t.t.Fatalf("failed to get absolute path: %v", err)
		}
	}
	argv := []string{"awk", "-f", prog}

	input := t.inputpath()
	if len(t.outfiles) > 0 && input != "" {
		input, err = filepath.Abs(input)
		if err != nil {
			t.t.Fatalf("failed to get absolute path: %v", err)
		}
	}
	if input != "" {
		argv = append(argv, input)
	}

	if len(t.outfiles) > 0 {
		pwd, err := os.Getwd()
		if err != nil {
			t.t.Fatal(err)
		}
		dir = t.t.TempDir()
		if err := os.Chdir(dir); err != nil {
			t.t.Fatalf("failed to chdir to temp directory: %s", err)
		}
		defer func() { _ = os.Chdir(pwd) }()
	}

	cmd := hive.Command(argv...)
	cmd.Stdout = new(strings.Builder)
	cmd.Stderr = new(strings.Builder)
	ret := cmd.Run()
	if ret != 0 && t.err() == "" {
		t.t.Errorf("response code: want 0, got %d\nstderr\n---\n%s\n", ret,
			cmd.Stderr.(*strings.Builder).String())
	} else if ret == 0 && t.err() != "" {
		t.t.Errorf("response code: want non-zero, got 0\nstderr\n---\n%s\n",
			cmd.Stderr.(*strings.Builder).String())
	}
	return cmd.Stdout.(*strings.Builder).String(), cmd.Stderr.(*strings.Builder).String(), dir
}

func TestAwk(t *testing.T) {
	files, err := os.ReadDir("testdata/awk")
	if err != nil {
		t.Fatal(err)
	}
	tests := make(map[string]*awkTest)
	exts := stringset("awk", "out", "err")
	for _, file := range files {
		if !strings.Contains(file.Name(), ".") {
			continue
		}
		test := strings.Split(file.Name(), ".")
		if len(test) < 2 {
			t.Fatalf("bad test case: %s", file.Name())
		}
		input := test[0]
		name := test[1]
		ftype := test[2]
		if _, ok := tests[name]; !ok {
			tests[name] = &awkTest{input: input, name: name}
		} else if ftype == "awk" {
			t.Fatal("duplicate test case: " + name)
		}
		if !exts[ftype] {
			tests[name].outfiles = append(tests[name].outfiles, ftype)
		}
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.t = t
			stdout, stderr, outdir := test.run()
			if stderr != test.err() {
				t.Errorf("bad output (stderr)\ngot\n---\n%s\nwant\n----\n%s\n",
					stdout, test.err())
			} else if stdout != test.out() {
				t.Errorf("bad output (stdout)\ngot\n---\n%s\nwant\n----\n%s\n",
					stdout, test.out())
			}
			for _, outfile := range test.outfiles {
				buf, err := os.ReadFile("testdata/awk/" + test.input + "." +
					test.name + "." + outfile + ".out")
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				want := string(buf)
				buf, err = os.ReadFile(outdir + "/" + outfile)
				if err != nil {
					t.Fatalf("could not read output file: %v", err)
				}
				got := string(buf)
				if got != want {
					t.Errorf("bad output (%s)\ngot\n---\n%s\nwant\n----\n%s",
						outfile, got, want)
				}
			}
		})
	}
}

func TestAwkInline(t *testing.T) {
	tests := []struct {
		prog string
	}{
		{prog: "{print}"},
		{prog: "{ print}"},
		{prog: "{print }"},
		{prog: "{ print }"},
		{prog: "{ print; }"},
		{prog: " { print; }"},
		{prog: "  { print; }"},
		{prog: "{ print; } "},
		{prog: "{ print; }  "},
		{prog: " { print; } "},
	}
	for _, tt := range tests {
		t.Run(tt.prog, func(*testing.T) {
			cmd := hive.Command("awk", tt.prog)
			cmd.Stdin = strings.NewReader("hello world")
			cmd.Stdout = new(strings.Builder)
			cmd.Stderr = new(strings.Builder)
			ret := cmd.Run()
			if ret != 0 {
				t.Fatalf("response code: want 0, got %d\nstderr\n---\n%s\n", ret,
					cmd.Stderr.(*strings.Builder).String())
			}
			if cmd.Stdout.(*strings.Builder).String() != "hello world\n" {
				t.Fatalf("output mismatch\n----\n%s\n----\n%s\n",
					cmd.Stdout.(*strings.Builder).String(), "hello world\n")
			}
		})
	}
}

func TestAwkExitCode(t *testing.T) {
	tests := []struct {
		prog string
		code int
	}{
		{prog: "BEGIN { exit }", code: 0},
		{prog: "BEGIN { exit 0 }", code: 0},
		{prog: "BEGIN { exit 2 }", code: 2},
		{prog: "BEGIN { exit 257 }", code: 257},
		{prog: "END { exit }", code: 0},
		{prog: "END { exit 0 }", code: 0},
		{prog: "END { exit 2 }", code: 2},
		{prog: "END { exit 257 }", code: 257},
	}
	for _, tt := range tests {
		t.Run(tt.prog, func(t *testing.T) {
			cmd := hive.Command("awk", tt.prog)
			cmd.Stdin = strings.NewReader("")
			ret := cmd.Run()
			if ret != tt.code {
				t.Errorf("bad exit code: got %d, want %d", ret, tt.code)
			}
		})
	}
}

func TestAwkInlineLoop(t *testing.T) {
	stdout := new(strings.Builder)
	stderr := new(strings.Builder)
	ret := hive.Command("awk", "$3 > 11", "testdata/awk/elements").Run()
	if ret != 0 {
		t.Fatalf("response code: want 0, got %d\nstderr\n---\n%s\n", ret, stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("output mismatch\n----\n%s\n----\n%s\n", stdout.String(), "\n")
	}
}

func TestAwkCliArgs(t *testing.T) {
	tests := []struct {
		in   string
		infs []string
		args []string
		out  string
	}{{
		in:   "hello\ngoodbye\n",
		args: []string{"awk", "BEGIN { x=0; print x; getline; print x, $0 }"},
		out:  "0\n0 hello\n",
	}, {
		in:   "hello\ngoodbye\n",
		args: []string{"awk", "BEGIN { x=0; print x; getline; print x, $0 }", "x=1"},
		out:  "0\n1 hello\n",
	}, {
		in: "hello\ngoodbye\n",
		args: []string{"awk", "BEGIN { x=0; print x; getline; print x, $0 }",
			"x=1", "x=2", "x=3"},
		out: "0\n3 hello\n",
	}, {
		args: []string{"awk", "BEGIN { x=0; print x; getline; print x, $0 }",
			"x=1", "x=2", "x=3", "testdata/awk/hello"},
		out: "0\n3 hello\n",
	}, {
		args: []string{"awk", "BEGIN { getline; print x }", "x=4", "/dev/null"},
		out:  "4\n",
	}, {
		args: []string{"awk", `BEGIN { x=0; getline <"/dev/null"; print x }`,
			"x=5", "/dev/null"},
		out: "0\n",
	}, {
		args: []string{"awk", `BEGIN { x=0; getline <"/dev/null"; print x }`,
			"x=5", "/dev/null"},
		out: "0\n",
	}, {
		args: []string{"awk", `BEGIN { x=0; getline; print x } END { print x }`,
			"x=6", "testdata/awk/hello", "x=end"},
		out: "6\nend\n",
	}, {
		args: []string{"awk", `BEGIN {_=0;getline <"/dev/null";print _} END {print _}`,
			"_=foo", "/dev/null", "_=end"},
		out: "0\nend\n",
	}, {
		args: []string{"awk", "-v", "x=123", `BEGIN { print x }`},
		out:  "123\n",
	}, {
		args: []string{"awk", "-vx=123", `BEGIN { print x }`},
		out:  "123\n",
	}, {
		args: []string{"awk", "-v", "x=123", "-v", "y=abc", "-v", "z1=10.99",
			`BEGIN { print x, y, z1 }`},
		out: "123 abc 10.99\n",
	}, {
		args: []string{"awk", "-vx=123", "-vy=abc", "-vz1=10.99",
			`BEGIN { print x, y, z1 }`},
		out: "123 abc 10.99\n",
	}, {
		args: []string{"awk", "-v", "x=123", "-v", "y=abc", "-v", "z1=10.99", "--",
			`BEGIN { print x, y, z1 }`},
		out: "123 abc 10.99\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 }"},
		args: []string{"awk", "-v", "x=123", "-v", "y=abc", "-f", "f0", "-v", "z1=10.99"},
		out:  "123 abc 10.99\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 }"},
		args: []string{"awk", "-vx=123", "-vy=abc", "-f", "f0", "-vz1=10.99"},
		out:  "123 abc 10.99\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 }"},
		args: []string{"awk", "-f", "f0", "-v", "x=123", "-v", "y=abc", "-v", "z1=10.99"},
		out:  "123 abc 10.99\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 }"},
		args: []string{"awk", "-f", "f0", "-vx=123", "-vy=abc", "-vz1=10.99"},
		out:  "123 abc 10.99\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 } END { print x }"},
		args: []string{"awk", "-f", "f0", "-v", "x=123", "-v", "y=abc", "-v", "z1=10.99",
			"/dev/null", "x=4567", "/dev/null"},
		out: "123 abc 10.99\n4567\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 } END { print x }"},
		args: []string{"awk", "-f", "f0", "-vx=123", "-vy=abc", "-vz1=10.99", "/dev/null",
			"x=4567", "/dev/null"},
		out: "123 abc 10.99\n4567\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 } NR==1 { print x }", "foo\nbar\n"},
		args: []string{"awk", "-v", "x=123", "-v", "y=abc", "-v", "z1=10.99", "-f", "f0",
			"x=4567", "f1"},
		out: "123 abc 10.99\n4567\n",
	}, {
		infs: []string{"BEGIN { print x, y, z1 } NR==1 { print x }", "foo\nbar\n"},
		args: []string{"awk", "-vx=123", "-vy=abc", "-vz1=10.99", "-f", "f0",
			"x=4567", "f1"},
		out: "123 abc 10.99\n4567\n",
	}, {
		infs: []string{"foo\nbar\n"},
		args: []string{"awk", `{ print NR, FNR, FILENAME, $0 }`, "f0", "f0"},
		out:  "1 1 f0 foo\n2 2 f0 bar\n3 1 f0 foo\n4 2 f0 bar\n",
	}, {
		in:   "foo:bar:baz\n",
		args: []string{"awk", "-F:", `{ print $1, $2, $3 }`},
		out:  "foo bar baz\n",
	}, {
		in:   "foo:bar:baz\n",
		args: []string{"awk", "-F", ":", `{ print $1, $2, $3 }`},
		out:  "foo bar baz\n",
	}, {
		in:   "hello\ngoodbye\n",
		args: []string{"awk", `BEGIN { getline x < "-"; print x }`},
		out:  "hello\n",
	}}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			cmd := hive.Command(tt.args...)
			cmd.Stdout = new(strings.Builder)
			cmd.Stderr = new(strings.Builder)
			if tt.in != "" {
				cmd.Stdin = strings.NewReader(tt.in)
			}
			if len(tt.infs) > 0 {
				pwd, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				dir := t.TempDir()
				if err := os.Chdir(dir); err != nil {
					t.Fatalf("failed to chdir to temp directory: %s", err)
				}
				defer func() { _ = os.Chdir(pwd) }()
				for i, f := range tt.infs {
					path := filepath.Join(dir, fmt.Sprintf("f%d", i))
					err := os.WriteFile(path, []byte(f), 0600)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			ret := cmd.Run()
			if ret != 0 {
				t.Fatalf("response code: want 0, got %d\nstderr\n---\n%s\n", ret,
					cmd.Stderr.(*strings.Builder).String())
			}
			if cmd.Stdout.(*strings.Builder).String() != tt.out {
				t.Fatalf("output mismatch\n----\n%s\n----\n%s\n",
					cmd.Stdout.(*strings.Builder).String(), tt.out)
			}
		})
	}

}

func TestAwkEnvVars(t *testing.T) {
	cmd := hive.Command("awk", `BEGIN { print ENVIRON["FOO"] }`)
	cmd.Stdout = new(strings.Builder)
	cmd.Env = []string{"FOO=bar"}
	if cmd.Run() != 0 {
		t.Fatal("command failed")
	}
	out := strings.TrimSuffix(cmd.Stdout.(*strings.Builder).String(), "\n")
	if out != "bar" {
		t.Fatalf("got %s, want %s", out, "bar")
	}
}
