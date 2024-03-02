package hive_test

import (
	"strconv"
	"strings"
	"testing"

	"lesiw.io/buzzybox/hive"
)

type base64Test struct {
	in  string
	out string
	w   *int
	d   bool
}

func TestBase64(t *testing.T) {
	tests := []base64Test{{
		in:  "hello world\n",
		out: "aGVsbG8gd29ybGQK\n",
	}, {
		in:  "hello world wrapped\n",
		w:   intptr(15),
		out: "aGVsbG8gd29ybGQ\ngd3JhcHBlZAo=\n",
	}, {
		in: "much longer input to test wrapping at 76 columns by default\n",
		out: `bXVjaCBsb25nZXIgaW5wdXQgdG8gdGVzdCB3cmFwcGluZyBhdCA3NiBjb2x1bW5zIGJ5IGRlZmF1
bHQK
`,
	}, {
		in:  "input that ends with padding\n",
		out: "aW5wdXQgdGhhdCBlbmRzIHdpdGggcGFkZGluZwo=\n",
	}, {
		in:  "input without trailing newline",
		out: "aW5wdXQgd2l0aG91dCB0cmFpbGluZyBuZXdsaW5l\n",
	}, {
		in:  "wrapping over multiple lines",
		w:   intptr(5),
		out: "d3Jhc\nHBpbm\ncgb3Z\nlciBt\ndWx0a\nXBsZS\nBsaW5\nlcw==\n",
	}, {
		in:  "ZGVjb2RlIHRlc3Q=",
		out: "decode test",
		d:   true,
	}, {
		in:  "dGVzdCBkZWNvZGUgaWdub3JlcyAtdw==",
		out: "test decode ignores -w",
		d:   true,
		w:   intptr(5),
	}}
	for _, tt := range tests {
		t.Run(strings.Trim(tt.in, "\n"), func(t *testing.T) {
			testBase64Stdin(t, tt)
			testBase64File(t, tt)
		})
	}
}

func testBase64Stdin(t *testing.T, tt base64Test) {
	in := strings.NewReader(tt.in)
	out := &strings.Builder{}
	args := []string{"base64"}
	if tt.d {
		args = append(args, "-d")
	}
	if tt.w != nil {
		args = append(args, "-w", strconv.Itoa(*tt.w))
	}
	cmd := hive.Command(args...)
	cmd.Stdin = in
	cmd.Stdout = out
	if code := cmd.Run(); code != 0 {
		t.Errorf("exit status %v, want 0", code)
	}
	if got := out.String(); got != tt.out {
		t.Errorf("got %q, want %q", got, tt.out)
	}
}

func testBase64File(t *testing.T, tt base64Test) {
	tmpfile := tmpfile(t, tt.in)
	out := &strings.Builder{}
	args := []string{"base64", tmpfile}
	if tt.d {
		args = append(args, "-d")
	}
	if tt.w != nil {
		args = append(args, "-w", strconv.Itoa(*tt.w))
	}
	cmd := hive.Command(args...)
	cmd.Stdout = out
	if code := cmd.Run(); code != 0 {
		t.Errorf("exit status %v, want 0", code)
	}
	if got := out.String(); got != tt.out {
		t.Errorf("got %q, want %q", got, tt.out)
	}
}
