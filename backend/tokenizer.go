package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type TokenKind int

const (
	TokenKindWord TokenKind = iota
	TokenKindString
	TokenKindSingleChar
)

type Token struct {
	kind     TokenKind
	value    string
	location Location
}

var ErrTokenizerUnexpectedEof = errors.New("unexpected end of file")
var ErrTokenizerUnterminatedString = errors.New("unterminated string literal")

type Tokenizer struct {
	runes            []rune
	cursor           int
	location         Location
	singleCharTokens []rune
}

func NewTokenizer(code string, location Location) *Tokenizer {
	return &Tokenizer{
		runes:            []rune(code),
		cursor:           0,
		location:         location,
		singleCharTokens: []rune{'(', ')', '{', '}', ','},
	}
}

func (t *Tokenizer) peek() (rune, bool) {
	if t.cursor >= len(t.runes) {
		return 0, false
	}
	return t.runes[t.cursor], true
}

func (t *Tokenizer) advance() (rune, bool) {
	if t.cursor >= len(t.runes) {
		return 0, false
	}
	char := t.runes[t.cursor]
	t.cursor++
	if char == '\n' {
		t.location.row++
		t.location.column = 1
	} else {
		t.location.column++
	}
	return char, true
}

func (t *Tokenizer) skipWhitespace() {
	for {
		char, ok := t.peek()
		if !ok || !unicode.IsSpace(char) {
			return
		}
		t.advance()
	}
}

func (t *Tokenizer) isSingleChar(r rune) bool {
	return slices.Contains(t.singleCharTokens, r)
}

func (t *Tokenizer) NextToken() (*Token, error) {
	t.skipWhitespace()

	char, ok := t.peek()
	if !ok {
		return nil, nil
	}

	loc := t.location

	if t.isSingleChar(char) {
		t.advance()
		return &Token{kind: TokenKindSingleChar, value: string(char), location: loc}, nil
	}

	if char == '"' {
		t.advance()
		var value strings.Builder

		for {
			c, ok := t.advance()
			if !ok {
				return nil, fmt.Errorf(
					"%s:%d:%d: %w",
					t.location.path, t.location.row, t.location.column,
					ErrTokenizerUnterminatedString,
				)
			}
			if c == '\\' {
				escaped, ok := t.advance()
				if !ok {
					return nil, fmt.Errorf(
						"%s:%d:%d: %w",
						t.location.path, t.location.row, t.location.column,
						ErrTokenizerUnterminatedString,
					)
				}
				switch escaped {
				case 'n':
					value.WriteRune('\n')
				case 't':
					value.WriteRune('\t')
				case 'r':
					value.WriteRune('\r')
				case '"':
					value.WriteRune('"')
				case '\\':
					value.WriteRune('\\')
				default:
					value.WriteString("\\" + string(escaped))
				}
				continue
			}
			if c == '"' {
				break
			}
			value.WriteRune(c)
		}
		return &Token{kind: TokenKindString, value: value.String(), location: loc}, nil
	}

	var value strings.Builder
	for {
		char, ok = t.peek()
		if !ok || unicode.IsSpace(char) || t.isSingleChar(char) || char == '"' {
			break
		}
		t.advance()
		value.WriteRune(char)
	}
	return &Token{kind: TokenKindWord, value: value.String(), location: loc}, nil
}

func (tok *Token) IsWord() bool {
	return tok.kind == TokenKindWord
}

func (tok *Token) IsString() bool {
	return tok.kind == TokenKindString
}

func (tok *Token) IsSingleChar(r rune) bool {
	return tok.kind == TokenKindSingleChar && []rune(tok.value)[0] == r
}
