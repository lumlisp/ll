package main

import (
	"fmt"
	"os"
	"strings"
)

func readFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (e *Eval) builtinJsEncodeString(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("js/encode-string requires 1 argument (ll-code-string)")
	}
	src, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("js/encode-string: argument must be a string")
	}
	js, err := transpileLL(string(src))
	if err != nil {
		return nil, fmt.Errorf("js/encode-string: %v", err)
	}
	return String(js), nil
}

func (e *Eval) builtinJsEncodeFile(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("js/encode-file requires 1 argument (path)")
	}
	path, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("js/encode-file: argument must be a string")
	}
	data, err := readFileString(string(path))
	if err != nil {
		return nil, fmt.Errorf("js/encode-file: %v", err)
	}
	js, err := transpileLL(string(data))
	if err != nil {
		return nil, fmt.Errorf("js/encode-file: %v", err)
	}
	return String(js), nil
}

func transpileLL(input string) (string, error) {
	l := &Lexer{}
	p := &Parser{}
	tokens, err := l.Tokenize(input)
	if err != nil {
		return "", err
	}
	ast, err := p.Parse(tokens)
	if err != nil {
		return "", err
	}
	var out []string
	for _, expr := range ast {
		js, err := transpileExpr(expr, false)
		if err != nil {
			return "", err
		}
		out = append(out, js)
	}
	return strings.Join(out, "\n"), nil
}

func transpileExpr(v Value, asExpr bool) (string, error) {
	switch val := v.(type) {
	case *NilType:
		return "null", nil
	case Integer:
		return fmt.Sprintf("%d", int64(val)), nil
	case Float:
		return fmt.Sprintf("%g", float64(val)), nil
	case String:
		return fmt.Sprintf("%q", string(val)), nil
	case Boolean:
		if val {
			return "true", nil
		}
		return "false", nil
	case *Sym:
		if val.Name == "#t" {
			return "true", nil
		}
		if val.Name == "#f" {
			return "false", nil
		}
		if val.Name == "nil" {
			return "null", nil
		}
		return val.Name, nil
	case *Cons:
		return transpileCons(val)
	case *Vector:
		var items []string
		for _, item := range val.Items {
			s, err := transpileExpr(item, true)
			if err != nil {
				return "", err
			}
			items = append(items, s)
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	default:
		return "", fmt.Errorf("unsupported type: %T", v)
	}
}

func transpileCons(c *Cons) (string, error) {
	if c.Car == Nil {
		return "null", nil
	}
	sym, ok := c.Car.(*Sym)
	if !ok {
		// Function call where car is not a symbol (e.g., lambda call)
		fn, err := transpileExpr(c.Car, true)
		if err != nil {
			return "", err
		}
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		return fn + "(" + strings.Join(args, ", ") + ")", nil
	}

	switch sym.Name {
	case "quote":
		argCons, ok := c.Cdr.(*Cons)
		if !ok || argCons.Cdr != Nil {
			return "", fmt.Errorf("quote: wrong arg count")
		}
		return transpileExpr(argCons.Car, true)

	case "define":
		return transpileDefine(c)

	case "set!":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) < 2 {
			return "", fmt.Errorf("set!: requires 2 arguments")
		}
		return args[0] + " = " + args[1] + ";", nil

	case "if":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 3 {
			return "(" + args[0] + " ? " + args[1] + " : " + args[2] + ")", nil
		}
		if len(args) == 2 {
			return "(" + args[0] + " ? " + args[1] + " : null)", nil
		}
		return "", fmt.Errorf("if: wrong arg count")

	case "cond":
		return transpileCond(c)

	case "begin":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "null", nil
		}
		return "(function() { " + strings.Join(args, "; ") + "; })()", nil

	case "lambda":
		return transpileLambda(c)

	case "and":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "true", nil
		}
		return "(" + strings.Join(args, " && ") + ")", nil

	case "or":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "false", nil
		}
		return "(" + strings.Join(args, " || ") + ")", nil

	case "while":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) < 1 {
			return "", fmt.Errorf("while: requires condition")
		}
		body := "null"
		if len(args) > 1 {
			body = strings.Join(args[1:], "; ")
		}
		return "while(" + args[0] + ") { " + body + "; }", nil

	case "for":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) < 3 {
			return "", fmt.Errorf("for: requires var start end body")
		}
		return "for(let " + args[0] + " = " + args[1] + "; " + args[0] + " < " + args[2] + "; " + args[0] + "++) { " + strings.Join(args[3:], "; ") + "; }", nil

	case "define-macro":
		return "// define-macro not supported in JS", nil

	case "future":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		return "new Promise(function(resolve) { resolve(" + strings.Join(args, "; ") + "); })", nil

	case "await":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("await: requires 1 argument")
		}
		return "await " + args[0], nil

	case "co":
		return transpileCo(c)

	case "return":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "return null", nil
		}
		return "return " + args[0], nil

	case "list":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		return "[" + strings.Join(args, ", ") + "]", nil

	case "cons":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("cons: requires 2 arguments")
		}
		return "[" + args[0] + ", ..." + args[1] + "]", nil

	case "car":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("car: requires 1 argument")
		}
		return args[0] + "[0]", nil

	case "cdr":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("cdr: requires 1 argument")
		}
		return args[0] + ".slice(1)", nil

	case "null?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("null?: requires 1 argument")
		}
		return "(" + args[0] + " === null || " + args[0] + " === undefined || " + args[0] + ".length === 0)", nil

	case "display", "println", "print":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "console.log()", nil
		}
		return "console.log(" + strings.Join(args, ", ") + ")", nil

	case "newline":
		return "console.log()", nil

	case "length":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("length: requires 1 argument")
		}
		return args[0] + ".length", nil

	case "not":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("not: requires 1 argument")
		}
		return "!" + args[0], nil

	case "=":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("=: requires 2 arguments")
		}
		return "(" + args[0] + " === " + args[1] + ")", nil

	case "<":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("<: requires 2 arguments")
		}
		return "(" + args[0] + " < " + args[1] + ")", nil

	case ">":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf(">: requires 2 arguments")
		}
		return "(" + args[0] + " > " + args[1] + ")", nil

	case "<=":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("<=: requires 2 arguments")
		}
		return "(" + args[0] + " <= " + args[1] + ")", nil

	case ">=":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf(">=: requires 2 arguments")
		}
		return "(" + args[0] + " >= " + args[1] + ")", nil

	case "string-append":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) < 1 {
			return "", fmt.Errorf("string-append: requires at least 1 argument")
		}
		return "(" + strings.Join(args, " + ") + ")", nil

	case "string-length":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("string-length: requires 1 argument")
		}
		return args[0] + ".length", nil

	case "string=?", "equal?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("%s: requires 2 arguments", sym.Name)
		}
		return "(" + args[0] + " === " + args[1] + ")", nil

	case "string-upcase":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("string-upcase: requires 1 argument")
		}
		return args[0] + ".toUpperCase()", nil

	case "string-downcase":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("string-downcase: requires 1 argument")
		}
		return args[0] + ".toLowerCase()", nil

	case "string-split":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 1 {
			return args[0] + ".split(/\\s+/)", nil
		}
		if len(args) == 2 {
			return args[0] + ".split(" + args[1] + ")", nil
		}
		return "", fmt.Errorf("string-split: wrong arg count")

	case "string->number":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("string->number: requires 1 argument")
		}
		return "Number(" + args[0] + ")", nil

	case "number->string":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("number->string: requires 1 argument")
		}
		return "String(" + args[0] + ")", nil

	case "integer?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("integer?: requires 1 argument")
		}
		return "Number.isInteger(" + args[0] + ")", nil

	case "string?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("string?: requires 1 argument")
		}
		return "(typeof " + args[0] + " === 'string')", nil

	case "boolean?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("boolean?: requires 1 argument")
		}
		return "(typeof " + args[0] + " === 'boolean')", nil

	case "number?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("number?: requires 1 argument")
		}
		return "(typeof " + args[0] + " === 'number')", nil

	case "fn?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("fn?: requires 1 argument")
		}
		return "(typeof " + args[0] + " === 'function')", nil

	case "symbol?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("symbol?: requires 1 argument")
		}
		return "(typeof " + args[0] + " === 'symbol')", nil

	case "vector":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		return "[" + strings.Join(args, ", ") + "]", nil

	case "list->vector":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("list->vector: requires 1 argument")
		}
		return args[0], nil

	case "vector->list":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("vector->list: requires 1 argument")
		}
		return args[0], nil

	case "exit":
		return "process.exit(0)", nil

	case "sleep":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("sleep: requires 1 argument")
		}
		return "new Promise(r => setTimeout(r, " + args[0] + " * 1000))", nil

	case "string-trim":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("string-trim: requires 1 argument")
		}
		return args[0] + ".trim()", nil

	case "substring":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 2 {
			return args[0] + ".slice(" + args[1] + ")", nil
		}
		if len(args) == 3 {
			return args[0] + ".slice(" + args[1] + ", " + args[2] + ")", nil
		}
		return "", fmt.Errorf("substring: requires 2 or 3 arguments")

	case "string-join":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("string-join: requires 2 arguments")
		}
		return args[0] + ".join(" + args[1] + ")", nil

	case "range":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 1 {
			return "Array.from({length: " + args[0] + "}, (_, i) => i)", nil
		}
		if len(args) == 2 {
			return "Array.from({length: " + args[1] + " - " + args[0] + "}, (_, i) => i + " + args[0] + ")", nil
		}
		return "", fmt.Errorf("range: requires 1 or 2 arguments")

	case "map":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("map: requires 2 arguments")
		}
		return args[1] + ".map(" + args[0] + ")", nil

	case "filter":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("filter: requires 2 arguments")
		}
		return args[1] + ".filter(" + args[0] + ")", nil

	case "foldl":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 3 {
			return "", fmt.Errorf("foldl: requires 3 arguments")
		}
		return args[2] + ".reduce(" + args[0] + ", " + args[1] + ")", nil

	case "append":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "[]", nil
		}
		return "[..." + strings.Join(args, ", ...") + "]", nil

	case "reverse":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("reverse: requires 1 argument")
		}
		return args[0] + ".slice().reverse()", nil

	case "member":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("member: requires 2 arguments")
		}
		return args[1] + ".includes(" + args[0] + ") ? " + args[1] + " : false", nil

	case "assoc":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("assoc: requires 2 arguments")
		}
		return args[1] + ".find(function(p) { return p[0] === " + args[0] + "; }) || false", nil

	case "list-ref":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("list-ref: requires 2 arguments")
		}
		return args[0] + "[" + args[1] + "]", nil

	case "cons?", "pair?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("%s: requires 1 argument", sym.Name)
		}
		return "Array.isArray(" + args[0] + ")", nil

	case "list?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("list?: requires 1 argument")
		}
		return "Array.isArray(" + args[0] + ")", nil

	case "zero?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("zero?: requires 1 argument")
		}
		return "(" + args[0] + " === 0)", nil

	case "even?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("even?: requires 1 argument")
		}
		return "(" + args[0] + " % 2 === 0)", nil

	case "odd?":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("odd?: requires 1 argument")
		}
		return "(" + args[0] + " % 2 === 1)", nil

	case "+":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "0", nil
		}
		return "(" + strings.Join(args, " + ") + ")", nil

	case "-":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "0", nil
		}
		if len(args) == 1 {
			return "(-" + args[0] + ")", nil
		}
		return "(" + strings.Join(args, " - ") + ")", nil

	case "*":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "1", nil
		}
		return "(" + strings.Join(args, " * ") + ")", nil

	case "/":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 1 {
			return "(1 / " + args[0] + ")", nil
		}
		return "(" + strings.Join(args, " / ") + ")", nil

	case "%":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("%%: requires 2 arguments")
		}
		return "(" + args[0] + " % " + args[1] + ")", nil

	case "expt":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 2 {
			return "", fmt.Errorf("expt: requires 2 arguments")
		}
		return "Math.pow(" + strings.Join(args, ", ") + ")", nil

	case "sqrt":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("sqrt: requires 1 argument")
		}
		return "Math.sqrt(" + args[0] + ")", nil

	case "abs":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("abs: requires 1 argument")
		}
		return "Math.abs(" + args[0] + ")", nil

	case "min":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		return "Math.min(" + strings.Join(args, ", ") + ")", nil

	case "max":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		return "Math.max(" + strings.Join(args, ", ") + ")", nil

	case "floor":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("floor: requires 1 argument")
		}
		return "Math.floor(" + args[0] + ")", nil

	case "ceil":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("ceil: requires 1 argument")
		}
		return "Math.ceil(" + args[0] + ")", nil

	case "round":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("round: requires 1 argument")
		}
		return "Math.round(" + args[0] + ")", nil

	case "inc":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("inc: requires 1 argument")
		}
		return "(" + args[0] + " + 1)", nil

	case "dec":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) != 1 {
			return "", fmt.Errorf("dec: requires 1 argument")
		}
		return "(" + args[0] + " - 1)", nil

	case "new", "send", "slot-ref", "slot-set!", "instance?", "class-of", "add-method":
		return "// " + sym.Name + " not supported in JS", nil

	case "require", "include":
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		if len(args) == 0 {
			return "", fmt.Errorf("require: filename required")
		}
		return "// require " + args[0], nil

	default:
		// Regular function call
		args, err := transpileArgs(c.Cdr)
		if err != nil {
			return "", err
		}
		return sym.Name + "(" + strings.Join(args, ", ") + ")", nil
	}
}

func transpileDefine(c *Cons) (string, error) {
	args := c.Cdr
	firstCons, ok := args.(*Cons)
	if !ok {
		return "", fmt.Errorf("define: requires at least 2 arguments")
	}
	first := firstCons.Car
	rest := firstCons.Cdr

	if sym, ok := first.(*Sym); ok {
		valCons, ok := rest.(*Cons)
		if !ok {
			return "let " + sym.Name + " = null", nil
		}
		val, err := transpileExpr(valCons.Car, true)
		if err != nil {
			return "", err
		}
		return "let " + sym.Name + " = " + val + ";", nil
	}

	listCons, ok := first.(*Cons)
	if !ok {
		return "", fmt.Errorf("invalid define syntax")
	}
	fnSym, ok := listCons.Car.(*Sym)
	if !ok {
		return "", fmt.Errorf("define: function name must be a symbol")
	}

	var params []string
	paramList := listCons.Cdr
	for paramList != Nil {
		pc, ok := paramList.(*Cons)
		if !ok {
			break
		}
		if psym, ok := pc.Car.(*Sym); ok {
			params = append(params, psym.Name)
		}
		paramList = pc.Cdr
	}

	var body []string
	for rest != Nil {
		bc, ok := rest.(*Cons)
		if !ok {
			break
		}
		js, err := transpileExpr(bc.Car, false)
		if err != nil {
			return "", err
		}
		body = append(body, js)
		rest = bc.Cdr
	}

	bodyStr := strings.Join(body, "; ")
	return "function " + fnSym.Name + "(" + strings.Join(params, ", ") + ") { " + bodyStr + "; }", nil
}

func transpileLambda(c *Cons) (string, error) {
	args := c.Cdr
	paramsCons, ok := args.(*Cons)
	if !ok {
		return "", fmt.Errorf("lambda: requires parameter list")
	}
	paramsList := paramsCons.Car
	bodyList := paramsCons.Cdr

	var params []string
	if paramsList != Nil {
		for p := paramsList; p != Nil; p = p.(*Cons).Cdr {
			pc := p.(*Cons)
			if psym, ok := pc.Car.(*Sym); ok {
				params = append(params, psym.Name)
			}
		}
	}

	var body []string
	for bodyList != Nil {
		bc, ok := bodyList.(*Cons)
		if !ok {
			break
		}
		js, err := transpileExpr(bc.Car, false)
		if err != nil {
			return "", err
		}
		body = append(body, js)
		bodyList = bc.Cdr
	}

	if len(body) == 0 {
		return "(function(" + strings.Join(params, ", ") + ") { return null; })", nil
	}
	return "(function(" + strings.Join(params, ", ") + ") { " + strings.Join(body, "; ") + "; })", nil
}

func transpileCo(c *Cons) (string, error) {
	args := c.Cdr
	paramsCons, ok := args.(*Cons)
	if !ok {
		return "", fmt.Errorf("co: requires parameter list")
	}
	paramsList := paramsCons.Car
	bodyList := paramsCons.Cdr

	var params []string
	if paramsList != Nil {
		for p := paramsList; p != Nil; p = p.(*Cons).Cdr {
			pc := p.(*Cons)
			if psym, ok := pc.Car.(*Sym); ok {
				params = append(params, psym.Name)
			}
		}
	}

	var body []string
	for bodyList != Nil {
		bc, ok := bodyList.(*Cons)
		if !ok {
			break
		}
		js, err := transpileExpr(bc.Car, false)
		if err != nil {
			return "", err
		}
		body = append(body, js)
		bodyList = bc.Cdr
	}

	if len(body) == 0 {
		return "(async function(" + strings.Join(params, ", ") + ") { return null; })", nil
	}
	return "(async function(" + strings.Join(params, ", ") + ") { " + strings.Join(body, "; ") + "; })", nil
}

func transpileCond(c *Cons) (string, error) {
	clauses := c.Cdr
	var result strings.Builder
	result.WriteByte('(')
	first := true
	hasElse := false
	for clauses != Nil {
		clauseCons, ok := clauses.(*Cons)
		if !ok {
			break
		}
		clause, ok := clauseCons.Car.(*Cons)
		if !ok {
			return "", fmt.Errorf("bad cond clause")
		}

		isElse := false
		if sym, ok := clause.Car.(*Sym); ok && sym.Name == "else" {
			isElse = true
			hasElse = true
		}

		var bodyJS string
		bodyExprs := clause.Cdr
		if bodyExprs != Nil {
			bodyCons, ok := bodyExprs.(*Cons)
			if ok {
				if bodyCons.Cdr == Nil {
					var err error
					bodyJS, err = transpileExpr(bodyCons.Car, true)
					if err != nil {
						return "", err
					}
				} else {
					var err error
					bodyJS, err = transpileExpr(&Cons{Car: &Sym{Name: "begin"}, Cdr: bodyExprs}, true)
					if err != nil {
						return "", err
					}
				}
			}
		}
		if bodyJS == "" {
			bodyJS = "null"
		}

		if isElse {
			if !first {
				result.WriteString(" : ")
			}
			result.WriteString(bodyJS)
		} else {
			test, err := transpileExpr(clause.Car, true)
			if err != nil {
				return "", err
			}
			if !first {
				result.WriteString(" : ")
			}
			result.WriteString(test + " ? " + bodyJS)
			first = false
		}
		clauses = clauseCons.Cdr
	}
	if first && !hasElse {
		result.WriteString("null")
	}
	result.WriteByte(')')
	return result.String(), nil
}

func transpileArgs(v Value) ([]string, error) {
	var result []string
	for v != Nil {
		cons, ok := v.(*Cons)
		if !ok {
			break
		}
		s, err := transpileExpr(cons.Car, true)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
		v = cons.Cdr
	}
	return result, nil
}
