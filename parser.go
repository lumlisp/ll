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
		return nil, 0, fmt.Errorf("unexpected end of input")
	}

	tok := tokens[i]

	switch tok.Type {
	case TkLParen:
		return p.parseList(tokens, i+1)
	case TkRParen:
		return nil, 0, fmt.Errorf("unexpected ')'")
	case TkQuote:
		expr, next, err := p.parseExpr(tokens, i+1)
		if err != nil {
			return nil, 0, err
		}
		return &Cons{Car: &Sym{Name: "quote"}, Cdr: &Cons{Car: expr, Cdr: Nil}}, next, nil
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
		return nil, 0, fmt.Errorf("unknown token: %v", tok)
	}
}

func (p *Parser) parseVector(tokens []Token, i int) (Value, int, error) {
	var items []Value
	for i < len(tokens) && tokens[i].Type != TkRParen {
		expr, next, err := p.parseExpr(tokens, i)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, expr)
		i = next
	}
	if i >= len(tokens) {
		return nil, 0, fmt.Errorf("unterminated vector")
	}
	return &Vector{Items: items}, i + 1, nil
}

func (p *Parser) parseList(tokens []Token, i int) (Value, int, error) {
	var items []Value
	for i < len(tokens) && tokens[i].Type != TkRParen {
		expr, next, err := p.parseExpr(tokens, i)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, expr)
		i = next
	}
	if i >= len(tokens) {
		return nil, 0, fmt.Errorf("unterminated list")
	}
	return SliceToList(items), i + 1, nil
}
