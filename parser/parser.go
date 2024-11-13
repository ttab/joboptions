package parser

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

type TokenType int

func (tt TokenType) String() string {
	switch tt {
	case TypeBinary:
		return "binary"
	case TypeBoolean:
		return "boolean"
	case TypeEndArray:
		return "end_array"
	case TypeEndDictionary:
		return "end_dictionary"
	case TypeIdentifier:
		return "identifier"
	case TypeLiteral:
		return "literal"
	case TypeNumber:
		return "number"
	case TypeStartArray:
		return "start_array"
	case TypeStartDictionary:
		return "start_dictionary"
	case TypeString:
		return "string"
	case TypeUnknown:
		return "unknown"
	default:
		panic(fmt.Sprintf("unexpected TokenType: %#v", tt))
	}
}

const (
	TypeUnknown TokenType = iota
	TypeStartDictionary
	TypeEndDictionary
	TypeStartArray
	TypeEndArray
	TypeLiteral
	TypeString
	TypeBoolean
	TypeIdentifier
	TypeNumber
	TypeBinary
)

type Token struct {
	Type  TokenType
	Value []byte
}

func (t Token) String() string {
	return string(t.Value)
}

func (t Token) NewBinaryReader() io.Reader {
	return hex.NewDecoder(bytes.NewReader(t.Value))
}

func NewScanner(data []byte) *Scanner {
	return &Scanner{
		buffer: data,
		line:   1,
	}
}

type Scanner struct {
	buffer []byte
	offset int
	line   int

	token Token
	err   error
}

var (
	tStart = []byte("<<")
	tEnd   = []byte(">>")
	tTrue  = []byte("true")
	tFalse = []byte("false")
)

func (s *Scanner) Scan() bool {
	if s.err != nil {
		return false
	}

	t := s.scan()

	if t != nil {
		s.token = *t
	}

	return t != nil
}

// WrapError wraps an error with the current scanner state.
func (s *Scanner) WrapError(err error) error {
	e := fmt.Errorf("on line %d: %w", s.line, err)

	if s.err != nil {
		return errors.Join(e, s.err)
	}

	return e
}

func (s *Scanner) WrapErrorf(format string, a ...any) error {
	return s.WrapError(fmt.Errorf(format, a...))
}

func (s *Scanner) UnexpectedTokenError(want TokenType, got Token) error {
	return s.WrapErrorf(
		"expected a %s token, got %s %q",
		want, got.Type, string(got.Value))
}

func (s *Scanner) Token() Token {
	return s.token
}

func (s *Scanner) Err() error {
	if s.err == nil {
		return nil
	}

	return s.WrapError(s.err)
}

func (s *Scanner) scan() *Token {
	ok := s.skipWS()
	if !ok {
		return nil
	}

	next := s.buffer[s.offset:]

	switch next[0] {
	case '/':
		return s.captureUntilWs(TypeLiteral)
	case '[':
		return s.advanceAndCapture(TypeStartArray, 1)
	case ']':
		return s.advanceAndCapture(TypeEndArray, 1)
	case '(':
		return s.captureUntil(TypeString, ')')
	case '-':
		return s.captureUntilWs(TypeNumber)
	}

	switch {
	case bytes.HasPrefix(next, tStart):
		return s.advanceAndCapture(
			TypeStartDictionary, len(tStart))
	case bytes.HasPrefix(next, tEnd):
		return s.advanceAndCapture(
			TypeEndDictionary, len(tEnd))
	case bytes.HasPrefix(next, tTrue):
		return s.advanceAndCapture(
			TypeBoolean, len(tTrue))
	case bytes.HasPrefix(next, tFalse):
		return s.advanceAndCapture(
			TypeBoolean, len(tFalse))
	case next[0] >= 'a' && next[0] <= 'z':
		return s.captureUntilWs(TypeIdentifier)
	case next[0] >= '0' && next[0] <= '9':
		return s.captureNumber()
	case next[0] == '<':
		return s.captureUntil(TypeBinary, '>')
	default:
		s.err = fmt.Errorf("unexpected token %q", next[0:1])

		return nil
	}
}

func (s *Scanner) captureUntil(t TokenType, c byte) *Token {
	start := s.offset + 1

	for {
		s.offset++

		if s.offset == len(s.buffer) {
			s.err = fmt.Errorf("expected end of string: %w",
				io.ErrUnexpectedEOF)

			return nil
		}

		if s.buffer[s.offset] == c {
			s.offset++

			break
		}
	}

	return &Token{
		Type:  t,
		Value: s.buffer[start : s.offset-1],
	}
}

func (s *Scanner) captureNumber() *Token {
	dotPos := -1

	start := s.offset

	for {
		s.offset++

		if s.offset == len(s.buffer) {
			break
		}

		c := s.buffer[s.offset]

		if c == '.' {
			if dotPos != -1 {
				s.err = errors.New("unexpected character '.'")

				return nil
			}

			dotPos = s.offset
		}

		isNum := c == '.' || (c >= '0' && c <= '9')
		if !isNum {
			break
		}
	}

	return &Token{
		Type:  TypeNumber,
		Value: s.buffer[start:s.offset],
	}
}

var ws = map[byte]bool{
	'\n': true,
	'\r': true,
	' ':  true,
	'\t': true,
}

func (s *Scanner) captureUntilWs(
	t TokenType,
) *Token {
	start := s.offset

	for s.offset < len(s.buffer) {
		if ws[s.buffer[s.offset]] {
			break
		}

		s.offset++
	}

	return &Token{
		Type:  t,
		Value: s.buffer[start:s.offset],
	}
}

func (s *Scanner) advanceAndCapture(
	t TokenType, l int,
) *Token {
	v := s.buffer[s.offset : s.offset+l]

	s.offset += l

	return &Token{
		Type:  t,
		Value: v,
	}
}

func (s *Scanner) skipWS() bool {
	if s.offset == len(s.buffer) {
		return false
	}

	for {
		if !ws[s.buffer[s.offset]] {
			break
		}

		if s.buffer[s.offset] == '\n' {
			s.line++
		}

		s.offset++

		if s.offset == len(s.buffer) {
			return false
		}
	}

	return true
}
