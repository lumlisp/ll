package main

import "fmt"

type Parser struct{}

func (p *Parser) Parse(tokens []Token) ([]Value, error) {
	var ast []Value
	i := 0
	for i < len(tokens) {
		expr, next, err := p.parseExpr(tokens, i)
		if err != nil {
			return nil, err
		}
		ast = append(ast, expr)
		i = next
	}
	return ast, nil
}

func (p *Parser) parseExpr(tokens []Token, i int) (Value, int, error) {
	if i >= len(tokens) {
		if len(tokens) > 0 {
			return nil, 0, fmt.Errorf("line %d: unexpected end of input", tokens[len(tokens)-1].Line)
		}
		return nil, 0, fmt.Errorf("unexpected end of input")
	}

	tok := tokens[i]

	switch tok.Type {
	case TkLParen:
		return p.parseList(tokens, i+1)
	case TkRParen:
		return nil, 0, fmt.Errorf("line %d: unexpected ')'", tok.Line)
	case TkQuote:
		expr, next, err := p.parseExpr(tokens, i+1)
		if err != nil {
			return nil, 0, err
		}
		cons := &Cons{Car: &Sym{Name: "quote", Line: tok.Line}, Cdr: &Cons{Car: expr, Cdr: Nil, Line: tok.Line}, Line: tok.Line}
		return cons, next, nil
	case TkNumber:
		return tok.Value, i + 1, nil
	case TkString:
		return tok.Value, i + 1, nil
	case TkBoolean:
		return tok.Value, i + 1, nil
	case TkVectorStart:
		return p.parseVector(tokens, i+1)
	case TkSymbol:
		return tok.Value, i + 1, nil
	default:
		return nil, 0, fmt.Errorf("line %d: unknown token: %v", tok.Line, tok)
	}
}

func (p *Parser) parseVector(tokens []Token, i int) (Value, int, error) {
	var items []Value
	startLine := tokens[i-1].Line
	for i < len(tokens) && tokens[i].Type != TkRParen {
		expr, next, err := p.parseExpr(tokens, i)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, expr)
		i = next
	}
	if i >= len(tokens) {
		return nil, 0, fmt.Errorf("line %d: unterminated vector", startLine)
	}
	return &Vector{Items: items}, i + 1, nil
}

func (p *Parser) parseList(tokens []Token, i int) (Value, int, error) {
	var items []Value
	line := tokens[i-1].Line
	for i < len(tokens) && tokens[i].Type != TkRParen {
		expr, next, err := p.parseExpr(tokens, i)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, expr)
		i = next
	}
	if i >= len(tokens) {
		return nil, 0, fmt.Errorf("line %d: unterminated list", line)
	}

	dotIdx := -1
	for j, item := range items {
		if sym, ok := item.(*Sym); ok && sym.Name == "." && j > 0 {
			dotIdx = j
			break
		}
	}

	if dotIdx >= 0 {
		if dotIdx+1 >= len(items) {
			return nil, 0, fmt.Errorf("line %d: dotted pair: missing element after '.'", line)
		}
		if dotIdx+2 < len(items) {
			return nil, 0, fmt.Errorf("line %d: dotted pair: expected exactly one element after '.'", line)
		}
		result := SliceToList(items[:dotIdx])
		if cons, ok := result.(*Cons); ok && cons.Line == 0 {
			cons.Line = line
		}
		for c := result.(*Cons); ; c = c.Cdr.(*Cons) {
			if c.Cdr == Nil {
				c.Cdr = items[dotIdx+1]
				break
			}
		}
		return result, i + 1, nil
	}

	result := SliceToList(items)
	if cons, ok := result.(*Cons); ok && cons.Line == 0 {
		cons.Line = line
	}
	return result, i + 1, nil
}
