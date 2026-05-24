package main

import (
	"testing"
)

func TestParseNumber(t *testing.T) {
	p := &Parser{}
	ast, err := p.Parse(mkTokens(TkNumber, Integer(42)))
	if err != nil {
		t.Fatal(err)
	}
	if len(ast) != 1 {
		t.Fatalf("expected 1 expr, got %d", len(ast))
	}
	if ast[0] != Integer(42) {
		t.Fatalf("expected 42, got %v", ast[0])
	}
}

func TestParseList(t *testing.T) {
	p := &Parser{}
	ast, err := p.Parse(mkTokens(
		TkLParen, TkNumber, Integer(1), TkNumber, Integer(2), TkRParen,
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(ast) != 1 {
		t.Fatalf("expected 1 expr, got %d", len(ast))
	}
	cons, ok := ast[0].(*Cons)
	if !ok {
		t.Fatalf("expected Cons, got %T", ast[0])
	}
	if cons.Car != Integer(1) {
		t.Fatalf("expected car=1, got %v", cons.Car)
	}
	cdr := cons.Cdr.(*Cons)
	if cdr.Car != Integer(2) {
		t.Fatalf("expected cadr=2, got %v", cdr.Car)
	}
	if cdr.Cdr != Nil {
		t.Fatalf("expected cddr=Nil")
	}
}

func TestParseQuote(t *testing.T) {
	p := &Parser{}
	ast, err := p.Parse([]Token{
		{Type: TkQuote},
		{Type: TkSymbol, Value: &Sym{Name: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	cons, ok := ast[0].(*Cons)
	if !ok {
		t.Fatalf("expected Cons, got %T", ast[0])
	}
	sym, ok := cons.Car.(*Sym)
	if !ok || sym.Name != "quote" {
		t.Fatalf("expected (quote x), got car=%v", cons.Car)
	}
}

func TestParseVector(t *testing.T) {
	p := &Parser{}
	toks := []Token{
		{Type: TkVectorStart},
		{Type: TkNumber, Value: Integer(1)},
		{Type: TkNumber, Value: Integer(2)},
		{Type: TkRParen},
	}
	ast, err := p.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := ast[0].(*Vector)
	if !ok {
		t.Fatalf("expected Vector, got %T", ast[0])
	}
	if len(v.Items) != 2 || v.Items[0] != Integer(1) || v.Items[1] != Integer(2) {
		t.Fatalf("expected #(1 2), got %v", v)
	}
}

func TestParseNested(t *testing.T) {
	p := &Parser{}
	ast, err := p.Parse(mkTokens(
		TkLParen, TkSymbol, &Sym{Name: "a"},
		TkLParen, TkSymbol, &Sym{Name: "b"}, TkRParen,
		TkRParen,
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(ast) != 1 {
		t.Fatalf("expected 1 expr")
	}
}

func TestParseUnterminated(t *testing.T) {
	p := &Parser{}
	_, err := p.Parse(mkTokens(TkLParen, TkNumber, Integer(1)))
	if err == nil {
		t.Fatal("expected error for unterminated list")
	}
}

func mkTokens(items ...interface{}) []Token {
	var toks []Token
	for i := 0; i < len(items); i++ {
		switch v := items[i].(type) {
		case TokenType:
			tok := Token{Type: v}
			if i+1 < len(items) {
				if val, ok := items[i+1].(Value); ok {
					tok.Value = val
					i++
				}
			}
			toks = append(toks, tok)
		case int:
			toks = append(toks, Token{Value: Integer(v)})
		}
	}
	return toks
}
