package main

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	TkLParen TokenType = iota
	TkRParen
	TkQuote
	TkNumber
	TkString
	TkBoolean
	TkSymbol
	TkVectorStart
)

type Token struct {
	Type  TokenType
	Value Value
}

type Lexer struct {
	input []rune
	pos   int
	len   int
}

func (l *Lexer) Tokenize(input string) ([]Token, error) {
	l.input = []rune(input)
	l.pos = 0
	l.len = len(l.input)
	var tokens []Token

	// skip shebang (#!) on first line
	if l.len > 1 && l.input[0] == '#' && l.input[1] == '!' {
		for l.pos < l.len && l.input[l.pos] != '\n' {
			l.pos++
		}
		if l.pos < l.len {
			l.pos++ // skip the \n
		}
	}

	for l.pos < l.len {
		ch := l.input[l.pos]

		if ch == ';' {
			for l.pos < l.len && l.input[l.pos] != '\n' {
				l.pos++
			}
			continue
		}

		if unicode.IsSpace(ch) {
			l.pos++
			continue
		}

		switch ch {
		case '(':
			tokens = append(tokens, Token{Type: TkLParen})
			l.pos++
		case ')':
			tokens = append(tokens, Token{Type: TkRParen})
			l.pos++
		case '\'':
			tokens = append(tokens, Token{Type: TkQuote})
			l.pos++
		case '"':
			tok, err := l.readString()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case '#':
			if l.pos+1 < l.len {
				next := l.input[l.pos+1]
				if next == 't' {
					tokens = append(tokens, Token{Type: TkBoolean, Value: Boolean(true)})
					l.pos += 2
					continue
				}
				if next == 'f' {
					tokens = append(tokens, Token{Type: TkBoolean, Value: Boolean(false)})
					l.pos += 2
					continue
				}
				if next == '(' {
					tokens = append(tokens, Token{Type: TkVectorStart})
					l.pos += 2
					continue
				}
			}
			tok, err := l.readAtom()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		default:
			tok, err := l.readAtom()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		}
	}

	return tokens, nil
}

func (l *Lexer) readString() (Token, error) {
	l.pos++
	var b strings.Builder
	for l.pos < l.len {
		ch := l.input[l.pos]
		if ch == '"' {
			l.pos++
			return Token{Type: TkString, Value: String(b.String())}, nil
		}
		if ch == '\\' && l.pos+1 < l.len {
			l.pos++
			next := l.input[l.pos]
			switch next {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				b.WriteRune(next)
			}
		} else {
			b.WriteRune(ch)
		}
		l.pos++
	}
	return Token{}, fmt.Errorf("unterminated string")
}

func (l *Lexer) readAtom() (Token, error) {
	start := l.pos
	for l.pos < l.len {
		ch := l.input[l.pos]
		if unicode.IsSpace(ch) || ch == '(' || ch == ')' || ch == '"' || ch == '\'' || ch == ';' {
			break
		}
		l.pos++
	}

	atom := string(l.input[start:l.pos])

	if isNumeric(atom) {
		if strings.Contains(atom, ".") {
			var f float64
			fmt.Sscanf(atom, "%f", &f)
			return Token{Type: TkNumber, Value: Float(f)}, nil
		}
		var n int64
		fmt.Sscanf(atom, "%d", &n)
		return Token{Type: TkNumber, Value: Integer(n)}, nil
	}

	return Token{Type: TkSymbol, Value: &Sym{Name: atom}}, nil
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		if len(s) == 1 {
			return false
		}
		start = 1
	}
	hasDot := false
	for i := start; i < len(s); i++ {
		if s[i] == '.' {
			if hasDot {
				return false
			}
			hasDot = true
		} else if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
