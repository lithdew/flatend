package main

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/chzyer/readline"
	"io/ioutil"
	"log"
	"strings"
)

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func hold(fn func() error) {
	check(fn())
}

var runtime cue.Runtime

func main() {
	flag.Parse()

	code := ""

	filename := flag.Arg(0)
	if filename != "" {
		buf, err := ioutil.ReadFile(filename)
		check(err)

		code = string(buf)
	}

	rl, err := readline.New("> ")
	check(err)
	defer hold(rl.Close)

	rl.Config.FuncFilterInputRune = func(r rune) (rune, bool) {
		switch r {
		case readline.CharCtrlZ:
			return 0, false
		default:
			return r, true
		}
	}

	for {
		ln := rl.Line()
		if ln.CanContinue() {
			continue
		}
		if ln.CanBreak() {
			break
		}

		if len(ln.Line) == 0 {
			in, err := runtime.Compile("", code)
			if err == nil {
				err = in.Err
			}
			if err != nil {
				fmt.Fprintln(rl, err)
				continue
			}

			formatted, err := format.Node(in.Value().Syntax())
			if err != nil {
				fmt.Fprintln(rl, err)
				continue
			}

			fmt.Fprint(rl, string(formatted))

			continue
		}

		if ln.Line[0] == ':' {
			fields := strings.SplitN(ln.Line[1:], " ", 2)

			switch fields[0] {
			case "q", "quit":
				return
			case "o", "out":
				in, err := runtime.Compile("", code)
				if err == nil {
					err = in.Err
				}
				if err != nil {
					fmt.Fprintln(rl, err)
					continue
				}

				formatted, err := json.MarshalIndent(in.Value(), "", "\t")
				if err != nil {
					fmt.Fprintln(rl, err)
					continue
				}

				fmt.Fprintln(rl, string(formatted))
			case "a", "append":
				in, err := runtime.Compile("", code)
				if err == nil {
					err = in.Err
				}
				if err != nil {
					fmt.Fprintln(rl, err)
					continue
				}

				out, err := runtime.Compile("", fields[1])
				if err == nil {
					err = in.Err
				}
				if err != nil {
					fmt.Fprintln(rl, err)
					continue
				}

				val := in.Value().Unify(out.Value())
				if err := val.Err(); err != nil {
					fmt.Fprintln(rl, err)
					continue
				}

				formatted, err := format.Node(val.Syntax())
				if err != nil {
					fmt.Fprintln(rl, err)
					continue
				}

				code = string(formatted)
			}

			continue
		}

		expr, err := parser.ParseExpr("", ln.Line)
		if err != nil {
			fmt.Fprintln(rl, err)
			continue
		}

		in, err := runtime.Compile("", code)
		if err == nil {
			err = in.Err
		}
		if err != nil {
			fmt.Fprintln(rl, err)
			continue
		}

		result := in.Eval(expr)
		if err := result.Err(); err != nil {
			fmt.Fprintln(rl, err)
			continue
		}

		formatted, err := format.Node(result.Syntax())
		if err != nil {
			fmt.Fprintln(rl, err)
			continue
		}

		fmt.Fprintln(rl, string(formatted))
	}
}
