package main

import (
	"testing"
)

func TestLexerNumbers(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("42 3.14 -7")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(toks))
	}
	checkToken(t, toks[0], TkNumber, Integer(42))
	checkToken(t, toks[1], TkNumber, Float(3.14))
	checkToken(t, toks[2], TkNumber, Integer(-7))
}

func TestLexerStrings(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize(`"hello" "a\nb"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(toks))
	}
	checkToken(t, toks[0], TkString, String("hello"))
}

func TestLexerBooleans(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("#t #f")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(toks))
	}
	checkToken(t, toks[0], TkBoolean, Boolean(true))
	checkToken(t, toks[1], TkBoolean, Boolean(false))
}

func TestLexerSymbols(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("foo bar? + <=>")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(toks))
	}
	checkTokenSym(t, toks[0], "foo")
	checkTokenSym(t, toks[1], "bar?")
	checkTokenSym(t, toks[2], "+")
	checkTokenSym(t, toks[3], "<=>")
}

func TestLexerParens(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("(a b)")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(toks))
	}
	if toks[0].Type != TkLParen || toks[3].Type != TkRParen {
		t.Fatalf("expected LPAREN and RPAREN")
	}
}

func TestLexerQuote(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("'x")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 2 || toks[0].Type != TkQuote {
		t.Fatalf("expected QUOTE + symbol, got %d tokens", len(toks))
	}
}

func TestLexerVector(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("#(1 2)")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) < 1 || toks[0].Type != TkVectorStart {
		t.Fatalf("expected TkVectorStart as first token, got %d tokens first type=%d", len(toks), toks[0].Type)
	}
}

func TestLexerComments(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("; comment\n42")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 1 {
		t.Fatalf("expected 1 token, got %d", len(toks))
	}
	checkToken(t, toks[0], TkNumber, Integer(42))
}

func TestLexerShebang(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("#!/usr/bin/env ll\n42")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 1 {
		t.Fatalf("expected 1 token after shebang, got %d", len(toks))
	}
	checkToken(t, toks[0], TkNumber, Integer(42))
}

func TestLexerLineTracking(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("a\nb\nc")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(toks))
	}
	if toks[0].Line != 1 {
		t.Errorf("expected 'a' on line 1, got line %d", toks[0].Line)
	}
	if toks[1].Line != 2 {
		t.Errorf("expected 'b' on line 2, got line %d", toks[1].Line)
	}
	if toks[2].Line != 3 {
		t.Errorf("expected 'c' on line 3, got line %d", toks[2].Line)
	}
}

func TestLexerLineTrackingParens(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("(\n  a\n)")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(toks))
	}
	if toks[0].Type != TkLParen || toks[0].Line != 1 {
		t.Errorf("expected '(' on line 1, got line %d", toks[0].Line)
	}
	if toks[1].Line != 2 {
		t.Errorf("expected 'a' on line 2, got line %d", toks[1].Line)
	}
	if toks[2].Type != TkRParen || toks[2].Line != 3 {
		t.Errorf("expected ')' on line 3, got line %d", toks[2].Line)
	}
}

func TestLexerLineTrackingShebang(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("#!/usr/bin/env ll\n42\n(+ 1 2)")
	if err != nil {
		t.Fatal(err)
	}
	// 42 (line 2), (, +, 1, 2, ) (line 3) = 6 tokens
	if len(toks) != 6 {
		t.Fatalf("expected 6 tokens after shebang, got %d", len(toks))
	}
	if toks[0].Line != 2 {
		t.Errorf("expected 42 on line 2, got line %d", toks[0].Line)
	}
	if toks[1].Line != 3 {
		t.Errorf("expected '(' on line 3, got line %d", toks[1].Line)
	}
}

func TestLexerLineTrackingComment(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("a\n; comment\nb")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(toks))
	}
	if toks[0].Line != 1 {
		t.Errorf("expected 'a' on line 1, got line %d", toks[0].Line)
	}
	if toks[1].Line != 3 {
		t.Errorf("expected 'b' on line 3, got line %d", toks[1].Line)
	}
}

func TestLexerLineTrackingMultilineString(t *testing.T) {
	l := &Lexer{}
	toks, err := l.Tokenize("\"hello\nworld\"")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 1 {
		t.Fatalf("expected 1 token, got %d", len(toks))
	}
	if toks[0].Line != 1 {
		t.Errorf("expected string starting on line 1, got line %d", toks[0].Line)
	}
	if toks[0].Value != String("hello\nworld") {
		t.Errorf("expected string value with newline, got %q", toks[0].Value)
	}
}

func TestLexerLineTrackingSubsequentCalls(t *testing.T) {
	l := &Lexer{}
	toks1, err := l.Tokenize("a\nb")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks1) != 2 || toks1[0].Line != 1 || toks1[1].Line != 2 {
		t.Fatalf("first call: expected lines 1,2; got %d,%d", toks1[0].Line, toks1[1].Line)
	}

	toks2, err := l.Tokenize("x\n\ny")
	if err != nil {
		t.Fatal(err)
	}
	if len(toks2) != 2 || toks2[0].Line != 1 || toks2[1].Line != 3 {
		t.Fatalf("second call: expected lines 1,3; got %d,%d", toks2[0].Line, toks2[1].Line)
	}
}

func checkToken(t *testing.T, tok Token, typ TokenType, val Value) {
	t.Helper()
	if tok.Type != typ {
		t.Errorf("expected type %d, got %d", typ, tok.Type)
	}
	if tok.Value != val {
		t.Errorf("expected value %v, got %v", val, tok.Value)
	}
}

func checkTokenSym(t *testing.T, tok Token, name string) {
	t.Helper()
	if tok.Type != TkSymbol {
		t.Fatalf("expected TkSymbol, got %d", tok.Type)
	}
	sym, ok := tok.Value.(*Sym)
	if !ok || sym.Name != name {
		t.Fatalf("expected symbol %q, got %v", name, tok.Value)
	}
}
