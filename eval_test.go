package main

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
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

func TestEvalFuture(t *testing.T) {
	e := NewEval()
	v, err := e.Eval(parserExpr(t, "(future (+ 1 2))"))
	if err != nil {
		t.Fatal(err)
	}
	f, ok := v.(*Future)
	if !ok {
		t.Fatalf("expected Future, got %T", v)
	}
	result, err := f.Await()
	if err != nil {
		t.Fatal(err)
	}
	if result != Integer(3) {
		t.Fatalf("expected 3, got %v", result)
	}
}

func TestEvalFutureAwait(t *testing.T) {
	e := NewEval()
	v, err := e.Eval(parserExpr(t, "(await (future (+ 1 2)))"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(3) {
		t.Fatalf("expected 3, got %v", v)
	}
}

func TestEvalFutureAwaitMultiExpr(t *testing.T) {
	e := NewEval()
	v, err := e.Eval(parserExpr(t, "(await (future (define x 42) (+ x 1)))"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(43) {
		t.Fatalf("expected 43, got %v", v)
	}
}

func TestEvalFuturePredicate(t *testing.T) {
	e := NewEval()
	v, err := e.Eval(parserExpr(t, "(future? (future 1))"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Boolean(true) {
		t.Fatalf("expected #t, got %v", v)
	}

	v2 := evalOne(t, "(future? 42)")
	if v2 != Boolean(false) {
		t.Fatalf("expected #f, got %v", v2)
	}
}

func TestEvalCo(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define f (co () (+ 1 2)))"))
	v, err := e.Eval(parserExpr(t, "(await (f))"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(3) {
		t.Fatalf("expected 3, got %v", v)
	}
}

func TestEvalCoWithArgs(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define async-add (co (a b) (+ a b)))"))
	v, err := e.Eval(parserExpr(t, "(await (async-add 10 20))"))
	if err != nil {
		t.Fatal(err)
	}
	if v != Integer(30) {
		t.Fatalf("expected 30, got %v", v)
	}
}

func TestEvalCoDirect(t *testing.T) {
	e := NewEval()
	fn, err := e.Eval(parserExpr(t, "(co (x) (* x x))"))
	if err != nil {
		t.Fatal(err)
	}
	fnVal, ok := fn.(*Closure)
	if !ok {
		t.Fatalf("expected Closure, got %T", fn)
	}
	if !fnVal.isAsync {
		t.Fatal("expected async closure")
	}
}

func TestEvalParallelFutures(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, "(define slow-add (co (a b) (+ a b)))"))

	v1, _ := e.Eval(parserExpr(t, "(slow-add 10 20)"))
	v2, _ := e.Eval(parserExpr(t, "(slow-add 30 40)"))

	f1, ok := v1.(*Future)
	if !ok {
		t.Fatalf("expected Future, got %T", v1)
	}
	f2, ok := v2.(*Future)
	if !ok {
		t.Fatalf("expected Future, got %T", v2)
	}

	r1, _ := f1.Await()
	r2, _ := f2.Await()

	if r1 != Integer(30) {
		t.Fatalf("expected 30, got %v", r1)
	}
	if r2 != Integer(70) {
		t.Fatalf("expected 70, got %v", r2)
	}
}

func cons(a, b Value) *Cons {
	return &Cons{Car: a, Cdr: b}
}

// --- OOP tests ---

func TestOopMakeClass(t *testing.T) {
	e := NewEval()
	v, err := e.Eval(parserExpr(t, `(make-class 'Animal () '((name "unknown") (age 0)))`))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := v.(*ClassType)
	if !ok {
		t.Fatalf("expected ClassType, got %T", v)
	}
	if c.Name != "Animal" {
		t.Fatalf("expected name Animal, got %s", c.Name)
	}
	if len(c.Slots) != 2 || c.Slots[0].Name != "name" || c.Slots[1].Name != "age" {
		t.Fatalf("unexpected slots: %+v", c.Slots)
	}
}

func TestOopMakeClassInherited(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown") (age 0))))`))
	v, err := e.Eval(parserExpr(t, `(make-class 'Dog Animal '((breed "mixed")))`))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := v.(*ClassType)
	if !ok {
		t.Fatalf("expected ClassType, got %T", v)
	}
	if c.Name != "Dog" {
		t.Fatalf("expected name Dog, got %s", c.Name)
	}
	if len(c.Slots) != 3 {
		t.Fatalf("expected 3 slots (2 inherited + 1 own), got %d", len(c.Slots))
	}
	if c.Slots[0].Name != "name" || c.Slots[1].Name != "age" || c.Slots[2].Name != "breed" {
		t.Fatalf("unexpected slots: %+v", c.Slots)
	}
	if c.Parent == nil || c.Parent.Name != "Animal" {
		t.Fatalf("parent should be Animal")
	}
}

func TestOopNew(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown") (age 0))))`))
	v, err := e.Eval(parserExpr(t, `(new Animal 'name "Rex" 'age 5)`))
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := v.(*Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", v)
	}
	if inst.Class.Name != "Animal" {
		t.Fatalf("expected class Animal, got %s", inst.Class.Name)
	}
	if len(inst.Data) != 2 {
		t.Fatalf("expected 2 slot values, got %d", len(inst.Data))
	}
	if inst.Data[0] != String("Rex") || inst.Data[1] != Integer(5) {
		t.Fatalf("unexpected slot values: %v", inst.Data)
	}
}

func TestOopNewDefaults(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown") (age 0))))`))
	v, err := e.Eval(parserExpr(t, `(new Animal)`))
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := v.(*Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", v)
	}
	if inst.Data[0] != String("unknown") || inst.Data[1] != Integer(0) {
		t.Fatalf("expected defaults, got %v", inst.Data)
	}
}

func TestOopSlotRef(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown") (age 0))))`))
	v, err := e.Eval(parserExpr(t, `(slot-ref (new Animal 'name "Rex") 'name)`))
	if err != nil {
		t.Fatal(err)
	}
	if v != String("Rex") {
		t.Fatalf("expected Rex, got %v", v)
	}
}

func TestOopSlotSet(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown") (age 0))))`))
	e.Eval(parserExpr(t, `(define a (new Animal 'name "Rex"))`))
	_, err := e.Eval(parserExpr(t, `(slot-set! a 'name "Buddy")`))
	if err != nil {
		t.Fatal(err)
	}
	v, _ := e.Eval(parserExpr(t, `(slot-ref a 'name)`))
	if v != String("Buddy") {
		t.Fatalf("expected Buddy, got %v", v)
	}
}

func TestOopInstanceOf(t *testing.T) {
	v := evalOne(t, `(instance? 42)`)
	if v != Boolean(false) {
		t.Fatalf("expected #f, got %v", v)
	}
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '()))`))
	v, _ = e.Eval(parserExpr(t, `(instance? (new Animal))`))
	if v != Boolean(true) {
		t.Fatalf("expected #t, got %v", v)
	}
}

func TestOopClassOf(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '()))`))
	v, err := e.Eval(parserExpr(t, `(class-of (new Animal))`))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := v.(*ClassType)
	if !ok {
		t.Fatalf("expected ClassType, got %T", v)
	}
	if c.Name != "Animal" {
		t.Fatalf("expected Animal, got %s", c.Name)
	}
}

func TestOopAddMethodAndSend(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown"))))`))
	_, err := e.Eval(parserExpr(t, `(add-method Animal 'speak (lambda (self) (slot-ref self 'name)))`))
	if err != nil {
		t.Fatal(err)
	}
	v, err := e.Eval(parserExpr(t, `(send (new Animal 'name "Rex") 'speak)`))
	if err != nil {
		t.Fatal(err)
	}
	if v != String("Rex") {
		t.Fatalf("expected Rex, got %v", v)
	}
}

func TestOopInheritedMethod(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown"))))`))
	e.Eval(parserExpr(t, `(add-method Animal 'speak (lambda (self) (slot-ref self 'name)))`))
	e.Eval(parserExpr(t, `(define Dog (make-class 'Dog Animal '()))`))

	v, err := e.Eval(parserExpr(t, `(send (new Dog 'name "Max") 'speak)`))
	if err != nil {
		t.Fatal(err)
	}
	if v != String("Max") {
		t.Fatalf("expected Max, got %v", v)
	}
}

func TestOopMethodOverride(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown"))))`))
	e.Eval(parserExpr(t, `(add-method Animal 'speak (lambda (self) "animal"))`))
	e.Eval(parserExpr(t, `(define Dog (make-class 'Dog Animal '()))`))
	e.Eval(parserExpr(t, `(add-method Dog 'speak (lambda (self) "dog"))`))

	v, err := e.Eval(parserExpr(t, `(send (new Dog) 'speak)`))
	if err != nil {
		t.Fatal(err)
	}
	if v != String("dog") {
		t.Fatalf("expected dog, got %v", v)
	}

	// Parent method unchanged
	v2, err := e.Eval(parserExpr(t, `(send (new Animal) 'speak)`))
	if err != nil {
		t.Fatal(err)
	}
	if v2 != String("animal") {
		t.Fatalf("expected animal, got %v", v2)
	}
}

func TestOopSendWithArgs(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown"))))`))
	e.Eval(parserExpr(t, `(add-method Animal 'greet (lambda (self other) (list (slot-ref self 'name) other)))`))

	v, err := e.Eval(parserExpr(t, `(send (new Animal 'name "Alice") 'greet "Bob")`))
	if err != nil {
		t.Fatal(err)
	}
	sl, ok := ListToSlice(v)
	if !ok || len(sl) != 2 || sl[0] != String("Alice") || sl[1] != String("Bob") {
		t.Fatalf("expected (Alice Bob), got %v", v)
	}
}

func TestOopNewInvalidSlot(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '((name "unknown"))))`))
	_, err := e.Eval(parserExpr(t, `(new Animal 'nonexistent 42)`))
	if err == nil {
		t.Fatal("expected error for unknown slot")
	}
}

func TestOopSendNoMethod(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '()))`))
	_, err := e.Eval(parserExpr(t, `(send (new Animal) 'missing)`))
	if err == nil {
		t.Fatal("expected error for missing method")
	}
}

func TestOopSlotRefInvalidSlot(t *testing.T) {
	e := NewEval()
	e.Eval(parserExpr(t, `(define Animal (make-class 'Animal () '()))`))
	_, err := e.Eval(parserExpr(t, `(slot-ref (new Animal) 'missing)`))
	if err == nil {
		t.Fatal("expected error for missing slot")
	}
}

func TestHttpCreateServer(t *testing.T) {
	e := NewEval()
	err := e.EvalString(`(define s (http/create-server "0.0.0.0" 8080))`)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := e.Env().Get("s")
	srv, ok := v.(*HttpServer)
	if !ok {
		t.Fatalf("expected *HttpServer, got %T", v)
	}
	if srv.Host != "0.0.0.0" || srv.Port != 8080 {
		t.Fatalf("expected 0.0.0.0:8080, got %s:%d", srv.Host, srv.Port)
	}
	if srv.Handler != Nil {
		t.Fatal("expected nil handler initially")
	}
}

func TestHttpCreateServerErrors(t *testing.T) {
	e := NewEval()
	_, err := e.Eval(parserExpr(t, `(http/create-server "localhost")`))
	if err == nil {
		t.Fatal("expected error for missing port")
	}
	_, err = e.Eval(parserExpr(t, `(http/create-server 123 8080)`))
	if err == nil {
		t.Fatal("expected error for non-string host")
	}
	_, err = e.Eval(parserExpr(t, `(http/create-server "localhost" "8080")`))
	if err == nil {
		t.Fatal("expected error for non-integer port")
	}
}

func TestHttpSetHandler(t *testing.T) {
	e := NewEval()
	err := e.EvalString(`
		(define s (http/create-server "localhost" 0))
		(http/set-handler s (lambda (req) (http/make-response 200 '() "ok")))
	`)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := e.Env().Get("s")
	srv := v.(*HttpServer)
	if srv.Handler == nil {
		t.Fatal("handler should be set")
	}
}

func TestHttpSetHandlerErrors(t *testing.T) {
	e := NewEval()
	_, err := e.Eval(parserExpr(t, `(http/set-handler "not-a-server" (lambda (req) 42))`))
	if err == nil {
		t.Fatal("expected error for non-server argument")
	}
	e.EvalString(`(define s (http/create-server "localhost" 0))`)
	_, err = e.Eval(parserExpr(t, `(http/set-handler s "not-a-function")`))
	if err == nil {
		t.Fatal("expected error for non-function handler")
	}
}

func TestHttpMakeResponse(t *testing.T) {
	e := NewEval()
	err := e.EvalString(`
		(define r (http/make-response 200
			'(("Content-Type" . "text/plain") ("X-Custom" . "val"))
			"hello world"))
	`)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := e.Env().Get("r")
	resp, ok := v.(*HttpResponse)
	if !ok {
		t.Fatalf("expected *HttpResponse, got %T", v)
	}
	if resp.Status != 200 {
		t.Fatalf("expected status 200, got %d", resp.Status)
	}
	if resp.Body != "hello world" {
		t.Fatalf("expected body 'hello world', got %q", resp.Body)
	}
	if resp.Headers["Content-Type"] != "text/plain" {
		t.Fatalf("expected Content-Type header, got %v", resp.Headers)
	}
	if resp.Headers["X-Custom"] != "val" {
		t.Fatalf("expected X-Custom header, got %v", resp.Headers)
	}
}

func TestHttpMakeResponseErrors(t *testing.T) {
	e := NewEval()
	_, err := e.Eval(parserExpr(t, `(http/make-response "200" '() "body")`))
	if err == nil {
		t.Fatal("expected error for non-integer status")
	}
	_, err = e.Eval(parserExpr(t, `(http/make-response 200 '() 123)`))
	if err == nil {
		t.Fatal("expected error for non-string body")
	}
	_, err = e.Eval(parserExpr(t, `(http/make-response 200 '(("K" . "V")) "b" "extra")`))
	if err == nil {
		t.Fatal("expected error for extra argument")
	}
}

func TestHttpResponseAccessors(t *testing.T) {
	e := NewEval()
	err := e.EvalString(`
		(define r (http/make-response 201 '(("Content-Type" . "text/html")) "<h1>hi</h1>"))
		(define st (http/response-status r))
		(define hdrs (http/response-headers r))
		(define bd (http/response-body r))
	`)
	if err != nil {
		t.Fatal(err)
	}

	v, _ := e.Env().Get("st")
	if v != Integer(201) {
		t.Fatalf("expected status 201, got %v", v)
	}

	v, _ = e.Env().Get("bd")
	if v != String("<h1>hi</h1>") {
		t.Fatalf("expected body '<h1>hi</h1>', got %v", v)
	}

	v, _ = e.Env().Get("hdrs")
	if v == Nil {
		t.Fatal("headers should not be nil")
	}
}

func TestHttpServerEndToEnd(t *testing.T) {
	e := NewEval()

	err := e.EvalString(`
		(define s (http/create-server "localhost" 9997))
		(http/set-handler s (lambda (req)
			(define m (http/request-method req))
			(define p (http/request-path req))
			(define b (http/request-body req))
			(http/make-response 200
				'(("Content-Type" . "text/plain"))
				(string-append m " " p " [" b "]"))))
	`)
	if err != nil {
		t.Fatal(err)
	}

	go e.EvalString(`(http/start-server s)`)
	time.Sleep(200 * time.Millisecond)

	// Test GET
	resp, err := http.Get("http://localhost:9997/hello")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "GET /hello []" {
		t.Fatalf("GET: unexpected response %q", body)
	}

	// Test POST with body
	resp, err = http.Post("http://localhost:9997/test", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "POST /test []" {
		t.Fatalf("POST: unexpected response %q", body)
	}
}

func TestHttpRequestAccessors(t *testing.T) {
	e := NewEval()

	err := e.EvalString(`
		(define s (http/create-server "localhost" 9996))
		(http/set-handler s (lambda (req)
			(define headers (http/request-headers req))
			(define ua (if (null? headers) "none"
				(let ((pair (car headers)))
					(cdr pair))))
			(http/make-response 200 '(("Content-Type" . "text/plain")) ua)))
	`)
	if err != nil {
		t.Fatal(err)
	}

	go e.EvalString(`(http/start-server s)`)
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get("http://localhost:9996/check")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if len(body) == 0 {
		t.Fatal("expected non-empty User-Agent in response")
	}
}

func TestJsonEncode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"(json/encode nil)", "null"},
		{"(json/encode #t)", "true"},
		{"(json/encode #f)", "false"},
		{"(json/encode 42)", "42"},
		{"(json/encode 3.14)", "3.14"},
		{`(json/encode "hello")`, `"hello"`},
		{`(json/encode "a\"b")`, `"a\"b"`},
		{"(json/encode (vector 1 2 3))", "[1,2,3]"},
		{`(json/encode (list "a" "b"))`, `["a","b"]`},
		{`(json/encode (list (cons "k" 1) (cons "v" 2)))`, `{"k":1,"v":2}`},
	}
	for _, tt := range tests {
		e := NewEval()
		got, err := evalStringResult(t, e, tt.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tt.input, err)
			continue
		}
		s, ok := got.(String)
		if !ok {
			t.Errorf("expected String, got %T for %q", got, tt.input)
			continue
		}
		if string(s) != tt.want {
			t.Errorf("json/encode %q: got %q, want %q", tt.input, string(s), tt.want)
		}
	}
}

func TestJsonDecode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`(json/decode "null")`, "()"},
		{`(json/decode "true")`, "#t"},
		{`(json/decode "false")`, "#f"},
		{`(json/decode "42")`, "42"},
		{`(json/decode "3.14")`, "3.14"},
		{`(json/decode "\"hello\"")`, "hello"},
		{`(json/decode "[1,2,3]")`, "#(1 2 3)"},
		{`(json/decode "{\"a\":1}")`, "((a . 1))"},
	}
	for _, tt := range tests {
		e := NewEval()
		got, err := evalStringResult(t, e, tt.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tt.input, err)
			continue
		}
		s := FormatValue(got)
		if s != tt.want {
			t.Errorf("json/decode %q: got %q, want %q", tt.input, s, tt.want)
		}
	}
}

func TestJsonRoundtrip(t *testing.T) {
	e := NewEval()
	// encode -> decode roundtrip (keys sorted alphabetically)
	expr := `(json/decode (json/encode (list (cons "name" "test") (cons "count" 42))))`
	got, err := evalStringResult(t, e, expr)
	if err != nil {
		t.Fatal(err)
	}
	cons, ok := got.(*Cons)
	if !ok {
		t.Fatalf("expected Cons, got %T", got)
	}
	// after roundtrip, keys are sorted: "count" before "name"
	countPair := cons.Car.(*Cons)
	if string(countPair.Car.(String)) != "count" {
		t.Errorf("expected first key 'count', got %q", countPair.Car)
	}
	countVal, ok := countPair.Cdr.(Integer)
	if !ok || int64(countVal) != 42 {
		t.Errorf("expected count=42, got %v", countPair.Cdr)
	}
	namePair := cons.Cdr.(*Cons).Car.(*Cons)
	if string(namePair.Car.(String)) != "name" || string(namePair.Cdr.(String)) != "test" {
		t.Errorf("expected (name . \"test\"), got %v", namePair)
	}
}

func TestJsonEncodeErrors(t *testing.T) {
	e := NewEval()
	_, err := evalStringResult(t, e, `(json/encode)`)
	if err == nil {
		t.Error("expected error for (json/encode) with no args")
	}

	_, err = evalStringResult(t, e, `(json/encode 1 2)`)
	if err == nil {
		t.Error("expected error for (json/encode) with extra args")
	}
}

func TestJsonDecodeErrors(t *testing.T) {
	e := NewEval()
	_, err := evalStringResult(t, e, `(json/decode)`)
	if err == nil {
		t.Error("expected error for (json/decode) with no args")
	}

	_, err = evalStringResult(t, e, `(json/decode 42)`)
	if err == nil {
		t.Error("expected error for (json/decode) with non-string")
	}

	_, err = evalStringResult(t, e, `(json/decode "invalid{")`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestErrRuntimeFormat(t *testing.T) {
	err := &ErrRuntime{File: "test.ll", Line: 5, Msg: "undefined variable: x"}
	want := "test.ll:5: undefined variable: x"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}

	err2 := &ErrRuntime{Line: 3, Msg: "syntax error"}
	want2 := "line 3: syntax error"
	if err2.Error() != want2 {
		t.Errorf("got %q, want %q", err2.Error(), want2)
	}

	err3 := &ErrRuntime{Msg: "bare error"}
	if err3.Error() != "bare error" {
		t.Errorf("got %q, want %q", err3.Error(), "bare error")
	}
}

func TestErrorWithFileLine(t *testing.T) {
	e := NewEval()
	e.SetCurrentFile("test.ll")

	err := e.EvalString("(/ 1 0)")
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if errStr != "test.ll:1: division by zero" {
		t.Errorf("unexpected error format: %q", errStr)
	}

	e2 := NewEval()
	err = e2.EvalString("(undefined-var 42)")
	if err == nil {
		t.Fatal("expected error")
	}
	errStr = err.Error()
	if errStr != "line 1: undefined variable: undefined-var" {
		t.Errorf("unexpected error format: %q", errStr)
	}
}

func TestErrorWithFileParserError(t *testing.T) {
	e := NewEval()
	e.SetCurrentFile("main.ll")

	// parser error with line
	err := e.EvalString("(\n  \n  )\n)")
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	// the extra ')' is on line 4
	if errStr != "main.ll:4: unexpected ')'" {
		t.Errorf("unexpected error format: %q", errStr)
	}
}

func TestErrorWithFileLexerError(t *testing.T) {
	e := NewEval()
	e.SetCurrentFile("script.ll")

	// unterminated string
	err := e.EvalString("\"hello\n")
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if errStr != "script.ll:1: unterminated string" {
		t.Errorf("unexpected error format: %q", errStr)
	}
}

// evalStringResult evaluates a single expression and returns the result
func evalStringResult(t *testing.T, e *Eval, input string) (Value, error) {
	t.Helper()
	tokens, err := e.lexer.Tokenize(input)
	if err != nil {
		return nil, err
	}
	ast, err := e.parser.Parse(tokens)
	if err != nil {
		return nil, err
	}
	if len(ast) == 0 {
		return nil, fmt.Errorf("empty ast")
	}
	return e.Eval(ast[0])
}
