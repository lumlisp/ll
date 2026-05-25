package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
)

//go:embed stdlib.ll
var stdlibSource string

type Eval struct {
	env         *Env
	lexer       *Lexer
	parser      *Parser
	w           io.Writer
	vfs         map[string]string
	currentFile string
}

func NewEval() *Eval {
	return newEval(nil)
}

func NewEvalWithVFS(vfs map[string]string) *Eval {
	return newEval(vfs)
}

func newEval(vfs map[string]string) *Eval {
	e := &Eval{
		env:    NewEnv(nil),
		lexer:  &Lexer{},
		parser: &Parser{},
		w:      os.Stdout,
		vfs:    vfs,
	}
	e.initBuiltins()
	e.loadStdlib()
	return e
}

func (e *Eval) loadStdlib() {
	err := e.EvalString(stdlibSource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mWarning: stdlib error: %v\033[0m\n", err)
	}
}

func (e *Eval) SetOutput(w io.Writer) {
	e.w = w
}

func (e *Eval) SetScriptArgs(args []string) {
	vals := make([]Value, len(args))
	for i, a := range args {
		vals[i] = String(a)
	}
	e.env.Set("*args*", SliceToList(vals))
}

func (e *Eval) SetCurrentFile(file string) {
	e.currentFile = file
}

func (e *Eval) Env() *Env {
	return e.env
}

func (e *Eval) EvalFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return e.EvalString(string(data))
}

func (e *Eval) EvalString(input string) error {
	tokens, err := e.lexer.Tokenize(input)
	if err != nil {
		return err
	}
	ast, err := e.parser.Parse(tokens)
	if err != nil {
		return err
	}
	for _, expr := range ast {
		_, err := e.Eval(expr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Eval) Eval(expr Value) (Value, error) {
	return e.eval(expr, e.env)
}

func (e *Eval) eval(expr Value, env *Env) (Value, error) {
	switch val := expr.(type) {
	case *NilType, Integer, Float, String, Boolean, *Vector, *ClassType, *Instance:
		return val, nil
	}

	if sym, ok := expr.(*Sym); ok {
		return env.Get(sym.Name)
	}

	cons, ok := expr.(*Cons)
	if !ok {
		return Nil, fmt.Errorf("cannot evaluate: %v", expr)
	}

	if cons.Car == Nil {
		return Nil, nil
	}

	sym, ok := cons.Car.(*Sym)
	if ok {
		switch sym.Name {
		case "quote":
			return e.evalQuote(cons, env)
		case "define":
			return e.evalDefine(cons, env)
		case "set!":
			return e.evalSet(cons, env)
		case "if":
			return e.evalIf(cons, env)
		case "cond":
			return e.evalCond(cons, env)
		case "lambda":
			return e.evalLambda(cons, env)
		case "begin":
			return e.evalBegin(cons, env)
		case "while":
			return e.evalWhile(cons, env)
		case "for":
			return e.evalFor(cons, env)
		case "and":
			return e.evalAnd(cons, env)
		case "or":
			return e.evalOr(cons, env)
		case "require", "include":
			return e.evalRequire(cons, env)
		case "define-macro":
			return e.evalDefineMacro(cons, env)
		case "future":
			return e.evalFuture(cons, env)
		case "await":
			return e.evalAwait(cons, env)
		case "co":
			return e.evalCo(cons, env)
		case "return":
			return e.evalReturn(cons, env)
		}
	}

	return e.evalCall(cons, env)
}

func (e *Eval) evalReturn(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	argCons, ok := args.(*Cons)
	var val Value = Nil
	if ok {
		var err error
		val, err = e.eval(argCons.Car, env)
		if err != nil {
			return nil, err
		}
	}
	return nil, &ReturnSignal{Value: val}
}

func (e *Eval) evalQuote(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	cons, ok := args.(*Cons)
	if !ok {
		return Nil, nil
	}
	return cons.Car, nil
}

func (e *Eval) evalDefine(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	firstCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("define requires at least 2 arguments")
	}

	first := firstCons.Car
	rest := firstCons.Cdr

	if sym, ok := first.(*Sym); ok {
		valCons, ok := rest.(*Cons)
		var val Value = Nil
		if ok {
			var err error
			val, err = e.eval(valCons.Car, env)
			if err != nil {
				return nil, err
			}
		}
		env.Set(sym.Name, val)
		return val, nil
	}

	listCons, ok := first.(*Cons)
	if !ok {
		return nil, fmt.Errorf("invalid define syntax")
	}

	fnSym, ok := listCons.Car.(*Sym)
	if !ok {
		return nil, fmt.Errorf("define: function name must be a symbol")
	}

	var params []*Sym
	hasRest := false
	paramList := listCons.Cdr
	for paramList != Nil {
		pc, ok := paramList.(*Cons)
		if !ok {
			break
		}
		psym, ok := pc.Car.(*Sym)
		if !ok {
			return nil, fmt.Errorf("define: function parameters must be symbols")
		}
		if psym.Name == "&rest" {
			hasRest = true
		}
		params = append(params, psym)
		paramList = pc.Cdr
	}

	var body []Value
	for rest != Nil {
		bc, ok := rest.(*Cons)
		if !ok {
			break
		}
		body = append(body, bc.Car)
		rest = bc.Cdr
	}

	fn := &Closure{
		Env:     env,
		Params:  params,
		Body:    body,
		HasRest: hasRest,
	}
	env.Set(fnSym.Name, fn)
	return fn, nil
}

func (e *Eval) evalSet(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	firstCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("set! requires 2 arguments")
	}

	sym, ok := firstCons.Car.(*Sym)
	if !ok {
		return nil, fmt.Errorf("set!: first argument must be a symbol")
	}

	valCons, ok := firstCons.Cdr.(*Cons)
	if !ok {
		return nil, fmt.Errorf("set! requires 2 arguments")
	}

	val, err := e.eval(valCons.Car, env)
	if err != nil {
		return nil, err
	}

	err = env.SetMutate(sym.Name, val)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (e *Eval) evalIf(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	condCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("if requires at least 2 arguments")
	}

	condition, err := e.eval(condCons.Car, env)
	if err != nil {
		return nil, err
	}

	thenCons, ok := condCons.Cdr.(*Cons)
	if !ok {
		return Nil, nil
	}

	if IsTruthy(condition) {
		return e.eval(thenCons.Car, env)
	}

	elseCons := thenCons.Cdr
	if elseCons != Nil {
		ec, ok := elseCons.(*Cons)
		if ok {
			return e.eval(ec.Car, env)
		}
	}
	return Nil, nil
}

func (e *Eval) evalCond(expr *Cons, env *Env) (Value, error) {
	clauses := expr.Cdr
	for clauses != Nil {
		clauseCons, ok := clauses.(*Cons)
		if !ok {
			break
		}
		clause, ok := clauseCons.Car.(*Cons)
		if !ok {
			return nil, fmt.Errorf("bad cond clause")
		}
		clauseRest := clauseCons.Cdr

		test := clause.Car
		isElse := false
		if sym, ok := test.(*Sym); ok && sym.Name == "else" {
			isElse = true
		}

		var testResult bool
		if isElse {
			testResult = true
		} else {
			tv, err := e.eval(test, env)
			if err != nil {
				return nil, err
			}
			testResult = IsTruthy(tv)
		}

		if testResult {
			bodyExprs := clause.Cdr
			bodyCons, ok := bodyExprs.(*Cons)
			if !ok {
				return Nil, nil
			}
			if bodyCons.Cdr == Nil {
				return e.eval(bodyCons.Car, env)
			}
			return e.evalBegin(&Cons{Car: &Sym{Name: "begin"}, Cdr: bodyExprs}, env)
		}

		clauses = clauseRest
	}
	return Nil, nil
}

func (e *Eval) evalLambda(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	paramsCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("lambda requires parameter list")
	}

	paramsList := paramsCons.Car
	bodyList := paramsCons.Cdr

	if paramsList != Nil {
		if _, ok := paramsList.(*Cons); !ok {
			return nil, fmt.Errorf("lambda: parameter list must be a list")
		}
	}

	var params []*Sym
	hasRest := false
	if paramsList != Nil {
		for p := paramsList; p != Nil; p = p.(*Cons).Cdr {
			pc := p.(*Cons)
			psym, ok := pc.Car.(*Sym)
			if !ok {
				return nil, fmt.Errorf("lambda: parameters must be symbols")
			}
			if psym.Name == "&rest" {
				hasRest = true
			}
			params = append(params, psym)
		}
	}

	var body []Value
	for bodyList != Nil {
		bc, ok := bodyList.(*Cons)
		if !ok {
			break
		}
		body = append(body, bc.Car)
		bodyList = bc.Cdr
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("lambda requires at least one body expression")
	}

	return &Closure{
		Env:     env,
		Params:  params,
		Body:    body,
		HasRest: hasRest,
	}, nil
}

func (e *Eval) evalBegin(expr *Cons, env *Env) (Value, error) {
	body := expr.Cdr
	var result Value = Nil
	for body != Nil {
		bc, ok := body.(*Cons)
		if !ok {
			break
		}
		var err error
		result, err = e.eval(bc.Car, env)
		if err != nil {
			return nil, err
		}
		body = bc.Cdr
	}
	return result, nil
}

func (e *Eval) evalWhile(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	condCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("while requires a condition")
	}
	condition := condCons.Car
	bodyList := condCons.Cdr

	var result Value = Nil
	for {
		cv, err := e.eval(condition, env)
		if err != nil {
			return nil, err
		}
		if !IsTruthy(cv) {
			break
		}
		for b := bodyList; b != Nil; b = b.(*Cons).Cdr {
			bc := b.(*Cons)
			result, err = e.eval(bc.Car, env)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (e *Eval) evalFor(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	varCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("for requires a variable")
	}
	varSym, ok := varCons.Car.(*Sym)
	if !ok {
		return nil, fmt.Errorf("for: first argument must be a symbol")
	}

	startCons, ok := varCons.Cdr.(*Cons)
	if !ok {
		return nil, fmt.Errorf("for requires start value")
	}
	start, err := e.eval(startCons.Car, env)
	if err != nil {
		return nil, err
	}
	startInt, ok := start.(Integer)
	if !ok {
		return nil, fmt.Errorf("for: start must be an integer")
	}

	endCons, ok := startCons.Cdr.(*Cons)
	if !ok {
		return nil, fmt.Errorf("for requires end value")
	}
	end, err := e.eval(endCons.Car, env)
	if err != nil {
		return nil, err
	}
	endInt, ok := end.(Integer)
	if !ok {
		return nil, fmt.Errorf("for: end must be an integer")
	}

	bodyList := endCons.Cdr

	var result Value = Nil
	for i := int64(startInt); i < int64(endInt); i++ {
		env.Set(varSym.Name, Integer(i))
		for b := bodyList; b != Nil; b = b.(*Cons).Cdr {
			bc := b.(*Cons)
			result, err = e.eval(bc.Car, env)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (e *Eval) evalAnd(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	if args == Nil {
		return Boolean(true), nil
	}
	var last Value = Boolean(true)
	for args != Nil {
		ac := args.(*Cons)
		result, err := e.eval(ac.Car, env)
		if err != nil {
			return nil, err
		}
		if !IsTruthy(result) {
			return result, nil
		}
		last = result
		args = ac.Cdr
	}
	return last, nil
}

func (e *Eval) evalOr(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	if args == Nil {
		return Boolean(false), nil
	}
	for args != Nil {
		ac, ok := args.(*Cons)
		if !ok {
			break
		}
		result, err := e.eval(ac.Car, env)
		if err != nil {
			return nil, err
		}
		if IsTruthy(result) {
			return result, nil
		}
		args = ac.Cdr
	}
	return Boolean(false), nil
}

func (e *Eval) evalRequire(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	fileCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("require: filename required")
	}
	filename, ok := fileCons.Car.(String)
	if !ok {
		return nil, fmt.Errorf("require: filename must be a string")
	}

	var source string
	if e.vfs != nil {
		if content, ok := e.vfs[string(filename)]; ok {
			source = content
		}
	}
	if source == "" {
		data, err := os.ReadFile(string(filename))
		if err != nil {
			return nil, fmt.Errorf("cannot find file: %s", filename)
		}
		source = string(data)
	}

	tokens, err := e.lexer.Tokenize(source)
	if err != nil {
		return nil, err
	}
	ast, err := e.parser.Parse(tokens)
	if err != nil {
		return nil, err
	}
	var result Value = Nil
	for _, expr := range ast {
		result, err = e.eval(expr, env)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (e *Eval) evalCall(expr *Cons, env *Env) (Value, error) {
	fn, err := e.eval(expr.Car, env)
	if err != nil {
		return nil, err
	}

	if m, ok := fn.(*Macro); ok {
		var rawArgs []Value
		argList := expr.Cdr
		for argList != Nil {
			ac := argList.(*Cons)
			rawArgs = append(rawArgs, ac.Car)
			argList = ac.Cdr
		}
		result, err := e.applyMacro(m, rawArgs)
		if err != nil {
			return nil, err
		}
		return e.eval(result, env)
	}

	var args []Value
	argList := expr.Cdr
	for argList != Nil {
		ac, ok := argList.(*Cons)
		if !ok {
			break
		}
		ev, err := e.eval(ac.Car, env)
		if err != nil {
			return nil, err
		}
		args = append(args, ev)
		argList = ac.Cdr
	}

	return e.Apply(fn, args)
}

func (e *Eval) Apply(fn Value, args []Value) (Value, error) {
	switch f := fn.(type) {
	case *Primitive:
		return f.Fn(args)
	case *Closure:
		return e.applyClosure(f, args)
	case *Macro:
		result, err := e.applyMacro(f, args)
		if err != nil {
			return nil, err
		}
		return e.eval(result, e.env)
	default:
		return nil, fmt.Errorf("not callable: %s", Sprint(fn))
	}
}

func (e *Eval) applyClosure(fn *Closure, args []Value) (Value, error) {
	env := fn.Env.Extend()

	if fn.HasRest {
		regularCount := 0
		restIdx := -1
		for i, p := range fn.Params {
			if p.Name == "&rest" {
				restIdx = i
				break
			}
			regularCount++
		}
		for i := 0; i < regularCount; i++ {
			if i >= len(args) {
				return nil, fmt.Errorf("missing argument: %s", fn.Params[i].Name)
			}
			env.Set(fn.Params[i].Name, args[i])
		}
		if restIdx >= 0 && restIdx+1 < len(fn.Params) {
			restParam := fn.Params[restIdx+1]
			env.Set(restParam.Name, SliceToList(args[regularCount:]))
		}
	} else {
		if len(args) != len(fn.Params) {
			return nil, fmt.Errorf("expected %d arguments, got %d", len(fn.Params), len(args))
		}
		for i, p := range fn.Params {
			env.Set(p.Name, args[i])
		}
	}

	if fn.isAsync {
		f := NewFuture()
		go func() {
			var result Value = Nil
			for _, bodyExpr := range fn.Body {
				var err error
				result, err = e.eval(bodyExpr, env)
				if err != nil {
					if rs, ok := err.(*ReturnSignal); ok {
						f.Resolve(rs.Value, nil)
					} else {
						f.Resolve(Nil, err)
					}
					return
				}
			}
			f.Resolve(result, nil)
		}()
		return f, nil
	}

	var result Value = Nil
	for _, bodyExpr := range fn.Body {
		var err error
		result, err = e.eval(bodyExpr, env)
		if err != nil {
			if rs, ok := err.(*ReturnSignal); ok {
				return rs.Value, nil
			}
			return nil, err
		}
	}
	return result, nil
}

func (e *Eval) evalDefineMacro(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	firstCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("define-macro requires at least 2 arguments")
	}
	first := firstCons.Car
	rest := firstCons.Cdr

	listCons, ok := first.(*Cons)
	if !ok {
		return nil, fmt.Errorf("define-macro: (name params) list expected")
	}

	fnSym, ok := listCons.Car.(*Sym)
	if !ok {
		return nil, fmt.Errorf("define-macro: macro name must be a symbol")
	}

	var params []*Sym
	hasRest := false
	paramList := listCons.Cdr
	for paramList != Nil {
		pc := paramList.(*Cons)
		psym, ok := pc.Car.(*Sym)
		if !ok {
			return nil, fmt.Errorf("define-macro: parameters must be symbols")
		}
		if psym.Name == "&rest" {
			hasRest = true
		}
		params = append(params, psym)
		paramList = pc.Cdr
	}

	var body []Value
	for rest != Nil {
		bc := rest.(*Cons)
		body = append(body, bc.Car)
		rest = bc.Cdr
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("define-macro requires at least one body expression")
	}

	m := &Macro{Env: env, Params: params, Body: body, HasRest: hasRest}
	env.Set(fnSym.Name, m)
	return m, nil
}

func (e *Eval) applyMacro(m *Macro, rawArgs []Value) (Value, error) {
	env := m.Env.Extend()
	if m.HasRest {
		regularCount := 0
		restIdx := -1
		for i, p := range m.Params {
			if p.Name == "&rest" {
				restIdx = i
				break
			}
			regularCount++
		}
		for i := 0; i < regularCount; i++ {
			if i >= len(rawArgs) {
				return nil, fmt.Errorf("missing argument in macro: %s", m.Params[i].Name)
			}
			env.Set(m.Params[i].Name, rawArgs[i])
		}
		if restIdx >= 0 && restIdx+1 < len(m.Params) {
			restParam := m.Params[restIdx+1]
			env.Set(restParam.Name, SliceToList(rawArgs[regularCount:]))
		}
	} else {
		if len(rawArgs) != len(m.Params) {
			return nil, fmt.Errorf("macro: expected %d arguments, got %d", len(m.Params), len(rawArgs))
		}
		for i, p := range m.Params {
			env.Set(p.Name, rawArgs[i])
		}
	}
	var result Value = Nil
	for _, bodyExpr := range m.Body {
		var err error
		result, err = e.eval(bodyExpr, env)
		if err != nil {
			if rs, ok := err.(*ReturnSignal); ok {
				return rs.Value, nil
			}
			return nil, err
		}
	}
	return result, nil
}

func (e *Eval) evalFuture(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	if args == Nil {
		return nil, fmt.Errorf("future requires at least one body expression")
	}

	f := NewFuture()

	go func() {
		body := args
		var result Value = Nil
		var err error
		for body != Nil {
			bc, ok := body.(*Cons)
			if !ok {
				break
			}
			result, err = e.eval(bc.Car, env)
			if err != nil {
				f.Resolve(Nil, err)
				return
			}
			body = bc.Cdr
		}
		f.Resolve(result, nil)
	}()

	return f, nil
}

func (e *Eval) evalAwait(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	argCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("await requires 1 argument")
	}

	val, err := e.eval(argCons.Car, env)
	if err != nil {
		return nil, err
	}

	f, ok := val.(*Future)
	if !ok {
		return nil, fmt.Errorf("await: argument must be a future")
	}

	return f.Await()
}

func (e *Eval) evalCo(expr *Cons, env *Env) (Value, error) {
	args := expr.Cdr
	paramsCons, ok := args.(*Cons)
	if !ok {
		return nil, fmt.Errorf("co requires parameter list")
	}

	paramsList := paramsCons.Car
	bodyList := paramsCons.Cdr

	if paramsList != Nil {
		if _, ok := paramsList.(*Cons); !ok {
			return nil, fmt.Errorf("co: parameter list must be a list")
		}
	}

	var params []*Sym
	hasRest := false
	if paramsList != Nil {
		for p := paramsList; p != Nil; p = p.(*Cons).Cdr {
			pc := p.(*Cons)
			psym, ok := pc.Car.(*Sym)
			if !ok {
				return nil, fmt.Errorf("co: parameters must be symbols")
			}
			if psym.Name == "&rest" {
				hasRest = true
			}
			params = append(params, psym)
		}
	}

	var body []Value
	for bodyList != Nil {
		bc, ok := bodyList.(*Cons)
		if !ok {
			break
		}
		body = append(body, bc.Car)
		bodyList = bc.Cdr
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("co requires at least one body expression")
	}

	return &Closure{
		Env:     env,
		Params:  params,
		Body:    body,
		HasRest: hasRest,
		isAsync: true,
	}, nil
}

func (e *Eval) initBuiltins() {
	e.env.Set("nil", Nil)
	e.env.Set("true", Boolean(true))
	e.env.Set("false", Boolean(false))
	e.env.Set("*args*", Nil)

	e.env.Set("+", &Primitive{Name: "+", Fn: e.builtinAdd})
	e.env.Set("-", &Primitive{Name: "-", Fn: e.builtinSub})
	e.env.Set("*", &Primitive{Name: "*", Fn: e.builtinMul})
	e.env.Set("/", &Primitive{Name: "/", Fn: e.builtinDiv})
	e.env.Set("%", &Primitive{Name: "%", Fn: e.builtinMod})

	e.env.Set("=", &Primitive{Name: "=", Fn: e.builtinNumEq})
	e.env.Set(">", &Primitive{Name: ">", Fn: e.builtinGt})
	e.env.Set("<", &Primitive{Name: "<", Fn: e.builtinLt})
	e.env.Set(">=", &Primitive{Name: ">=", Fn: e.builtinGte})
	e.env.Set("<=", &Primitive{Name: "<=", Fn: e.builtinLte})

	e.env.Set("car", &Primitive{Name: "car", Fn: e.builtinCar})
	e.env.Set("cdr", &Primitive{Name: "cdr", Fn: e.builtinCdr})
	e.env.Set("cons", &Primitive{Name: "cons", Fn: e.builtinCons})
	e.env.Set("list", &Primitive{Name: "list", Fn: e.builtinList})
	e.env.Set("null?", &Primitive{Name: "null?", Fn: e.builtinIsNull})
	e.env.Set("pair?", &Primitive{Name: "pair?", Fn: e.builtinIsPair})
	e.env.Set("length", &Primitive{Name: "length", Fn: e.builtinLength})
	e.env.Set("append", &Primitive{Name: "append", Fn: e.builtinAppend})
	e.env.Set("reverse", &Primitive{Name: "reverse", Fn: e.builtinReverse})
	e.env.Set("list-ref", &Primitive{Name: "list-ref", Fn: e.builtinListRef})
	e.env.Set("list-tail", &Primitive{Name: "list-tail", Fn: e.builtinListTail})
	e.env.Set("take", &Primitive{Name: "take", Fn: e.builtinTake})
	e.env.Set("drop", &Primitive{Name: "drop", Fn: e.builtinDrop})
	e.env.Set("range", &Primitive{Name: "range", Fn: e.builtinRange})
	e.env.Set("member", &Primitive{Name: "member", Fn: e.builtinMember})
	e.env.Set("assoc", &Primitive{Name: "assoc", Fn: e.builtinAssoc})
	e.env.Set("map", &Primitive{Name: "map", Fn: e.builtinMap})
	e.env.Set("filter", &Primitive{Name: "filter", Fn: e.builtinFilter})
	e.env.Set("foldl", &Primitive{Name: "foldl", Fn: e.builtinFoldl})
	e.env.Set("foldr", &Primitive{Name: "foldr", Fn: e.builtinFoldr})

	e.env.Set("symbol?", &Primitive{Name: "symbol?", Fn: e.builtinIsSymbol})
	e.env.Set("number?", &Primitive{Name: "number?", Fn: e.builtinIsNumber})
	e.env.Set("integer?", &Primitive{Name: "integer?", Fn: e.builtinIsInteger})
	e.env.Set("float?", &Primitive{Name: "float?", Fn: e.builtinIsFloat})
	e.env.Set("string?", &Primitive{Name: "string?", Fn: e.builtinIsString})
	e.env.Set("boolean?", &Primitive{Name: "boolean?", Fn: e.builtinIsBoolean})
	e.env.Set("list?", &Primitive{Name: "list?", Fn: e.builtinIsList})
	e.env.Set("fn?", &Primitive{Name: "fn?", Fn: e.builtinIsFn})
	e.env.Set("future?", &Primitive{Name: "future?", Fn: e.builtinIsFuture})
	e.env.Set("not", &Primitive{Name: "not", Fn: e.builtinNot})
	e.env.Set("zero?", &Primitive{Name: "zero?", Fn: e.builtinZero})
	e.env.Set("even?", &Primitive{Name: "even?", Fn: e.builtinEven})
	e.env.Set("odd?", &Primitive{Name: "odd?", Fn: e.builtinOdd})
	e.env.Set("positive?", &Primitive{Name: "positive?", Fn: e.builtinPositive})
	e.env.Set("negative?", &Primitive{Name: "negative?", Fn: e.builtinNegative})
	e.env.Set("equal?", &Primitive{Name: "equal?", Fn: e.builtinEqual})
	e.env.Set("eq?", &Primitive{Name: "eq?", Fn: e.builtinEq})

	e.env.Set("abs", &Primitive{Name: "abs", Fn: e.builtinAbs})
	e.env.Set("min", &Primitive{Name: "min", Fn: e.builtinMin})
	e.env.Set("max", &Primitive{Name: "max", Fn: e.builtinMax})
	e.env.Set("expt", &Primitive{Name: "expt", Fn: e.builtinExpt})
	e.env.Set("sqrt", &Primitive{Name: "sqrt", Fn: e.builtinSqrt})
	e.env.Set("quotient", &Primitive{Name: "quotient", Fn: e.builtinQuotient})
	e.env.Set("remainder", &Primitive{Name: "remainder", Fn: e.builtinRemainder})
	e.env.Set("floor", &Primitive{Name: "floor", Fn: e.builtinFloor})
	e.env.Set("ceil", &Primitive{Name: "ceil", Fn: e.builtinCeil})
	e.env.Set("round", &Primitive{Name: "round", Fn: e.builtinRound})
	e.env.Set("inc", &Primitive{Name: "inc", Fn: e.builtinInc})
	e.env.Set("dec", &Primitive{Name: "dec", Fn: e.builtinDec})

	e.env.Set("string-length", &Primitive{Name: "string-length", Fn: e.builtinStringLength})
	e.env.Set("string-ref", &Primitive{Name: "string-ref", Fn: e.builtinStringRef})
	e.env.Set("substring", &Primitive{Name: "substring", Fn: e.builtinSubstring})
	e.env.Set("string-append", &Primitive{Name: "string-append", Fn: e.builtinStringAppend})
	e.env.Set("string=?", &Primitive{Name: "string=?", Fn: e.builtinStringEq})
	e.env.Set("string-ci=?", &Primitive{Name: "string-ci=?", Fn: e.builtinStringCiEq})
	e.env.Set("string<?", &Primitive{Name: "string<?", Fn: e.builtinStringLt})
	e.env.Set("string>?", &Primitive{Name: "string>?", Fn: e.builtinStringGt})
	e.env.Set("string-downcase", &Primitive{Name: "string-downcase", Fn: e.builtinStringDowncase})
	e.env.Set("string-upcase", &Primitive{Name: "string-upcase", Fn: e.builtinStringUpcase})
	e.env.Set("string-trim", &Primitive{Name: "string-trim", Fn: e.builtinStringTrim})
	e.env.Set("string-split", &Primitive{Name: "string-split", Fn: e.builtinStringSplit})
	e.env.Set("string-join", &Primitive{Name: "string-join", Fn: e.builtinStringJoin})
	e.env.Set("number->string", &Primitive{Name: "number->string", Fn: e.builtinNumberToString})
	e.env.Set("string->number", &Primitive{Name: "string->number", Fn: e.builtinStringToNumber})
	e.env.Set("symbol->string", &Primitive{Name: "symbol->string", Fn: e.builtinSymbolToString})
	e.env.Set("string->symbol", &Primitive{Name: "string->symbol", Fn: e.builtinStringToSymbol})

	e.env.Set("display", &Primitive{Name: "display", Fn: e.builtinDisplay})
	e.env.Set("write", &Primitive{Name: "write", Fn: e.builtinWrite})
	e.env.Set("println", &Primitive{Name: "println", Fn: e.builtinPrintln})
	e.env.Set("print", &Primitive{Name: "print", Fn: e.builtinPrint})
	e.env.Set("newline", &Primitive{Name: "newline", Fn: e.builtinNewline})
	e.env.Set("read-line", &Primitive{Name: "read-line", Fn: e.builtinReadLine})

	e.env.Set("file->string", &Primitive{Name: "file->string", Fn: e.builtinFileToString})
	e.env.Set("string->file", &Primitive{Name: "string->file", Fn: e.builtinStringToFile})
	e.env.Set("file-exists?", &Primitive{Name: "file-exists?", Fn: e.builtinFileExists})
	e.env.Set("delete-file", &Primitive{Name: "delete-file", Fn: e.builtinDeleteFile})

	e.env.Set("vector", &Primitive{Name: "vector", Fn: e.builtinVector})
	e.env.Set("make-vector", &Primitive{Name: "make-vector", Fn: e.builtinMakeVector})
	e.env.Set("vector-ref", &Primitive{Name: "vector-ref", Fn: e.builtinVectorRef})
	e.env.Set("vector-set!", &Primitive{Name: "vector-set!", Fn: e.builtinVectorSet})
	e.env.Set("vector-length", &Primitive{Name: "vector-length", Fn: e.builtinVectorLength})
	e.env.Set("vector?", &Primitive{Name: "vector?", Fn: e.builtinIsVector})
	e.env.Set("vector->list", &Primitive{Name: "vector->list", Fn: e.builtinVectorToList})
	e.env.Set("list->vector", &Primitive{Name: "list->vector", Fn: e.builtinListToVector})
	e.env.Set("vector-fill!", &Primitive{Name: "vector-fill!", Fn: e.builtinVectorFill})
	e.env.Set("vector-map", &Primitive{Name: "vector-map", Fn: e.builtinVectorMap})

	e.env.Set("system", &Primitive{Name: "system", Fn: e.builtinSystem})
	e.env.Set("shell->string", &Primitive{Name: "shell->string", Fn: e.builtinShellToString})

	e.env.Set("sleep", &Primitive{Name: "sleep", Fn: e.builtinSleep})
	e.env.Set("usleep", &Primitive{Name: "usleep", Fn: e.builtinUsleep})
	e.env.Set("exit", &Primitive{Name: "exit", Fn: e.builtinExit})
	e.env.Set("get-file-dir", &Primitive{Name: "get-file-dir", Fn: e.builtinGetFileDir})

	// OOP
	e.env.Set("make-class", &Primitive{Name: "make-class", Fn: e.builtinMakeClass})
	e.env.Set("new", &Primitive{Name: "new", Fn: e.builtinNew})
	e.env.Set("send", &Primitive{Name: "send", Fn: e.builtinSend})
	e.env.Set("slot-ref", &Primitive{Name: "slot-ref", Fn: e.builtinSlotRef})
	e.env.Set("slot-set!", &Primitive{Name: "slot-set!", Fn: e.builtinSlotSet})
	e.env.Set("instance?", &Primitive{Name: "instance?", Fn: e.builtinInstanceOf})
	e.env.Set("class-of", &Primitive{Name: "class-of", Fn: e.builtinClassOf})
	e.env.Set("add-method", &Primitive{Name: "add-method", Fn: e.builtinAddMethod})

	// HTTP Server
	e.env.Set("http/create-server", &Primitive{Name: "http/create-server", Fn: e.builtinHttpCreateServer})
	e.env.Set("http/set-handler", &Primitive{Name: "http/set-handler", Fn: e.builtinHttpSetHandler})
	e.env.Set("http/start-server", &Primitive{Name: "http/start-server", Fn: e.builtinHttpStartServer})
	e.env.Set("http/request-method", &Primitive{Name: "http/request-method", Fn: e.builtinHttpRequestMethod})
	e.env.Set("http/request-path", &Primitive{Name: "http/request-path", Fn: e.builtinHttpRequestPath})
	e.env.Set("http/request-headers", &Primitive{Name: "http/request-headers", Fn: e.builtinHttpRequestHeaders})
	e.env.Set("http/request-body", &Primitive{Name: "http/request-body", Fn: e.builtinHttpRequestBody})
	e.env.Set("http/make-response", &Primitive{Name: "http/make-response", Fn: e.builtinHttpMakeResponse})
	e.env.Set("http/response-status", &Primitive{Name: "http/response-status", Fn: e.builtinHttpResponseStatus})
	e.env.Set("http/response-headers", &Primitive{Name: "http/response-headers", Fn: e.builtinHttpResponseHeaders})
	e.env.Set("http/response-body", &Primitive{Name: "http/response-body", Fn: e.builtinHttpResponseBody})
}
