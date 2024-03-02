package flag

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

var errParse = errors.New("parse error")
var errHelp = errors.New("help requested")

type Value interface {
	String() string
	Set(string) error
}

type boolValue bool

func newBoolValue(p *bool) *boolValue {
	return (*boolValue)(p)
}
func (b *boolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		err = errParse
	}
	*b = boolValue(v)
	return err
}
func (b *boolValue) Get() bool        { return bool(*b) }
func (b *boolValue) String() string   { return strconv.FormatBool(bool(*b)) }
func (b *boolValue) IsBoolFlag() bool { return true }

type stringValue string

func newStringValue(p *string) *stringValue {
	return (*stringValue)(p)
}
func (s *stringValue) Set(val string) error { *s = stringValue(val); return nil }
func (s *stringValue) Get() string          { return string(*s) }
func (s *stringValue) String() string       { return string(*s) }

type intValue int

func newIntValue(p *int) *intValue {
	return (*intValue)(p)
}
func (i *intValue) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, strconv.IntSize)
	if err != nil {
		err = errParse
	}
	*i = intValue(v)
	return err
}
func (i *intValue) Get() int       { return int(*i) }
func (i *intValue) String() string { return strconv.Itoa(int(*i)) }

type Flag struct {
	Name  string
	Usage string
	Value Value
	isSet bool
}

func (f *Flag) set(s string) error {
	f.isSet = true
	return f.Value.Set(s)
}

type FlagSet struct {
	args   []string
	output io.Writer
	name   string
	flags  map[string]*Flag
	Args   []string
	Usage  string
}

func NewFlagSet(output io.Writer, name string) *FlagSet {
	return &FlagSet{output: output, name: name, flags: make(map[string]*Flag)}
}

func (f *FlagSet) Var(value Value, name string, usage string) {
	f.flags[name] = &Flag{name, usage, value, false}
}

func (f *FlagSet) Bool(name string, usage string) *bool {
	var b bool
	f.BoolVar(&b, name, usage)
	return &b
}

func (f *FlagSet) BoolVar(p *bool, name string, usage string) {
	f.Var(newBoolValue(p), name, usage)
}

func (f *FlagSet) String(name string, usage string) *string {
	var s string
	f.StringVar(&s, name, usage)
	return &s
}

func (f *FlagSet) StringVar(p *string, name string, usage string) {
	f.Var(newStringValue(p), name, usage)
}

func (f *FlagSet) Int(name string, usage string) *int {
	var i int
	f.IntVar(&i, name, usage)
	return &i
}

func (f *FlagSet) IntVar(p *int, name string, usage string) {
	f.Var(newIntValue(p), name, usage)
}

func (f *FlagSet) Parse(args ...string) (err error) {
	defer func() {
		if err == nil {
			return
		} else if err != errHelp {
			fmt.Fprintln(f.output, err)
		}
		f.PrintUsage()
	}()
	f.args = args
	for len(f.args) > 0 {
		if f.args[0] == "--" {
			f.args = f.args[1:]
			f.Args = append(f.Args, f.args...)
			return
		} else if f.args[0] == "-" {
			f.Args = append(f.Args, "-")
			f.args = f.args[1:]
		} else if f.args[0][0] == '-' {
			if err = f.parseFlag(); err != nil {
				return
			}
		} else {
			f.Args = append(f.Args, f.args[0])
			f.args = f.args[1:]
		}
	}
	return
}

func (f *FlagSet) Set(s string) bool {
	if f.flags[s] != nil {
		return f.flags[s].isSet
	}
	return false
}

func (f *FlagSet) parseFlag() error {
	arg := f.args[0]
	f.args = f.args[1:]
	if arg[0] != '-' {
		return fmt.Errorf("bad flag: %s", arg)
	} else if len(arg) > 2 && arg[:2] == "--" {
		return f.parseLongFlag(arg[2:])
	} else {
		return f.parseShortFlag(arg[1:])
	}
}

func (f *FlagSet) parseLongFlag(arg string) error {
	name, val, _ := strings.Cut(arg, "=")
	if len(name) == 1 {
		return fmt.Errorf("bad flag: --%s", name) // Short flags are invalid.
	}
	if name == "help" {
		return errHelp
	}
	flag, ok := f.flags[name]
	if !ok {
		return fmt.Errorf("bad flag: --%s", name)
	}
	_, bool := flag.Value.(*boolValue)
	if val == "" && len(f.args) > 0 && !bool {
		val = f.args[0]
		f.args = f.args[1:]
	} else if val == "" && bool {
		val = "true"
	} else if val == "" && !bool {
		return fmt.Errorf("bad flag: needs value: --%s", flag.Name)
	}
	return flag.set(val)
}

func (f *FlagSet) parseShortFlag(arg string) error {
	for len(arg) > 0 {
		name := arg[0]
		arg = arg[1:]
		flag, ok := f.flags[string(name)]
		if !ok {
			return fmt.Errorf("bad flag: -%s", string(name))
		} else if _, bool := flag.Value.(*boolValue); bool {
			if err := flag.set("true"); err != nil {
				return err
			}
		} else if len(arg) > 0 {
			return flag.set(arg)
		} else if len(f.args) > 0 {
			arg = f.args[0]
			f.args = f.args[1:]
			return flag.set(arg)
		} else {
			return fmt.Errorf("bad flag: need value: -%s", flag.Name)
		}
	}
	return nil
}

func (f *FlagSet) PrintError(s string) {
	fmt.Fprintln(f.output, s)
	f.PrintUsage()
}

func (f *FlagSet) PrintUsage() {
	fmt.Fprintln(f.output, f.Usage)
	defaults := f.Defaults()
	if defaults != "" {
		fmt.Fprintln(f.output)
		fmt.Fprintln(f.output, defaults)
	}
}

func (f *FlagSet) Defaults() string {
	var b strings.Builder
	f.Visit(func(flag *Flag) {
		fmt.Fprintf(&b, "  -%s", flag.Name)
		name, usage := UnquoteUsage(flag)
		if len(name) > 0 {
			fmt.Fprintf(&b, " %s", name)
		}
		if b.Len() <= 4 {
			b.WriteString("\t")
		} else {
			b.WriteString("\n    \t")
		}
		b.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))
		b.WriteString("\n")
	})
	return strings.TrimSuffix(b.String(), "\n")
}

func (f *FlagSet) Visit(fn func(*Flag)) {
	for _, flag := range sortFlags(f.flags) {
		fn(flag)
	}
}

func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.Args) {
		return ""
	}
	return f.Args[i]
}

func sortFlags(flags map[string]*Flag) []*Flag {
	result := make([]*Flag, len(flags))
	i := 0
	for _, f := range flags {
		result[i] = f
		i++
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func UnquoteUsage(flag *Flag) (name string, usage string) {
	usage = flag.Usage
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name = usage[i+1 : j]
					usage = usage[:i] + name + usage[j+1:]
					return name, usage
				}
			}
			break
		}
	}
	name = "value"
	switch flag.Value.(type) {
	case *boolValue:
		name = ""
	case *intValue:
		name = "num"
	case *stringValue:
		name = "string"
	}
	return
}
