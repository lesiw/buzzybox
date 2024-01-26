package flag_test

import (
	"strings"
	"testing"

	"lesiw.io/buzzybox/internal/flag"
)

type config struct {
	a []string
	s string
	x bool
	y bool
	z bool
}

func TestFlag(t *testing.T) {
	tests := []struct {
		args []string
		want config
	}{{
		args: []string{"-x"},
		want: config{[]string{}, "", true, false, false},
	}, {
		args: []string{"--zee"},
		want: config{[]string{}, "", false, false, true},
	}, {
		args: []string{"--zee=true"},
		want: config{[]string{}, "", false, false, true},
	}, {
		args: []string{"--zee=false"},
		want: config{[]string{}, "", false, false, false},
	}, {
		args: []string{"--zee", "false"},
		want: config{[]string{"false"}, "", false, false, true},
	}, {
		args: []string{"-x", "-y"},
		want: config{[]string{}, "", true, true, false},
	}, {
		args: []string{"-xy"},
		want: config{[]string{}, "", true, true, false},
	}, {
		args: []string{"-xs", "foo"},
		want: config{[]string{}, "foo", true, false, false},
	}, {
		args: []string{"-xsfoo"},
		want: config{[]string{}, "foo", true, false, false},
	}, {
		args: []string{"-s", "foo", "bar"},
		want: config{[]string{"bar"}, "foo", false, false, false},
	}, {
		args: []string{"--zee", "foo", "-yxsbar", "baz"},
		want: config{[]string{"foo", "baz"}, "bar", true, true, true},
	}, {
		args: []string{"-x", "--", "-y"},
		want: config{[]string{"-y"}, "", true, false, false},
	}}
	for _, tt := range tests {
		fs := flag.NewFlagSet(new(strings.Builder), "test")
		var flag, x, y, z bool
		var s string
		fs.BoolVar(&x, "x", "")
		fs.BoolVar(&y, "y", "")
		fs.BoolVar(&z, "zee", "")
		fs.BoolVar(&flag, "flag", "")
		fs.StringVar(&s, "s", "")
		if err := fs.Parse(tt.args...); err != nil {
			t.Error(err)
		}
		if x != tt.want.x {
			t.Errorf("x: got %v, want %v", x, tt.want.x)
		}
		if y != tt.want.y {
			t.Errorf("y: got %v, want %v", y, tt.want.y)
		}
		if z != tt.want.z {
			t.Errorf("z: got %v, want %v", z, tt.want.z)
		}
		if s != tt.want.s {
			t.Errorf("s: got %v, want %v", s, tt.want.s)
		}
	}
}
