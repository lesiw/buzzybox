package hive

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lesiw.io/buzzybox/internal/flag"
	"lesiw.io/buzzybox/internal/posix"
)

const awkUsage = `usage: awk [-v VAR=VAL...] [-F SEP] [-f PROGRAM_FILE | PROGRAM] [FILE...]

A pattern scanning and processing language.`

func init() {
	Bees["awk"] = Awk
}

func Awk(cmd *Cmd) (code int) {
	var (
		err       error
		prog      string
		flags     = flag.NewFlagSet(cmd.Stderr, "awk")
		sep       = flags.String("F", "Field separator")
		progfiles = &stringlist{}
		vars      = &stringlist{}
	)
	flags.Var(progfiles, "f", "Path to awk program")
	flags.Var(vars, "v", "Set variable")
	flags.Usage = awkUsage
	if err = flags.Parse(cmd.Args[1:]...); err != nil {
		return 1
	}
	p := newawkp(cmd)
	if *sep != "" {
		p.sym("FS").SetString(*sep)
	}
	for _, f := range *progfiles {
		txt, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(cmd.Stderr, "bad file: %s\n", f)
			return 1
		}
		prog += string(txt)
	}
	if prog == "" && len(flags.Args) > 0 {
		prog = flags.Args[0]
		flags.Args = flags.Args[1:]
	}
	p.sym("ARGC").SetNum(float64(len(flags.Args) + 1))
	p.sym("ARGV").SetKey("0", p.string(cmd.Args[0]))
	for i, a := range flags.Args {
		p.sym("ARGV").SetKey(strconv.Itoa(i+1), p.string(a))
	}
	for _, v := range *vars {
		varval := strings.SplitN(v, "=", 2)
		if len(varval) != 2 {
			fmt.Fprintf(cmd.Stderr, "bad variable, want VAR=VAL: %s\n", v)
			return 1
		}
		var val string
		if val, err = p.unescape(varval[1]); err != nil {
			fmt.Fprintf(cmd.Stderr, "bad escape: %s\n", v)
		}
		p.sym(varval[0]).SetString(val)
	}
	if prog == "" {
		fmt.Fprintln(cmd.Stderr, awkUsage)
		flags.PrintDefaults()
		return 0
	}
	if p.tokens, err = p.lexer.lex(prog); err != nil {
		prettyPrintError(cmd.Stderr, err)
		return 1
	}
	if err := p.findblocks(); err != nil {
		prettyPrintError(cmd.Stderr, err)
		return 1
	}
	if code, err = p.exec(); err != nil {
		prettyPrintError(cmd.Stderr, err)
		return 1
	}
	return
}

type awkp struct {
	cmd *Cmd

	lexer  *lexer
	tokens []*token
	pos    int

	filereader io.RuneScanner
	argvoffset int
	readfile   bool

	readers map[string]runeScanCloser
	writers map[string]io.WriteCloser

	erefn        strset
	stopstmt     strset
	stopexpr     strset
	stopexprlist strset
	stopprint    strset
	endstmt      strset

	stmts map[string]awkeval
	exprs []awkeval

	fntok  []*token
	frames []*awkframe

	begins   []*token
	ends     []*token
	items    []*awkitem
	symbols  map[string]*awkcell
	fields   []*awkcell
	builtins map[string]awkbuiltin
}

type awkfn struct {
	params []*token
	block  *token
}

type awkframe struct {
	symbols map[string]*awkcell
}

type awkitem struct {
	token *token
	in    bool
}

type awkeval func(bool, strset) (*awkcell, error)
type awkbuiltin func([]*awkcell) (*awkcell, error)

func newawkp(cmd *Cmd) *awkp {
	p := &awkp{
		cmd:          cmd,
		symbols:      make(map[string]*awkcell),
		erefn:        stringset("gsub", "match", "split", "sub"),
		stopstmt:     stringset(";", "\n"),
		stopexpr:     stringset("}", ";", ",", "\n", ")"),
		stopexprlist: stringset("{", "}", ";", "\n", ")"),
		stopprint:    stringset("}", ";", ",", "\n", ">", ">>", "|"),
		endstmt:      stringset("", "{", "}", "\n", ";", "(", ")"),
		readers:      make(map[string]runeScanCloser),
		writers:      make(map[string]io.WriteCloser),
	}
	p.builtins = map[string]awkbuiltin{
		"atan2":   p.atan2fn,
		"close":   p.closefn,
		"cos":     p.cosfn,
		"exp":     p.expfn,
		"gsub":    p.gsubfn,
		"int":     p.intfn,
		"length":  p.lengthfn,
		"index":   p.indexfn,
		"log":     p.logfn,
		"match":   p.matchfn,
		"rand":    p.randfn,
		"sin":     p.sinfn,
		"split":   p.splitfn,
		"sprintf": p.sprintffn,
		"sqrt":    p.sqrtfn,
		"srand":   p.srandfn,
		"sub":     p.subfn,
		"substr":  p.substrfn,
		"tolower": p.tolowerfn,
		"toupper": p.toupperfn,
	}
	p.stmts = map[string]awkeval{
		"break":    p.jumpstmt,
		"continue": p.jumpstmt,
		"do":       p.dostmt,
		"delete":   p.deletestmt,
		"exit":     p.exitstmt,
		"for":      p.forstmt,
		"if":       p.ifstmt,
		"next":     p.jumpstmt,
		"nextfile": p.jumpstmt,
		"print":    p.printstmt,
		"printf":   p.printfstmt,
		"return":   p.returnstmt,
		"while":    p.whilestmt,
	}
	exprFns := []func(awkeval, bool, strset) (val *awkcell, err error){
		p.assign, p.cond, p.or, p.and, p.inarray, p.ere, p.cmp, p.concat, p.add,
		p.multiply, p.unary, p.exp, p.prefixop, p.postfixop, p.fieldref, p.group, p.val,
	}
	p.exprs = make([]awkeval, len(exprFns)+1)
	for i := len(exprFns) - 1; i >= 0; i-- {
		p.exprs[i] = func(i int) awkeval {
			return func(exec bool, stop strset) (*awkcell, error) {
				return exprFns[i](p.exprs[i+1], exec, stop)
			}
		}(i)
	}
	// '/' is ambiguous (division vs. start of regex); lex it based on the previous token.
	ere := fnPat("ere", func(l *lexer) *token {
		switch l.tpeek(0).kind {
		case ")", "name", "number", "string":
			return nil
		default:
			return dlPat("ere", '/').Match(l)
		}
	})
	patterns := []matcher{
		dlPat("string", '"'), ere, stPat("begin", "BEGIN"), stPat("end", "END"),
		stPat("break"), stPat("continue"), stPat("delete"), stPat("do"), stPat("else"),
		stPat("exit"), stPat("for"), stPat("function"), stPat("if"), stPat("in"),
		stPat("next"), stPat("nextfile"), stPat("printf"), stPat("print"), stPat("return"),
		stPat("while"), stPat("getline"), stPat("+="), stPat("-="), stPat("*="),
		stPat("/="), stPat("%="), stPat("^="), stPat("**="), stPat("||"), stPat("&&"),
		stPat("=="), stPat("<="), stPat(">="), stPat("!="), stPat("++"), stPat("--"),
		stPat(">>"), stPat("{"), stPat("}"), stPat("("), stPat(")"), stPat("["),
		stPat("]"), stPat(","), stPat(";"), stPat("\n"), stPat("+"), stPat("-"),
		stPat("*"), stPat(`/`), stPat("%"), stPat("^"), stPat("**"), stPat("!"),
		stPat(">"), stPat("<"), stPat("|"), stPat("?"), stPat(":"), stPat("~"), stPat("$"),
		stPat("="), stPat("builtin_func", "atan2", "cos", "sin", "exp", "log", "sqrt",
			"int", "rand", "srand", "gsub", "index", "length", "match", "split",
			"sprintf", "sub", "substr", "tolower", "toupper", "close", "system"),
		rePat("func_name", regexp.MustCompile(`(^[a-zA-Z_][a-zA-Z0-9_]*)\(`)),
		rePat("name", regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*")),
		rePat("number", regexp.MustCompile(`^[0-9]*(?:\.[0-9]+)?(?:[Ee]-?[0-9]+)?`)),
	}
	p.lexer = &lexer{
		patterns: patterns,
		comment:  regexp.MustCompile(`^(?m)#.*$`),
	}
	p.sym("CONVFMT").SetString("%.6g")
	p.sym("FS").SetString(" ")
	p.sym("OFMT").SetString("%.6g")
	p.sym("OFS").SetString(" ")
	p.sym("ORS").SetString("\n")
	p.sym("RS").SetString("\n")
	p.sym("SUBSEP").SetString("\034")
	p.sym("NF").assignhook = func() error {
		nf := int(p.sym("NF").Num())
		if nf >= len(p.fields) {
			p.SetField(nf, p.Field(nf))
		} else if nf < len(p.fields) {
			p.fields = p.fields[:nf+1]
		}
		return p.ftor()
	}
	for _, kv := range p.cmd.Environ() {
		k, v, _ := strings.Cut(kv, "=")
		p.sym("ENVIRON").Key(k).SetString(v)
	}
	return p
}

func (p *awkp) findblocks() error {
	for depth := 0; ; depth = 0 {
		switch p.next().kind {
		case "\n":
			continue
		case "{": // An item with no matching expression.
			p.items = append(p.items, &awkitem{p.peek(-1), false})
			depth++
		case "begin", "end":
			if err := p.mustmatch("{"); err != nil {
				return err
			}
			if p.peek(-2).kind == "end" {
				p.ends = append(p.ends, p.peek(0))
			} else {
				p.begins = append(p.begins, p.peek(0))
			}
			depth++
		case "function":
			funcname := p.peek(0)
			if funcname.kind != "func_name" && funcname.kind != "name" {
				return p.lexer.newTokenErrorf(funcname, "bad function name")
			}
			p.next()
			params, err := p.toklistp()
			if err != nil {
				return err
			}
			if err := p.mustmatch("{"); err != nil {
				return err
			}
			p.sym(funcname.name).SetFn(&awkfn{params, p.next()})
			depth++
		case "":
			return nil
		default:
			p.items = append(p.items, &awkitem{p.peek(-1), false})
		}
		if ok := p.skipblock(depth); !ok {
			return p.lexer.newTokenError(p.peek(0))
		}
	}
}

func (p *awkp) skipblock(depth int) bool {
	for {
		switch p.next().kind {
		case "{":
			depth++
		case "}":
			depth--
			if depth <= 0 {
				return depth == 0
			}
		case "":
			return depth == 0
		case "\n": // An item with no block.
			if depth == 0 {
				return true
			}
		}
	}
}

func (p *awkp) toklistp() (vals []*token, err error) {
	if err = p.mustmatch("("); err != nil {
		return
	}
	for {
		switch p.next().kind {
		case "name":
			if p.hastoken(vals, p.peek(-1)) {
				return nil, p.lexer.newTokenErrorf(p.peek(-1), "bad parameter")
			}
			vals = append(vals, p.peek(-1))
			if !p.match(",") && p.peek(0).name != ")" {
				err = p.lexer.newTokenError(p.peek(0))
				return
			}
		case ")":
			return
		default:
			return nil, p.lexer.newTokenError(p.peek(-1))
		}
	}
}

func (p *awkp) hastoken(toklist []*token, tok *token) bool {
	for _, t := range toklist {
		if t.name == tok.name {
			return true
		}
	}
	return false
}

func (p *awkp) exec() (code int, err error) {
	var val *awkcell
	defer func() {
		if r := recover(); r != nil {
			prettyPrintError(p.cmd.Stderr, p.lexer.newTokenErrorf(p.peek(0), "panic"))
			panic(r)
		}
	}()
	defer func() {
		var terr *tokenError
		if errors.As(err, &terr) && terr.isJump("exit") {
			code = int(val.Num())
		} else if err != nil {
			code = 1
			return
		}
		code, err = p.exit(code)
	}()
	for _, begin := range p.begins {
		p.pos = begin.pos
		if val, err = p.evalblock(true); err != nil {
			return
		}
	}
	if len(p.items) > 0 || len(p.ends) > 0 {
		val, err = p.recordloop()
	}
	return
}

func (p *awkp) recordloop() (val *awkcell, err error) {
	for {
		val, err = p.getline(nil, p.Field(0))
		if val.Num() == 0 && p.argvoffset >= int(p.sym("ARGC").Num())-1 {
			break // EOF and no more files to process.
		} else if val.Num() == 0 {
			continue // EOF but still more files to process.
		} else if err != nil {
			return
		}
		if val, err = p.itemloop(); err != nil {
			return
		}
	}
	return
}

func (p *awkp) itemloop() (val *awkcell, err error) {
	var skip bool
	for _, item := range p.items {
		p.pos = item.token.pos
		if skip, err = p.itemskip(item); err != nil {
			return
		} else if skip {
			continue
		}
		val, err = p.itemblock()
		var terr *tokenError
		switch {
		case err == nil:
			break
		case errors.As(err, &terr) && terr.isJump("next"):
			return nil, nil
		case errors.As(err, &terr) && terr.isJump("nextfile"):
			if err = p.nextreader(); err != nil {
				return
			}
		default:
			return
		}
	}
	return
}

func (p *awkp) itemskip(i *awkitem) (skip bool, err error) {
	var vals []*awkcell
	if vals, err = p.exprlist(true, p.stopexprlist); err != nil {
		return
	}
	switch len(vals) {
	case 0:
		// Implicit match.
	case 1:
		if !vals[0].Bool() {
			skip = true
		}
	case 2:
		if !i.in && vals[0].Bool() {
			i.in = true
		}
		if i.in && vals[1].Bool() {
			i.in = false
		} else if !i.in {
			skip = true
		}
	default:
		err = p.lexer.newTokenErrorf(i.token, "bad exprlist: want 0-2, got %d", len(vals))
	}
	return
}

func (p *awkp) itemblock() (val *awkcell, err error) {
	if p.match("{") {
		val, err = p.evalblock(true)
	} else {
		// Implicit "{ print }".
		fmt.Fprint(p.cmd.Stdout, p.Field(0).String())
		fmt.Fprint(p.cmd.Stdout, p.sym("ORS").String())
	}
	return
}

func (p *awkp) getline(reader io.RuneScanner, set *awkcell) (val *awkcell, err error) {
	val = p.num(-1)
	if reader == nil {
		reader, err = p.reader()
		if err != nil {
			return
		} else if reader == nil {
			return p.num(0), nil
		}
	}
	if err = p.skiptorecord(reader); err != nil {
		return
	}
	var eof bool
	var record string
	if record, err = p.readrecord(reader); err == io.EOF {
		eof = true
	} else if err != nil {
		return
	}
	if err = set.AssignString(record); err != nil {
		return
	}
	eof = len(record) == 0 && eof
	if reader == p.filereader {
		if eof {
			p.filereader = nil
		} else {
			p.sym("NR").SetNum(p.sym("NR").Num() + 1)
			p.sym("FNR").SetNum(p.sym("FNR").Num() + 1)
		}
	}
	if eof {
		return p.num(0), nil
	}
	return p.num(1), nil
}

func (p *awkp) reader() (io.RuneScanner, error) {
	if p.filereader == nil {
		if err := p.nextreader(); err != nil {
			return nil, err
		}
	}
	return p.filereader, nil
}

func (p *awkp) nextreader() (err error) {
	for {
		p.argvoffset++
		arg := "-"
		if p.argvoffset < int(p.sym("ARGC").Num()) {
			arg = p.sym("ARGV").Key(strconv.Itoa(p.argvoffset)).String()
		} else if p.readfile {
			p.filereader = nil
			return nil
		}
		if name, val, ok := strings.Cut(arg, "="); ok {
			if val, err = p.unescape(val); err != nil {
				return err
			}
			p.sym(name).SetString(val)
			continue
		}
		if arg == "-" {
			p.filereader = bufio.NewReader(p.cmd.Stdin)
		} else {
			// TODO: replace with hive.FS
			file, err := os.Open(arg)
			if err != nil {
				return fmt.Errorf("bad file '%s': %s", arg, err)
			}
			p.filereader = bufio.NewReader(file)
			p.readfile = true
		}
		p.sym("FILENAME").SetString(arg)
		p.sym("FNR").SetNum(0)
		return nil
	}
}

func (p *awkp) skiptorecord(reader io.RuneScanner) error {
	if p.sym("RS").String() != "" {
		return nil
	} else if err := skiprune(reader, '\n'); err == io.EOF {
		return nil
	} else {
		return err
	}
}

func (p *awkp) readrecord(reader io.RuneScanner) (string, error) {
	var record strings.Builder
	var r, pr, sr rune
	var err error
	if p.sym("RS").String() != "" {
		sr = []rune(p.sym("RS").String())[0]
	}
	for ; ; pr = r {
		r, _, err = reader.ReadRune()
		if err != nil {
			return record.String(), err
		}
		if sr == 0 {
			if pr == '\n' && r != '\n' {
				record.WriteRune(pr)
			} else if pr == '\n' && r == '\n' {
				return record.String(), nil
			} else if r == '\n' {
				continue
			}
		} else if sr == r {
			return record.String(), nil
		}
		record.WriteRune(r)
	}
}

func (p *awkp) exit(c int) (code int, err error) {
	code = c
	var val *awkcell
loop:
	for _, end := range p.ends {
		p.pos = end.pos
		val, err = p.evalblock(true)
		var terr *tokenError
		switch {
		case err == nil:
			break
		case errors.As(err, &terr) && terr.isJump("exit"):
			code = int(val.Num())
			err = nil
			break loop
		default:
			return
		}
	}
	for _, w := range p.writers {
		_ = w.Close()
	}
	return
}

func (p *awkp) evalblock(exec bool) (val *awkcell, err error) {
	for {
		if p.match("}") {
			return
		} else if p.matchnewlines() {
			continue
		} else if val, err = p.evalstmt(exec, p.stopstmt); err != nil {
			return
		}
	}
}

func (p *awkp) evalstmt(exec bool, stop strset) (val *awkcell, err error) {
	for p.match("\n") {
	}
	if p.match("") {
		err = p.lexer.newTokenError(p.peek(-1))
	} else if p.match("{") {
		val, err = p.evalblock(exec)
	} else if _, ok := p.stmts[p.peek(0).kind]; ok {
		val, err = p.stmts[p.next().kind](exec, stop)
	} else if p.stopstmt[p.peek(0).kind] {
		// Empty statement.
	} else {
		val, err = p.expr(exec, p.stopexpr)
	}
	if err != nil {
		return
	}
	if !p.matchstmtdelim() {
		err = p.lexer.newTokenError(p.peek(0))
	}
	return
}

func (p *awkp) jumpstmt(exec bool, _ strset) (val *awkcell, err error) {
	if !exec {
		return
	}
	return nil, p.lexer.newJumpError(p.peek(-1))
}

func (p *awkp) deletestmt(exec bool, _ strset) (val *awkcell, err error) {
	if err = p.mustmatch("name"); err != nil {
		return
	}
	arr := p.peek(-1).name
	if err = p.mustmatch("["); err != nil {
		return
	}
	var vals []*awkcell
	vals, err = p.exprlist(exec, p.stopexpr)
	if err != nil {
		return
	}
	if err = p.mustmatch("]"); err != nil {
		return
	}
	if !exec {
		return
	}
	p.sym(arr).DelKey(p.join(vals, p.sym("SUBSEP").String()))
	return
}

func (p *awkp) dostmt(exec bool, stop strset) (val *awkcell, err error) {
	start := p.pos
	var whileval *awkcell
	for {
		p.pos = start
		val, err = p.evalstmt(exec, stop)
		var terr *tokenError
		switch {
		case err == nil:
			break
		case errors.As(err, &terr) && terr.isJump("break"):
			err = nil
			exec = false
			continue
		case errors.As(err, &terr) && terr.isJump("continue"):
			err = nil
			continue
		default:
			return
		}
		if err = p.mustmatch("while"); err != nil {
			return
		}
		if whileval, err = p.exprp(exec, p.stopexpr); err != nil {
			return
		}
		if !exec || !whileval.Bool() {
			break
		}
	}
	return
}

func (p *awkp) exitstmt(exec bool, stop strset) (val *awkcell, err error) {
	if exec {
		err = p.lexer.newJumpError(p.peek(-1))
	}
	val, _ = p.expr(exec, stop)
	if exec && val == nil {
		val = p.num(0)
	}
	return
}

func (p *awkp) printstmt(exec bool, _ strset) (val *awkcell, err error) {
	var args []*awkcell
	args, err = p.exprlistoptp(exec, p.stopprint)
	if err != nil {
		return
	}
	if len(args) == 0 {
		args = []*awkcell{p.Field(0)}
	}
	var s strings.Builder
	if exec {
		for i, v := range args {
			if i > 0 {
				s.WriteString(p.sym("OFS").String())
			}
			s.WriteString(v.OutputString())
		}
		s.WriteString(p.sym("ORS").String())
	}
	if err = p.print(exec, s.String()); err != nil {
		return
	}
	return
}

func (p *awkp) printfstmt(exec bool, _ strset) (val *awkcell, err error) {
	var args []*awkcell
	args, err = p.exprlistoptp(exec, p.stopprint)
	if err != nil {
		return
	}
	if exec && len(args) == 0 {
		args = []*awkcell{p.Field(0)}
	}
	var fmtd string
	if exec {
		fmtd, err = p.sprintf(args[0].String(), args[1:])
		if err != nil {
			return
		}
	}
	if err = p.print(exec, fmtd); err != nil {
		return
	}
	return
}

func (p *awkp) print(exec bool, s string) (err error) {
	var w io.Writer = p.cmd.Stdout
	if p.matchany(">", ">>", "|") {
		tok := p.peek(-1)
		op := tok.kind
		var val *awkcell
		if val, err = p.expr(exec, p.stopexpr); err != nil || !exec {
			return
		}
		w = p.writers[val.String()]
		if w == nil {
			if op == ">" || op == ">>" {
				var mode int
				if op == ">" {
					mode = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
				} else if op == ">>" {
					mode = os.O_WRONLY | os.O_CREATE | os.O_APPEND
				}
				// TODO: replace with hive.FS
				if w, err = os.OpenFile(val.String(), mode, 0644); err != nil {
					return p.lexer.newTokenErrorf(tok, "bad file '%s': %s",
						val.String(), err)
				}
			} else {
				cmd := p.cmd.spawn("sh", "-c", val.String())
				if w, err = cmd.StdinCloser(); err != nil {
					return p.lexer.newTokenErrorf(tok, "bad command '%s': %s",
						val.String(), err)
				}
				cmd.Start()
			}
			p.writers[val.String()] = w.(io.WriteCloser)
		}
	}
	if !exec {
		return
	}
	fmt.Fprint(w, s)
	return
}

func (p *awkp) returnstmt(exec bool, stop strset) (val *awkcell, err error) {
	if exec {
		err = p.lexer.newJumpError(p.peek(-1))
	}
	val, _ = p.expr(exec, stop)
	return
}

func (p *awkp) forstmt(exec bool, stop strset) (*awkcell, error) {
	if p.match("(", "name", "in", "name", ")") {
		return p.forstmta(exec, p.sym(p.peek(-4).name), p.sym(p.peek(-2).name))
	} else {
		return p.forstmtc(exec, stop)
	}
}

func (p *awkp) forstmta(exec bool, loopval *awkcell, arrval *awkcell) (val *awkcell, err error) {
	pos := p.pos
loop:
	for i := range arrval.arrval.contents {
		for c := arrval.arrval.contents[i]; c != nil; c = c.next {
			p.pos = pos
			loopval.SetString(c.name)
			val, err = p.evalstmt(exec, p.stopstmt)
			var terr *tokenError
			switch {
			case !exec:
				return
			case err == nil:
				break
			case errors.As(err, &terr) && terr.isJump("break"):
				p.pos = pos
				break loop
			case errors.As(err, &terr) && terr.isJump("continue"):
				err = nil
			default:
				return
			}
		}
	}
	if pos == p.pos {
		val, err = p.evalstmt(false, p.stopstmt)
	}
	return
}

func (p *awkp) forstmtc(exec bool, _ strset) (val *awkcell, err error) {
	init := true
	for pos := p.pos; ; p.pos = pos {
		if exec, err = p.forheader(exec, init); err != nil {
			return
		}
		init = false
		val, err = p.evalstmt(exec, p.stopstmt)
		var terr *tokenError
		switch {
		case !exec:
			return
		case err == nil:
			break
		case errors.As(err, &terr) && terr.isJump("break"):
			p.pos = pos
			if _, err = p.forheader(false, init); err != nil {
				return
			}
			return p.evalstmt(false, p.stopstmt)
		case errors.As(err, &terr) && terr.isJump("continue"):
			err = nil
		default:
			return
		}
		if !exec {
			return
		}
	}
}

func (p *awkp) forheader(exec bool, init bool) (doloop bool, err error) {
	var val *awkcell
	if err = p.mustmatch("("); err != nil {
		return
	}
	if !p.matchfordelim() {
		if _, err = p.evalstmt(exec && init, p.stopexpr); err != nil {
			return
		}
		p.matchnewlines()
	}
	condpos := p.pos
	if !p.matchfordelim() {
		if _, err = p.expr(false, p.stopexpr); err != nil {
			return
		}
		if !p.matchfordelim() {
			return false, p.lexer.newTokenError(p.peek(0))
		}
	}
	if !p.match(")") {
		if _, err = p.evalstmt(exec && !init, p.stopexpr); err != nil {
			return
		}
		p.matchnewlines()
		if err = p.mustmatch(")"); err != nil {
			return
		}
	}
	stmtpos := p.pos
	p.pos = condpos
	if p.matchfordelim() {
		doloop = true
	} else {
		if val, err = p.expr(exec, p.stopexpr); err != nil {
			return
		}
		if exec {
			doloop = val.Bool()
		}
		p.matchnewlines()
	}
	p.pos = stmtpos
	return
}

func (p *awkp) ifstmt(exec bool, stop strset) (val *awkcell, err error) {
	var v, ifval *awkcell
	ifval, err = p.exprp(exec, stop)
	if err != nil {
		return
	}
	for {
		if val, err = p.evalstmt(exec && ifval.Bool(), p.stopstmt); err != nil {
			return
		}
		if exec && ifval.Bool() {
			exec = false
		}
		if !p.match("else") {
			return
		} else if p.match("if") {
			if v, err = p.exprp(exec, p.stopexpr); err != nil {
				return
			}
			if exec {
				ifval = v
			}
		} else {
			return p.evalstmt(exec, p.stopstmt)
		}
	}
}

func (p *awkp) whilestmt(exec bool, stop strset) (val *awkcell, err error) {
	start := p.pos
	var whileval *awkcell
	for {
		p.pos = start
		if whileval, err = p.exprp(exec, p.stopexpr); err != nil {
			return
		}
		val, err = p.evalstmt(exec && whileval.Bool(), stop)
		var terr *tokenError
		switch {
		case err == nil:
			break
		case errors.As(err, &terr) && terr.isJump("break"):
			err = nil
			exec = false
			continue
		case errors.As(err, &terr) && terr.isJump("continue"):
			err = nil
			continue
		default:
			return
		}
		if !exec || !whileval.Bool() {
			break
		}
	}
	return
}

func (p *awkp) expr(exec bool, stop strset) (val *awkcell, err error) {
	if val, err = p.exprs[0](exec, stop); err == nil {
		val, err = p.exprpipe(val, exec, stop)
	}
	return
}

func (p *awkp) exprpipe(in *awkcell, exec bool, stop strset) (val *awkcell, err error) {
	if stop["|"] || !p.match("|") {
		val = in
		return
	}
	tok := p.peek(-1)
	if !p.match("getline") {
		err = p.lexer.newTokenErrorf(tok, "bad pipe")
		return
	}
	r := p.readers[in.String()]
	if exec && r == nil {
		cmd := p.cmd.spawn("sh", "-c", in.String())
		var rc io.ReadCloser
		if rc, err = cmd.StdoutCloser(); err != nil {
			err = p.lexer.newTokenErrorf(tok, "bad command '%s': %s",
				in.String(), err)
			return
		}
		cmd.Start()
		r = newBufferedReadCloser(rc)
		p.readers[in.String()] = r
	}
	set := p.Field(0)
	if p.match("name") {
		set = p.sym(p.peek(-1).name)
	}
	return p.getline(r, set)
}

func (p *awkp) exprp(exec bool, stop strset) (val *awkcell, err error) {
	if err = p.mustmatch("("); err != nil {
		return
	}
	if val, err = p.expr(exec, stop); err != nil {
		return
	}
	err = p.mustmatch(")")
	return
}

func (p *awkp) exprlist(exec bool, stop strset) (vals []*awkcell, err error) {
	var val *awkcell
	for !stop[p.peek(0).kind] {
		if val, err = p.expr(exec, stop); err != nil {
			return
		}
		if val != nil {
			vals = append(vals, val)
		}
		if !p.match(",") {
			return
		}
		p.matchnewlines()
	}
	return
}

func (p *awkp) exprlistp(exec bool, stop strset) (vals []*awkcell, err error) {
	if err = p.mustmatch("("); err != nil {
		return
	}
	if vals, err = p.exprlist(exec, stop); err != nil {
		return
	}
	return vals, p.mustmatch(")")
}

func (p *awkp) exprlistoptp(exec bool, stop strset) (vals []*awkcell, err error) {
	pos := p.pos
	_, err = p.exprlist(false, stop)
	p.pos = pos
	if err == nil {
		return p.exprlist(exec, stop)
	} else {
		return p.exprlistp(exec, p.stopstmt)
	}
}

func (p *awkp) assign(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	if !p.matchany("=", "-=", "+=", "*=", "/=", "%=", "^=", "**=") {
		return
	}
	if exec && !val.assignable {
		err = p.lexer.newTokenErrorf(p.peek(-1), "bad variable")
		return
	}
	op := p.peek(-1)
	var rval *awkcell
	rval, err = p.assign(next, exec, stop) // Right associative.
	if err != nil || !exec {
		return
	}
	switch op.kind {
	case "=":
		val.Set(rval)
	case "-=":
		val.SetNum(val.Num() - rval.Num())
	case "+=":
		val.SetNum(val.Num() + rval.Num())
	case "*=":
		val.SetNum(val.Num() * rval.Num())
	case "/=":
		if rval.Num() == 0 {
			err = p.lexer.newTokenErrorf(op, "bad divisor: 0")
			return
		}
		val.SetNum(val.Num() / rval.Num())
	case "%=":
		val.SetNum(math.Mod(val.Num(), rval.Num()))
	case "^=", "**=":
		val.SetNum(math.Pow(val.Num(), rval.Num()))
	}
	return val, val.AssignHook()
}

func (p *awkp) cond(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	if !p.match("?") {
		return
	}
	var tval, fval *awkcell
	if tval, err = p.expr(exec && val.Bool(), stop); err != nil {
		return
	}
	if err = p.mustmatch(":"); err != nil {
		return
	}
	if fval, err = p.expr(exec && !val.Bool(), stop); err != nil || !exec {
		return
	}
	if val.Bool() {
		return tval, nil
	} else {
		return fval, nil
	}
}

func (p *awkp) or(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	var rval *awkcell
	for p.match("||") {
		if exec && val.Bool() {
			exec = false // Short circuit.
		}
		rval, err = next(exec, stop) // Left associative.
		if err != nil {
			return
		}
		if !exec {
			continue
		}
		val = p.bool(val.Bool() || rval.Bool())
	}
	return
}

func (p *awkp) and(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	var rval *awkcell
	for p.match("&&") {
		if exec && !val.Bool() {
			exec = false // Short circuit.
		}
		rval, err = next(exec, stop) // Left associative.
		if err != nil {
			return
		}
		if !exec {
			continue
		}
		val = p.bool(rval.Bool())
	}
	return
}

func (p *awkp) inarray(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	var arr *awkcell
	var idx string
	pos := p.pos
	if p.match("(", "exprlist", ")", "in") {
		var vals []*awkcell
		p.pos = pos
		if vals, err = p.exprlistp(exec, p.stopexprlist); err != nil {
			return
		}
		if exec {
			idx = p.join(vals, p.sym("SUBSEP").String())
		}
		if err = p.mustmatch("in"); err != nil {
			return
		}
	} else {
		val, err = next(exec, stop) // Left associative.
		if err != nil || !p.match("in") {
			return
		}
		if exec {
			idx = val.String()
		}
	}
	arr, err = next(exec, stop)
	if err != nil || !exec {
		return
	}
	return arr.HasKey(idx), nil
}

func (p *awkp) ere(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	var rval *awkcell
	var m bool
	if val, err = next(exec, stop); err != nil {
		return
	}
	if p.match("~") {
		m = true
	} else if p.match("!", "~") {
		m = false
	} else {
		return
	}
	for {
		if p.peek(0).kind == "ere" {
			rval = p.string(p.next().name)
			rval.regexp = true
		} else if rval, err = next(exec, stop); err != nil {
			return
		}
		if exec {
			var re *regexp.Regexp
			re, err = regexp.CompilePOSIX(rval.String())
			if err != nil {
				return nil, p.lexer.newTokenErrorf(p.peek(0), "bad regex: %s", err)
			}
			r := re.MatchString(val.String())
			if m {
				val = p.bool(r)
			} else {
				val = p.bool(!r)
			}
		}
		if p.match("~") {
			m = true
		} else if p.match("!", "~") {
			m = false
		} else {
			return
		}
	}
}

func (p *awkp) cmp(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	if val, err = next(exec, stop); err != nil {
		return
	}
	var op string
	var rval *awkcell
	for !stop[p.peek(0).kind] && p.matchany("<", "<=", "==", "!=", ">=", ">") {
		op = p.peek(-1).kind
		if rval, err = next(exec, stop); err != nil {
			return
		}
		if !exec {
			continue
		}
		switch op {
		case "<":
			val = p.bool(p.cmpvals(val, rval) < 0)
		case "<=":
			val = p.bool(p.cmpvals(val, rval) <= 0)
		case "==":
			val = p.bool(p.cmpvals(val, rval) == 0)
		case "!=":
			val = p.bool(p.cmpvals(val, rval) != 0)
		case ">=":
			val = p.bool(p.cmpvals(val, rval) >= 0)
		case ">":
			val = p.bool(p.cmpvals(val, rval) > 0)
		}
	}
	return
}

func (p *awkp) cmpvals(lval *awkcell, rval *awkcell) int {
	if lval.IsString() || rval.IsString() {
		return cmp.Compare(lval.String(), rval.String())
	} else {
		return cmp.Compare(lval.Num(), rval.Num())
	}
}

func (p *awkp) concat(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	for {
		pos := p.pos
		_, exprerr := next(false, stop)
		p.pos = pos
		if exprerr != nil {
			return // Invalid expression; nothing to concatenate.
		}
		var rval *awkcell
		rval, err = next(exec, stop) // Left associative, always matches.
		if err != nil {
			return
		}
		if exec {
			val = p.string(val.String() + rval.String())
		}
	}
}

func (p *awkp) add(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	var op string
	var rval *awkcell
	for p.matchany("+", "-") {
		op = p.peek(-1).kind
		rval, err = next(exec, stop)
		if err != nil {
			return
		}
		if !exec {
			continue
		}
		switch op {
		case "+":
			val = p.num(val.Num() + rval.Num())
		case "-":
			val = p.num(val.Num() - rval.Num())
		}
	}
	return
}

func (p *awkp) multiply(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	var op *token
	var rval *awkcell
	for p.matchany("*", "/", "%") {
		op = p.peek(-1)
		rval, err = next(exec, stop)
		if err != nil {
			return
		}
		if !exec {
			continue
		}
		switch op.kind {
		case "*":
			val = p.num(val.Num() * rval.Num())
		case "/":
			if rval.Num() == 0 {
				return nil, p.lexer.newTokenErrorf(op, "bad divisor: 0")
			}
			val = p.num(val.Num() / rval.Num())
		case "%":
			val = p.num(math.Mod(val.Num(), rval.Num()))
		}
	}
	return
}

func (p *awkp) unary(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	if !p.matchany("-", "+", "!") {
		return next(exec, stop)
	}
	op := p.peek(-1).kind
	val, err = p.unary(next, exec, stop) // Right associative.
	if err != nil || !exec {
		return
	}
	switch op {
	case "-":
		val = p.num(-1 * val.Num())
		if val.Num() == -0 {
			val.SetNum(0)
		}
	case "+":
		val = p.num(val.Num())
	case "!":
		val = p.bool(!val.Bool())
	}
	return
}

func (p *awkp) exp(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if err != nil {
		return
	}
	if !p.matchany("^", "**") {
		return
	}
	var rval *awkcell
	rval, err = p.exp(next, exec, stop) // Right associative.
	if err != nil || !exec {
		return
	}
	val = p.num(math.Pow(val.Num(), rval.Num()))
	return
}

func (p *awkp) prefixop(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	if !p.matchany("++", "--") {
		return next(exec, stop)
	}
	sign := p.peek(-1).kind
	val, err = next(exec, stop)
	if !exec {
		return
	}
	if !val.assignable {
		err = p.lexer.newTokenErrorf(p.peek(0), "bad variable")
		return
	}
	switch sign {
	case "++":
		val.SetNum(val.Num() + 1)
	case "--":
		val.SetNum(val.Num() - 1)
	}
	if val.assignhook != nil {
		if err = val.assignhook(); err != nil {
			return
		}
	}
	return
}

func (p *awkp) postfixop(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	val, err = next(exec, stop)
	if !p.matchany("++", "--") {
		return
	}
	if !exec {
		return
	}
	if !val.assignable {
		err = p.lexer.newTokenErrorf(p.peek(0), "bad variable")
		return
	}
	var varval *awkcell
	switch p.peek(-1).kind {
	case "++":
		varval = val
		val = p.num(varval.Num())
		varval.SetNum(val.Num() + 1)
	case "--":
		varval = val
		val = p.num(varval.Num())
		varval.SetNum(val.Num() - 1)
	}
	if varval.assignhook != nil {
		if err = varval.assignhook(); err != nil {
			return
		}
	}
	return
}

func (p *awkp) fieldref(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	if !p.match("$") {
		return next(exec, stop)
	}
	var i *awkcell
	i, err = next(exec, stop)
	if err != nil || !exec {
		return nil, err
	}
	val = p.Field(int(i.Num()))
	return
}

func (p *awkp) group(next awkeval, exec bool, stop strset) (val *awkcell, err error) {
	if !p.match("(") {
		return next(exec, stop)
	}
	val, err = p.expr(exec, p.stopexpr)
	if err != nil {
		return
	}
	err = p.mustmatch(")")
	return
}

func (p *awkp) val(_ awkeval, exec bool, _ strset) (val *awkcell, err error) {
	tok := p.next()
	switch tok.kind {
	case "number":
		return p.numval(tok, exec)
	case "name":
		return p.symval(tok, exec)
	case "string":
		return p.strval(tok, exec)
	case "ere":
		return p.ereval(tok, exec)
	case "builtin_func":
		p.fntok = append(p.fntok, tok)
		defer func() { p.fntok = p.fntok[:len(p.fntok)-1] }()
		return p.builtin(tok.name, exec)
	case "func_name":
		p.fntok = append(p.fntok, tok)
		defer func() { p.fntok = p.fntok[:len(p.fntok)-1] }()
		return p.fn(tok.name, exec)
	case "getline":
		return p.getlinefn(exec)
	default:
		return nil, p.lexer.newTokenError(tok)
	}
}

func (p *awkp) numval(tok *token, exec bool) (*awkcell, error) {
	if !exec {
		return nil, nil
	}
	num, err := strconv.ParseFloat(tok.name, 64)
	if err != nil {
		return nil, p.lexer.newTokenErrorf(tok, "bad number")
	}
	return p.num(num), nil
}

func (p *awkp) symval(tok *token, exec bool) (*awkcell, error) {
	if idx, ok, err := p.symidx(exec); err != nil {
		return nil, err
	} else if ok {
		return p.sym(tok.name).Key(idx), nil
	}
	return p.sym(tok.name), nil
}

func (p *awkp) symidx(exec bool) (string, bool, error) {
	if p.peek(-1).kind != "name" || !p.match("[") {
		return "", false, nil
	}
	vals, err := p.exprlist(exec, p.stopexprlist)
	if err != nil {
		return "", false, err
	}
	if err = p.mustmatch("]"); err != nil {
		return "", false, err
	}
	if !exec {
		return "", false, nil
	}
	return p.join(vals, p.sym("SUBSEP").String()), true, nil
}

func (p *awkp) strval(tok *token, exec bool) (val *awkcell, err error) {
	if !exec {
		return
	}
	if str, err := p.unescape(tok.name); err != nil {
		return nil, p.lexer.newTokenErrorf(tok, err.Error())
	} else {
		val = p.string(str)
	}
	val.string = true
	return
}

func (p *awkp) ereval(tok *token, exec bool) (val *awkcell, err error) {
	if !exec {
		return
	}
	if p.ereimplicit() {
		return p.ererecord(tok.name)
	}
	val = p.string(tok.name)
	val.regexp = true
	return
}

func (p *awkp) ereimplicit() bool {
	// A regex is an implicit match against field 0 if:
	//   - it is not part of a regex match expression; and
	//   - it is not an argument to specific builtin functions (see p.erefn).
	return len(p.fntok) == 0 || !p.erefn[p.fntok[len(p.fntok)-1].name]
}

func (p *awkp) ererecord(s string) (val *awkcell, err error) {
	var re *regexp.Regexp
	re, err = regexp.CompilePOSIX(s)
	if err != nil {
		return nil, p.lexer.newTokenErrorf(p.peek(0), "bad regex: %s", err)
	}
	val = p.bool(re.MatchString(p.Field(0).String()))
	return
}

func (p *awkp) getlinefn(exec bool) (val *awkcell, err error) {
	var r runeScanCloser
	set := p.Field(0)
	if p.match("name") {
		set = p.sym(p.peek(-1).name)
	}
	if p.match("<") {
		if val, err = p.expr(exec, p.stopexpr); err != nil || !exec {
			return
		}
		r = p.readers[val.String()]
		if r == nil {
			var f io.ReadCloser
			if val.String() == "-" {
				f = io.NopCloser(p.cmd.Stdin)
			} else if f, err = os.Open(val.String()); err != nil {
				// TODO: replace with hive.FS
				err = nil
				val = p.num(-1)
				return
			}
			r = newBufferedReadCloser(f)
			p.readers[val.String()] = r
		}
	}
	if !exec {
		return
	}
	return p.getline(r, set)
}

func (p *awkp) builtin(name string, exec bool) (val *awkcell, err error) {
	var args []*awkcell
	if p.match("(") {
		if args, err = p.exprlist(exec, p.stopexprlist); err != nil {
			return
		}
		if err = p.mustmatch(")"); err != nil {
			return
		}
	}
	if !exec {
		return
	}
	fn, ok := p.builtins[name]
	if !ok {
		err = p.lexer.newTokenErrorf(p.fntok[len(p.fntok)-1], "bad function")
		return
	}
	val, err = fn(args)
	if err != nil {
		err = p.lexer.newTokenErrorf(p.fntok[len(p.fntok)-1], err.Error())
	}
	return
}

func (p *awkp) fn(name string, exec bool) (val *awkcell, err error) {
	fn := p.sym(name).fnval
	if fn == nil {
		return nil, p.lexer.newTokenErrorf(p.peek(-1), "bad function: %s", name)
	}
	var args []*awkcell
	if args, err = p.exprlistp(exec, p.stopexprlist); err != nil {
		return
	}
	if !exec {
		return
	}
	val, err = p.call(fn, args)
	var terr *tokenError
	if errors.As(err, &terr) && terr.token.name == "return" {
		err = nil
	}
	return
}

func (p *awkp) call(fn *awkfn, args []*awkcell) (val *awkcell, err error) {
	frame := &awkframe{symbols: make(map[string]*awkcell)}
	p.frames = append(p.frames, frame)
	defer func() { p.frames = p.frames[:len(p.frames)-1] }()
	for i, tok := range fn.params {
		c := &awkcell{prog: p, assignable: true}
		if i < len(args) {
			c.Set(args[i])
		}
		frame.symbols[tok.name] = c
	}
	pos := p.pos
	defer func() { p.pos = pos }()
	p.pos = fn.block.pos
	return p.evalblock(true)
}

func (p *awkp) atan2fn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 2 {
		err = fmt.Errorf("bad argc: want 2, got %d", len(args))
		return
	}
	return p.num(math.Atan2(args[0].Num(), args[1].Num())), nil
}

func (p *awkp) closefn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		err = fmt.Errorf("bad argc: want 1, got %d", len(args))
		return
	}
	w := p.writers[args[0].String()]
	if w == nil {
		return
	}
	if err = w.Close(); err != nil {
		return
	}
	w = nil
	return
}

func (p *awkp) cosfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		err = fmt.Errorf("bad argc: want 1, got %d", len(args))
		return
	}
	return p.num(math.Cos(args[0].Num())), nil
}

func (p *awkp) expfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		err = fmt.Errorf("bad argc: want 1, got %d", len(args))
		return
	}
	return p.num(math.Exp(args[0].Num())), nil
}

func (p *awkp) lengthfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) > 2 {
		return nil, fmt.Errorf("bad argc: want 0-1, got %d", len(args))
	}
	var arg *awkcell
	if len(args) == 0 {
		arg = p.Field(0)
	} else {
		arg = args[0]
	}
	if arg.Arr().count > 0 {
		return p.num(float64(arg.Arr().count)), nil
	} else {
		return p.num(float64(len([]rune(arg.String())))), nil
	}
}

func (p *awkp) indexfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("bad argc: want 2, got %d", len(args))
	}
	s := []rune(args[0].String())
	t := []rune(args[1].String())
	if len(t) < 1 && len(s) > 0 {
		return p.num(1), nil
	}
	for i, r := range s {
		if len(t) < 1 {
			break
		} else if r == t[0] {
			for j, tr := range t {
				if i+j >= len(s) || s[i+j] != tr {
					break
				}
				if j == len(t)-1 {
					return p.num(float64(i + 1)), nil
				}
			}
		}
	}
	return p.num(0), nil
}

func (p *awkp) logfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("bad argc: want 1, got %d", len(args))
	}
	return p.num(math.Log(args[0].Num())), nil
}

func (p *awkp) matchfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("bad argc: want 2, got %d", len(args))
	}
	s := args[0].String()
	pat := args[1].String()
	var re *regexp.Regexp
	if re, err = regexp.CompilePOSIX(pat); err != nil {
		err = fmt.Errorf("bad regex: %s", err)
		return
	}
	idx := re.FindStringIndex(s)
	if idx != nil {
		var start int
		for range s[:idx[0]] {
			start++
		}
		var length int
		for range s[idx[0]:idx[1]] {
			length++
		}
		p.sym("RSTART").SetNum(float64(start + 1))
		p.sym("RLENGTH").SetNum(float64(length))
		val = p.num(float64(start + 1))
	} else {
		p.sym("RSTART").SetNum(0)
		p.sym("RLENGTH").SetNum(-1)
		val = p.num(0)
	}
	return
}

func (p *awkp) gsubfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("bad argc: want 2-3, got %d", len(args))
	}
	pat := args[0].String()
	rpl := args[1].String()
	var in *awkcell
	if len(args) == 3 {
		in = args[2]
	} else {
		in = p.Field(0)
	}
	var re *regexp.Regexp
	if re, err = regexp.CompilePOSIX(pat); err != nil {
		err = fmt.Errorf("bad regex: %s", err)
		return
	}
	var count int
	in.SetString(re.ReplaceAllStringFunc(in.String(), func(s string) string {
		count++
		m := re.FindString(s)
		var r string
		for i, c := range rpl {
			if c == '&' && (i == 0 || rpl[i-1] != '\\') {
				r += m
			} else if !(c == '\\' && i < len(rpl)-1 && rpl[i+1] == '&') {
				r += string(c)
			}
		}
		return re.ReplaceAllString(s, r)
	}))
	val = p.num(float64(count))
	return
}

func (p *awkp) intfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) < 1 || len(args) > 1 {
		return nil, fmt.Errorf("bad argc: want 1, got %d", len(args))
	}
	input := args[0]
	return p.num(float64(int(input.Num()))), nil
}

func (p *awkp) randfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("bad argc: want 0, got %d", len(args))
	}
	return p.num(float64(posix.Random()) / float64(math.MaxInt32)), nil
}

func (p *awkp) sinfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		err = fmt.Errorf("bad argc: want 1, got %d", len(args))
		return
	}
	return p.num(math.Sin(args[0].Num())), nil
}

func (p *awkp) splitfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("bad argc: want 1-3, got %d", len(args))
	}
	s := args[0]
	a := args[1]
	a.Arr().reset()
	var fs *awkcell
	if len(args) == 2 {
		fs = p.sym("FS")
	} else {
		fs = args[2]
	}
	var num int
	num, err = p.split(s, a, fs)
	val = p.num(float64(num))
	return
}

func (p *awkp) sprintffn(args []*awkcell) (val *awkcell, err error) {
	var fmtd string
	fmtd, err = p.sprintf(args[0].String(), args[1:])
	if err != nil {
		return
	}
	val = p.string(fmtd)
	return
}

func (p *awkp) sqrtfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		err = fmt.Errorf("bad argc: want 1, got %d", len(args))
		return
	}
	return p.num(math.Sqrt(args[0].Num())), nil
}

func (p *awkp) srandfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) == 0 {
		posix.Srandom(int(time.Now().Unix()))
	} else if len(args) == 1 {
		posix.Srandom(int(args[0].Num()))
	} else {
		err = fmt.Errorf("bad argc: want 0-1, got %d", len(args))
	}
	return p.num(1), err
}

func (p *awkp) subfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("bad argc: want 2-3, got %d", len(args))
	}
	pat := args[0].String()
	rpl := args[1].String()
	var in *awkcell
	if len(args) == 3 {
		in = args[2]
	} else {
		in = p.Field(0)
	}
	var re *regexp.Regexp
	if re, err = regexp.CompilePOSIX(pat); err != nil {
		err = fmt.Errorf("bad regex: %s", err)
		return
	}
	var count int
	in.SetString(re.ReplaceAllStringFunc(in.String(), func(s string) string {
		if count > 0 {
			return s
		}
		count++
		m := re.FindString(s)
		var r string
		for i, c := range rpl {
			if c == '&' && (i == 0 || rpl[i-1] != '\\') {
				r += m
			} else if !(c == '\\' && i < len(rpl)-1 && rpl[i+1] == '&') {
				r += string(c)
			}
		}
		return re.ReplaceAllString(s, r)
	}))
	val = p.num(float64(count))
	return
}

func (p *awkp) substrfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) > 3 {
		return nil, fmt.Errorf("bad argc: want 2-3, got %d", len(args))
	}
	s := []rune(args[0].String())
	m := int(args[1].Num()) - 1
	n := len(s)
	if m > len(s)-1 {
		return p.string(""), nil
	}
	if m < 0 {
		m = 0
	}
	if len(args) > 2 {
		n = int(args[2].Num())
		if n < 0 {
			n = 0
		}
	}
	if m+n > len(s) {
		val = p.string(string(s[m:]))
	} else {
		val = p.string(string(s[m : m+n]))
	}
	return
}

func (p *awkp) tolowerfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		err = fmt.Errorf("bad argc: want 1, got %d", len(args))
		return
	}
	return p.string(strings.ToLower(args[0].String())), nil
}

func (p *awkp) toupperfn(args []*awkcell) (val *awkcell, err error) {
	if len(args) != 1 {
		err = fmt.Errorf("bad argc: want 1, got %d", len(args))
		return
	}
	return p.string(strings.ToUpper(args[0].String())), nil
}

func (p *awkp) sym(s string) *awkcell {
	if len(p.frames) > 0 {
		tab := p.frames[len(p.frames)-1].symbols
		if val, ok := tab[s]; ok {
			return val
		}
	}
	if val, ok := p.symbols[s]; ok {
		return val
	}
	p.symbols[s] = &awkcell{prog: p, assignable: true}
	return p.symbols[s]
}

func (p *awkp) Field(i int) *awkcell {
	if i < len(p.fields) && p.fields[i] != nil {
		return p.fields[i]
	}
	c := p.string("")
	c.assignable = true
	if i > 0 {
		c.assignhook = func() error { p.SetField(i, c); return p.ftor() }
	} else {
		c.assignhook = func() error { p.SetField(i, c); return p.rtof() }
	}
	return c
}

func (p *awkp) SetField(i int, c *awkcell) {
	if i > len(p.fields)-1 {
		p.fields = append(p.fields, make([]*awkcell, i-len(p.fields)+1)...)
	}
	if p.fields[i] == nil {
		p.fields[i] = p.Field(i)
	}
	if i > int(p.sym("NF").Num()) {
		p.sym("NF").SetNum(float64(i))
	}
	p.fields[i].Set(c)
}

func (p *awkp) rtof() error {
	p.fields = p.fields[:1]
	nf, err := p.split(p.Field(0), p, p.sym("FS"))
	if err != nil {
		return err
	}
	p.sym("NF").SetNum(float64(nf))
	return nil
}

func (p *awkp) ftor() error {
	p.SetField(0, p.string(p.join(p.fields[1:], p.sym("OFS").String())))
	return nil
}

func (p *awkp) peek(n int) *token {
	if p.pos+n < 0 || p.pos+n > len(p.tokens)-1 {
		return &token{}
	}
	return p.tokens[p.pos+n]
}

func (p *awkp) next() *token {
	if p.pos > len(p.tokens)-1 {
		return &token{pos: -1}
	}
	p.pos++
	return p.tokens[p.pos-1]
}

func (p *awkp) match(kind ...string) bool {
	pos := p.pos
	defer func() { p.pos = pos }()
	for _, k := range kind {
		if k == "expr" {
			if _, err := p.expr(false, p.stopexpr); err != nil {
				return false
			}
		} else if k == "exprlist" {
			if _, err := p.exprlist(false, p.stopexprlist); err != nil {
				return false
			}
		} else if p.next().kind != k {
			return false
		}
	}
	pos = p.pos
	return true
}

func (p *awkp) mustmatch(kind ...string) error {
	for _, k := range kind {
		if !p.match(k) {
			return p.lexer.newTokenErrorf(p.peek(0), "want %s, got %s",
				k, p.peek(0).kind)
		}
	}
	return nil
}

func (p *awkp) matchany(kind ...string) bool {
	for _, k := range kind {
		if p.match(k) {
			return true
		}
	}
	return false
}

func (p *awkp) matchstmtdelim() bool {
	if !p.endstmt[p.peek(-1).kind] && !p.endstmt[p.peek(0).kind] {
		return false
	} else if p.stopstmt[p.peek(0).kind] {
		p.next()
	}
	return true
}

func (p *awkp) matchfordelim() bool {
	if !p.match(";") {
		return false
	}
	p.matchnewlines()
	return true
}

func (p *awkp) matchnewlines() (matched bool) {
	for p.match("\n") {
		matched = true
	}
	return
}

func (p *awkp) unescape(s string) (string, error) {
	runes := []rune(s)
	var ret strings.Builder
	for i := 0; i < len(runes); i++ {
		if runes[i] != '\\' || i >= len(runes)-1 {
			ret.WriteRune(runes[i])
			continue
		}
		i++
		switch runes[i] {
		case '\\':
			ret.WriteRune('\\')
		case 'a':
			ret.WriteRune('\a')
		case 'b':
			ret.WriteRune('\b')
		case 'f':
			ret.WriteRune('\f')
		case 'n':
			ret.WriteRune('\n')
		case 'r':
			ret.WriteRune('\r')
		case 't':
			ret.WriteRune('\t')
		case '&':
			ret.WriteRune('&')
		default:
			return "", fmt.Errorf("bad escape: \\%c", runes[i])
		}
	}
	return ret.String(), nil
}

func (p *awkp) sprintf(fmtstr string, a []*awkcell) (string, error) {
	var result strings.Builder
	var arg int
	format := []rune(fmtstr)
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			result.WriteRune(format[i])
			continue
		}
		if format[i+1] == '%' {
			result.WriteRune('%')
			i++
			continue
		}
		if arg >= len(a) {
			return "", fmt.Errorf("bad argc: not enough arguments")
		}
		strt := i
		for {
			i++
			if i >= len(format) {
				return "", fmt.Errorf("bad verb: %s", string(format[strt:i+1]))
			}
			if runealpha(format[i]) {
				break
			}
		}
		verb := string(format[strt : i+1])
		if err := p.sprintfv(&result, verb, a[arg]); err != nil {
			return "", err
		}
		arg++
	}
	return result.String(), nil
}

func (p *awkp) sprintfv(result *strings.Builder, verb string, val *awkcell) error {
	verbsl := []rune(verb)
	switch verbsl[len(verbsl)-1] {
	case 'c':
		if val.IsString() {
			runes := []rune(val.String())
			if len(runes) > 0 {
				result.WriteString(fmt.Sprintf(verb, runes[0]))
			}
		} else {
			result.WriteByte(byte(val.Num()))
		}
	case 's':
		result.WriteString(fmt.Sprintf(verb, val.String()))
	case 'd', 'i':
		result.WriteString(fmt.Sprintf(verb, int(val.Num())))
	case 'o', 'x', 'X':
		result.WriteString(fmt.Sprintf(verb, uint(val.Num())))
	case 'u':
		verbsl[len(verbsl)-1] = 'd'
		result.WriteString(fmt.Sprintf(string(verbsl), uint(val.Num())))
	case 'g', 'G':
		if verb == "%g" {
			verb = "%.6g"
		}
		result.WriteString(fmt.Sprintf(verb, val.Num()))
	case 'a', 'A', 'f', 'e', 'E':
		result.WriteString(fmt.Sprintf(verb, val.Num()))
	default:
		return fmt.Errorf("bad verb: %s", verb)
	}
	return nil
}

func (p *awkp) join(vals []*awkcell, by string) string {
	var s strings.Builder
	for i, v := range vals {
		if v == nil {
			v = &awkcell{}
		}
		if i > 0 {
			s.WriteString(by)
		}
		s.WriteString(v.String())
	}
	return s.String()
}

type fielder interface {
	Field(int) *awkcell
	SetField(int, *awkcell)
}

func (p *awkp) split(s *awkcell, a fielder, fs *awkcell) (count int, err error) {
	if s.String() == "" {
		return
	} else if fs.String() == "" {
		count = p.splitall(s.String(), a)
	} else if fs.String() == " " {
		count = p.splitspace(s.String(), a)
	} else if fs.regexp || len(fs.String()) > 1 {
		if re, err := regexp.CompilePOSIX(fs.String()); err != nil {
			return 0, fmt.Errorf("bad FS regex: %w", err)
		} else {
			count = p.splitregex(re, s.String(), a)
		}
	} else {
		fs := []rune(fs.String())[0]
		count = p.splitrune(fs, s.String(), a)
	}
	return
}

func (p *awkp) splitall(s string, a fielder) (count int) {
	for _, r := range s {
		count++
		a.SetField(count, p.string(string(r)))
	}
	return
}

func (p *awkp) splitspace(s string, a fielder) (count int) {
	var field strings.Builder
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if field.Len() > 0 {
				count++
				a.SetField(count, p.string(field.String()))
				field.Reset()
			}
			continue
		}
		field.WriteRune(r)
	}
	if field.Len() > 0 {
		count++
		a.SetField(count, p.string(field.String()))
	}
	return
}

func (p *awkp) splitregex(re *regexp.Regexp, s string, a fielder) (count int) {
	var f string
	for count, f = range re.Split(s, -1) {
		a.SetField(count+1, p.string(f))
	}
	return count + 1
}

func (p *awkp) splitrune(fs rune, s string, a fielder) (count int) {
	var rs = p.sym("RS").String()
	var field strings.Builder
	for _, r := range s {
		if r == fs || (rs == "" && r == '\n') {
			count++
			a.SetField(count, p.string(field.String()))
			field.Reset()
		} else {
			field.WriteRune(r)
		}
	}
	if len(s) > 0 {
		count++
		a.SetField(count, p.string(field.String()))
	}
	return
}

func (p *awkp) bool(b bool) *awkcell {
	var n float64
	if b {
		n = 1
	}
	return &awkcell{numval: &n, prog: p}
}

func (p *awkp) num(n float64) *awkcell {
	return &awkcell{numval: &n, prog: p}
}

func (p *awkp) string(s string) *awkcell {
	return &awkcell{strval: &s, prog: p}
}

type awkcell struct {
	prog       *awkp
	numval     *float64
	strval     *string
	arrval     *awkmap
	fnval      *awkfn
	assignable bool
	name       string
	next       *awkcell
	string     bool
	regexp     bool
	assignhook func() error
}

func (c *awkcell) Num() float64 {
	if c.numval != nil {
		return *c.numval
	}
	if c.strval == nil || *c.strval == "" {
		return 0
	}
	re := regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?`)
	numstr := re.FindString(*c.strval)
	numval, err := strconv.ParseFloat(numstr, 64)
	if err != nil {
		return 0
	}
	return numval
}

func (c *awkcell) IsString() bool {
	if c.string {
		return true
	} else if c.numval != nil {
		return false
	} else if c.strval == nil {
		return false
	} else if _, err := strconv.ParseFloat(c.String(), 64); err == nil {
		return false
	} else {
		return true
	}
}

func (c *awkcell) SetNum(n float64) {
	c.numval = &n
	c.strval = nil
}

func (c *awkcell) strconv(nconv string) string {
	if c.strval != nil && *c.strval != "" {
		return *c.strval
	} else if c.numval == nil {
		return ""
	}
	format := c.prog.sym(nconv).String()
	if _, frac := math.Modf(*c.numval); frac == 0 {
		format = "%.30g"
	}
	s, err := c.prog.sprintf(format, []*awkcell{c})
	if err != nil {
		return ""
	}
	return s
}

func (c *awkcell) String() string {
	return c.strconv("CONVFMT")
}

func (c *awkcell) OutputString() string {
	return c.strconv("OFMT")
}

func (c *awkcell) SetString(s string) {
	c.strval = &s
	c.numval = nil
}

func (c *awkcell) AssignString(s string) error {
	c.SetString(s)
	return c.AssignHook()
}

func (c *awkcell) Bool() bool {
	if c.numval != nil {
		return c.Num() != 0
	} else {
		return c.String() != ""
	}
}

func (c *awkcell) SetBool(b bool) {
	if b {
		c.SetNum(1)
	} else {
		c.SetNum(0)
	}
}

func (c *awkcell) Arr() *awkmap {
	if c.arrval == nil {
		c.arrval = &awkmap{}
	}
	return c.arrval
}

func (c *awkcell) Key(k string) *awkcell {
	val := c.Arr().get(k)
	if val == nil {
		c.Arr().set(k, &awkcell{prog: c.prog, assignable: true})
	}
	return c.Arr().get(k)
}

func (c *awkcell) HasKey(k string) *awkcell {
	val := c.Arr().get(k)
	return c.prog.bool(val != nil)
}

func (c *awkcell) SetKey(k string, v *awkcell) {
	c.Arr().set(k, v)
}

func (c *awkcell) DelKey(k string) {
	c.Arr().del(k)
}

func (c *awkcell) Field(i int) *awkcell {
	return c.Key(strconv.Itoa(i))
}

func (c *awkcell) SetField(i int, o *awkcell) {
	c.Key(strconv.Itoa(i)).Set(o)
}

func (c *awkcell) Fn() *awkfn {
	if c.fnval == nil {
		c.fnval = &awkfn{}
	}
	return c.fnval
}

func (c *awkcell) SetFn(f *awkfn) {
	c.fnval = f
}

func (c *awkcell) Set(o *awkcell) {
	if o.numval != nil {
		c.SetNum(*o.numval)
	} else {
		c.SetString(o.String())
	}
	c.arrval = o.Arr()
	c.fnval = o.Fn()
	c.prog = o.prog
	c.string = o.string
	c.regexp = o.regexp
}

func (c *awkcell) AssignHook() error {
	if c.assignhook == nil {
		return nil
	}
	return c.assignhook()
}

type awkmap struct {
	count    uint
	size     uint
	contents []*awkcell
}

const (
	awkmapinit = 50
	awkmapfull = 2
	awkmapgrow = 4
)

func (m *awkmap) get(key string) *awkcell {
	if m.size == 0 {
		m.size = awkmapinit
		m.contents = make([]*awkcell, m.size)
	}
	hash := m.hash(key)
	for c := m.contents[hash]; c != nil; c = c.next {
		if c.name == key {
			return c
		}
	}
	return nil
}

func (m *awkmap) set(key string, val *awkcell) {
	if m.size == 0 {
		m.size = awkmapinit
		m.contents = make([]*awkcell, m.size)
	}
	val.name = key
	hash := m.hash(key)
	c := m.contents[hash]
	val.next = c
	m.contents[hash] = val
	m.count++
	if m.count > m.size*awkmapfull {
		m.rehash()
	}
}

func (m *awkmap) del(key string) {
	if m.size == 0 {
		return
	}
	hash := m.hash(key)
	var prevc *awkcell
	for c := m.contents[hash]; c != nil; c = c.next {
		if c.name == key && prevc != nil {
			prevc.next = c.next
			m.count--
			return
		} else if c.name == key {
			m.contents[hash] = c.next
			m.count--
			return
		}
		prevc = c
	}
}

func (m *awkmap) hash(s string) uint {
	var val uint32
	for c := 0; c < len(s); c++ {
		val = uint32(s[c]) + 31*val
	}
	return uint(val) % m.size
}

func (m *awkmap) rehash() {
	m.size *= awkmapgrow
	nc := make([]*awkcell, m.size)
	for i := range m.contents {
		for c := m.contents[i]; c != nil; {
			next := c.next
			hash := m.hash(c.name)
			ncc := nc[hash]
			c.next = ncc
			nc[hash] = c
			c = next
		}
	}
	m.contents = nc
}

func (m *awkmap) reset() {
	m.contents = nil
	m.size = 0
	m.count = 0
}
