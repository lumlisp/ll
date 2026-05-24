package main

import (
	"testing"
)

func parserExpr(t *testing.T, input string) Value {
	t.Helper()
	l := &Lexer{}
	p := &Parser{}
	toks, err := l.Tokenize(input)
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}
	ast, err := p.Parse(toks)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(ast) == 0 {
		t.Fatal("empty ast")
	}
	return ast[0]
}

func evalOne(t *testing.T, input string) Value {
	t.Helper()
	e := NewEval()
	expr := parserExpr(t, input)
	v, err := e.Eval(expr)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	return v
}

func TestEvalSelfEvaluating(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{"42", Integer(42)},
		{"3.14", Float(3.14)},
		{`"hello"`, String("hello")},
		{"#t", Boolean(true)},
		{"#f", Boolean(false)},
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if got != tt.want {
			t.Errorf("eval %q = %v (%T), want %v (%T)", tt.input, got, got, tt.want, tt.want)
		}
	}
}

func TestEvalDefine(t *testing.T) {
	e := NewEval()
	expr := parserExpr(t, "(define x 42)")
	_, err := e.Eval(expr)
	if err != nil {
		t.Fatal(err)
	}
	v, err := e.Env().Get("x")
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(42) {
		t.Fatalf("expected x=42, got %v", v)
	}
}

func TestEvalSet(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define x 10)"))
	e.Eval(parserExpr(t, "(set! x 99)"))
	v, _ := e.Env().Get("x")
	if v != Integer(99) {
		t.Fatalf("expected x=99, got %v", v)
	}
}

func TestEvalArithmetic(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{"(+ 1 2)", Integer(3)},
		{"(- 10 3)", Integer(7)},
		{"(* 4 5)", Integer(20)},
		{"(/ 15 3)", Integer(5)},
		{"(% 17 5)", Integer(2)},
		{"(+ 1.5 2.5)", Float(4.0)},
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if got != tt.want {
			t.Errorf("eval %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEvalComparison(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{"(= 5 5)", Boolean(true)},
		{"(= 5 6)", Boolean(false)},
		{"(> 5 3)", Boolean(true)},
		{"(< 3 5)", Boolean(true)},
		{"(>= 5 5)", Boolean(true)},
		{"(<= 5 5)", Boolean(true)},
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if got != tt.want {
			t.Errorf("eval %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEvalListOps(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{"(car (quote (1 2 3)))", Integer(1)},
		{"(cdr (quote (1 2 3)))", cons(Integer(2), cons(Integer(3), Nil))},
		{"(cons 1 (quote (2 3)))", cons(Integer(1), cons(Integer(2), cons(Integer(3), Nil)))},
		{"(null? (quote ()))", Boolean(true)},
		{"(null? (quote (1)))", Boolean(false)},
		{"(pair? (quote (1)))", Boolean(true)},
		{"(pair? (quote ()))", Boolean(false)},
		{"(length (quote (1 2 3)))", Integer(3)},
		{"(length (quote ()))", Integer(0)},
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if !equalValue(got, tt.want) {
			t.Errorf("eval %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEvalIf(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{"(if #t 1 2)", Integer(1)},
		{"(if #f 1 2)", Integer(2)},
		{"(if 0 1 2)", Integer(1)}, // 0 is truthy
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if got != tt.want {
			t.Errorf("eval %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEvalLambda(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define (square x) (* x x))"))
	v, err := e.Eval(parserExpr(t, "(square 5)"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(25) {
		t.Fatalf("expected 25, got %v", v)
	}
}

func TestEvalClosure(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define (make-adder n) (lambda (x) (+ x n)))"))
	e.Eval(parserExpr(t, "(define add5 (make-adder 5))"))
	v, err := e.Eval(parserExpr(t, "(add5 10)"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(15) {
		t.Fatalf("expected 15, got %v", v)
	}
}

func TestEvalRecursion(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define (fib n) (if (<= n 1) n (+ (fib (- n 1)) (fib (- n 2)))))"))
	v, err := e.Eval(parserExpr(t, "(fib 10)"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(55) {
		t.Fatalf("expected 55, got %v", v)
	}
}

func TestEvalCond(t *testing.T) {
	e := NewEval()
	input := `(cond ((= 1 2) 'a) ((= 2 2) 'b) (else 'c))`
	v, err := e.Eval(parserExpr(t, input))
	if err != nil {
		t.Fatal(err)
	}
	sym, ok := v.(*Sym)
	if !ok || sym.Name != "b" {
		t.Fatalf("expected 'b', got %v", v)
	}
}

func TestEvalWhile(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define x 0)"))
	e.Eval(parserExpr(t, "(while (< x 5) (set! x (+ x 1)))"))
	v, _ := e.Env().Get("x")
	if v != Integer(5) {
		t.Fatalf("expected x=5, got %v", v)
	}
}

func TestEvalFor(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define s 0)"))
	e.Eval(parserExpr(t, "(for i 0 5 (set! s (+ s i)))"))
	v, _ := e.Env().Get("s")
	if v != Integer(10) { // 0+1+2+3+4 = 10
		t.Fatalf("expected s=10, got %v", v)
	}
}

func TestEvalAndOr(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{"(and #t #t)", Boolean(true)},
		{"(and #t #f)", Boolean(false)},
		{"(and)", Boolean(true)},
		{"(or #f #t)", Boolean(true)},
		{"(or #f #f)", Boolean(false)},
		{"(or)", Boolean(false)},
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if got != tt.want {
			t.Errorf("eval %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEvalQuote(t *testing.T) {
	v := evalOne(t, "'(1 2 3)")
	_, ok := v.(*Cons)
	if !ok {
		t.Fatalf("expected Cons, got %T", v)
	}
}

func TestEvalBegin(t *testing.T) {
	v := evalOne(t, "(begin (define x 1) (define y 2) (+ x y))")
	if v != Integer(3) {
		t.Fatalf("expected 3, got %v", v)
	}
}

func TestEvalMap(t *testing.T) {
	v := evalOne(t, "(map (lambda (x) (* x 2)) (quote (1 2 3)))")
	sl, ok := ListToSlice(v)
	if !ok || len(sl) != 3 || sl[0] != Integer(2) || sl[1] != Integer(4) || sl[2] != Integer(6) {
		t.Fatalf("expected (2 4 6), got %v", v)
	}
}

func TestEvalFilter(t *testing.T) {
	v := evalOne(t, "(filter (lambda (x) (even? x)) (quote (1 2 3 4 5)))")
	sl, _ := ListToSlice(v)
	if len(sl) != 2 || sl[0] != Integer(2) || sl[1] != Integer(4) {
		t.Fatalf("expected (2 4), got %v", v)
	}
}

func TestEvalFoldl(t *testing.T) {
	v := evalOne(t, "(foldl + 0 (quote (1 2 3 4 5)))")
	if v != Integer(15) {
		t.Fatalf("expected 15, got %v", v)
	}
}

func TestEvalVector(t *testing.T) {
	v := evalOne(t, "#(10 20 30)")
	vec, ok := v.(*Vector)
	if !ok || len(vec.Items) != 3 {
		t.Fatalf("expected Vector(3), got %T %v", v, v)
	}
}

func TestEvalVectorRef(t *testing.T) {
	v := evalOne(t, "(vector-ref #(10 20 30) 1)")
	if v != Integer(20) {
		t.Fatalf("expected 20, got %v", v)
	}
}

func TestEvalVectorSet(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define v #(1 2 3))"))
	e.Eval(parserExpr(t, "(vector-set! v 1 99)"))
	v, _ := e.Env().Get("v")
	vec := v.(*Vector)
	if vec.Items[1] != Integer(99) {
		t.Fatalf("expected 99 at index 1, got %v", vec.Items[1])
	}
}

func TestEvalMacro(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define-macro (unless cond body) (list (quote if) (list (quote not) cond) body))"))
	v, err := e.Eval(parserExpr(t, "(unless #f 42)"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(42) {
		t.Fatalf("expected 42, got %v", v)
	}
}

func TestEvalStringOps(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{`(string-length "hello")`, Integer(5)},
		{`(string-append "a" "b")`, String("ab")},
		{`(string=? "abc" "abc")`, Boolean(true)},
		{`(string-upcase "hello")`, String("HELLO")},
		{`(string-downcase "HELLO")`, String("hello")},
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if got != tt.want {
			t.Errorf("eval %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEvalPredicates(t *testing.T) {
	tests := []struct {
		input string
		want  Value
	}{
		{"(symbol? 'x)", Boolean(true)},
		{"(number? 42)", Boolean(true)},
		{`(string? "hi")`, Boolean(true)},
		{"(boolean? #t)", Boolean(true)},
		{"(list? '(1 2))", Boolean(true)},
		{"(null? (quote ()))", Boolean(true)},
		{"(fn? +)", Boolean(true)},
		{"(zero? 0)", Boolean(true)},
		{"(even? 4)", Boolean(true)},
		{"(odd? 7)", Boolean(true)},
		{"(equal? 5 5)", Boolean(true)},
		{"(equal? '(1 2) '(1 2))", Boolean(true)},
	}
	for _, tt := range tests {
		got := evalOne(t, tt.input)
		if got != tt.want {
			t.Errorf("eval %q = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func cons(a, b Value) *Cons {
	return &Cons{Car: a, Cdr: b}
}
