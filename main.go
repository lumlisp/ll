package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) >= 2 && args[0] == "-b" {
		err := runBundle(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
			os.Exit(1)
		}
		return
	}

	if len(args) == 0 {
		if bd, err := readBundle(); err == nil {
			runBundled(bd)
			return
		}
		runRepl()
		return
	}

	switch args[0] {
	case "-h", "--help":
		printHelp()
	case "-v", "--version":
		fmt.Println("LL v0.2.0 - Lum Lisp")
	default:
		runFile(args[0], args[1:])
	}
}

func printHelp() {
	fmt.Println("LL v0.2.0 - Lum Lisp")
	fmt.Println("Usage:")
	fmt.Println("  ll                  Start REPL")
	fmt.Println("  ll <file.ll> [args...]  Run script with arguments")
	fmt.Println("  ll -b <file.ll>     Bundle script and deps into executable")
	fmt.Println("  ll -b <file> -o <out>  Bundle with custom output path")
	fmt.Println("  ll -h               Show this help")
	fmt.Println("  ll -v               Show version")
}

func runBundled(bd *BundleData) {
	eval := NewEvalWithVFS(bd.VFS)
	eval.SetScriptArgs(os.Args[1:])
	err := eval.EvalString(bd.Main)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
		os.Exit(1)
	}
}

func runFile(filename string, scriptArgs []string) {
	eval := NewEval()
	eval.SetScriptArgs(scriptArgs)
	eval.SetCurrentFile(filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
		os.Exit(1)
	}

	err = eval.EvalString(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
		os.Exit(1)
	}
}

func runRepl() {
	fmt.Println("Lum Lisp v0.2.0")
	fmt.Println("Type 'exit' to quit, or use Ctrl+D")
	fmt.Println()

	lexer := &Lexer{}
	parser := &Parser{}
	eval := NewEval()
	scanner := bufio.NewScanner(os.Stdin)
	buffer := ""
	continuation := false

	for {
		if continuation {
			fmt.Print("... ")
		} else {
			fmt.Print("ll> ")
		}

		if !scanner.Scan() {
			fmt.Println()
			break
		}

		line := scanner.Text()

		if line == "exit" {
			break
		}

		buffer += line + "\n"

		tokens, err := lexer.Tokenize(buffer)
		if err != nil {
			continuation = isUnterminated(err)
			if !continuation {
				fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
				buffer = ""
			}
			continue
		}

		ast, err := parser.Parse(tokens)
		if err != nil {
			continuation = isUnterminated(err)
			if !continuation {
				fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
				buffer = ""
			}
			continue
		}

		if len(ast) == 0 || allNestedComplete(ast) {
			continuation = false
			var result Value
			for _, expr := range ast {
				var err error
				result, err = eval.Eval(expr)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
					result = Nil
					break
				}
			}
			switch result.(type) {
			case *NilType, *Closure, *Primitive, *Macro:
			default:
				fmt.Println(FormatValue(result))
			}
			buffer = ""
		} else {
			continuation = true
		}
	}
}

func isUnterminated(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "unterminated") || strings.Contains(msg, "unexpected end")
}

func allNestedComplete(ast []Value) bool {
	for _, expr := range ast {
		if !exprComplete(expr) {
			return false
		}
	}
	return true
}

func exprComplete(v Value) bool {
	switch v.(type) {
	case *NilType, Integer, Float, String, Boolean, *Sym:
		return true
	case *Cons:
		return true
	default:
		return true
	}
}
