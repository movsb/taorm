package filter

import (
	"bytes"
	"container/list"
	"fmt"
	"strings"
	"unicode/utf8"
)

// TokenType is the type of a token.
type TokenType uint

// TokenType list
const (
	_ TokenType = iota

	TokenTypeInvalid
	TokenTypeEOF

	TokenTypeAnd
	TokenTypeOr

	TokenTypeIdent

	TokenTypeEqual
	TokenTypeNotEqual
	TokenTypeInclude
	TokenTypeNotInclude
	TokenTypeStartsWith
	TokenTypeEndsWith
	TokenTypeMatch
	TokenTypeNotMatch

	TokenTypeGreaterThan
	TokenTypeLessThan
	TokenTypeGreaterThanOrEqual
	TokenTypeLessThanOrEqual
)

// Token is a token.
type Token struct {
	TokenType  TokenType
	TokenValue string
}

// Tokenizer tokenize input into tokens.
type Tokenizer struct {
	sr  *strings.Reader
	buf *list.List
}

// NewTokenizer creates a new tokenizer.
func NewTokenizer(input string) *Tokenizer {
	return &Tokenizer{
		sr:  strings.NewReader(input),
		buf: list.New(),
	}
}

// Next returns the next token tokenized.
func (t *Tokenizer) Next() (token Token, err error) {
	if t.buf.Len() > 0 {
		e := t.buf.Front()
		t.buf.Remove(e)
		return e.Value.(Token), nil
	}

	defer func() {
		if er := recover(); er != nil {
			err = er.(error)
		}
	}()

	token = t.next()
	err = nil
	return
}

// Undo undo-es a token.
func (t *Tokenizer) Undo(token Token) {
	t.buf.PushFront(token)
}

func (t *Tokenizer) read() rune {
	ch, _, _ := t.sr.ReadRune()
	return ch
}

func (t *Tokenizer) unread(r rune) {
	if r != 0 {
		t.sr.UnreadRune()
	}
}

func (t *Tokenizer) next() (token Token) {
	var ch rune

	for {
		ch = t.read()
		if ch == '=' {
			ch = t.read()
			switch ch {
			case '=':
				token.TokenType = TokenTypeEqual
				token.TokenValue = "=="
			case '@':
				token.TokenType = TokenTypeInclude
				token.TokenValue = "=@"
			case '~':
				token.TokenType = TokenTypeMatch
				token.TokenValue = "=~"
			default:
				t.unread(ch)
				token.TokenType = TokenTypeInvalid
				token.TokenValue = fmt.Sprintf("%c", ch)
			}
			return
		} else if ch == '!' {
			ch = t.read()
			switch ch {
			case '=':
				token.TokenType = TokenTypeNotEqual
				token.TokenValue = "!="
			case '@':
				token.TokenType = TokenTypeNotInclude
				token.TokenValue = "!@"
			case '~':
				token.TokenType = TokenTypeNotMatch
				token.TokenValue = "!~"
			default:
				t.unread(ch)
			}
			return
		} else if ch == '^' {
			ch = t.read()
			switch ch {
			case '=':
				token.TokenType = TokenTypeStartsWith
				token.TokenValue = "^="
			default:
				t.unread(ch)
			}
			return
		} else if ch == '$' {
			ch = t.read()
			switch ch {
			case '=':
				token.TokenType = TokenTypeEndsWith
				token.TokenValue = "$="
			default:
				t.unread(ch)
			}
			return
		} else if ch == '>' {
			ch = t.read()
			if ch == '=' {
				token.TokenType = TokenTypeGreaterThanOrEqual
				token.TokenValue = ">="
			} else {
				t.unread(ch)
				token.TokenType = TokenTypeGreaterThan
				token.TokenValue = ">"
			}
			return
		} else if ch == '<' {
			ch = t.read()
			if ch == '=' {
				token.TokenType = TokenTypeLessThanOrEqual
				token.TokenValue = "<="
			} else {
				t.unread(ch)
				token.TokenType = TokenTypeLessThan
				token.TokenValue = "<"
			}
			return
		} else if ch == ';' {
			token.TokenType = TokenTypeAnd
			token.TokenValue = " AND "
			return
		} else if ch == ',' {
			token.TokenType = TokenTypeOr
			token.TokenValue = " OR "
			return
		} else if ch == ' ' {
			continue
		} else if ch == 0 {
			token.TokenType = TokenTypeEOF
			token.TokenValue = ""
			return
		} else {
			t.unread(ch)
			sw := bytes.NewBuffer(nil)
			for {
				ch = t.read()
				if ch >= '0' && ch <= '9' ||
					ch >= 'a' && ch <= 'z' ||
					ch >= 'A' && ch <= 'Z' ||
					ch == ':' || ch == '.' ||
					ch == ' ' || ch == '/' ||
					ch == '_' || ch == '-' ||
					ch == '*' || ch == '+' ||
					ch == '\t' || ch >= utf8.RuneSelf {
					sw.WriteRune(ch)
				} else if ch == '\\' {
					ch = t.read()
					if ch == ',' || ch == ';' {
						sw.WriteRune(ch)
					} else {
						t.unread(ch)
						sw.WriteRune('\\')
					}
				} else {
					s := strings.Trim(sw.String(), "\t ")
					if len(s) > 0 {
						token.TokenType = TokenTypeIdent
						token.TokenValue = s
						t.unread(ch)
					} else {
						token.TokenType = TokenTypeInvalid
						token.TokenValue = fmt.Sprintf("%c", ch)
					}
					return
				}
			}
		}
	}
}
