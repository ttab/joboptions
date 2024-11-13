package joboptions

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ttab/joboptions/parser"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type Literal string

type ValueType int

const (
	ValueString ValueType = iota
	ValueArray
	ValueBoolean
	ValueBinary
	ValueDictionary
	ValueFloat
	ValueInteger
	ValueLiteral
)

type Value struct {
	Type       ValueType
	Array      []Value    `json:"array,omitempty"`
	Boolean    bool       `json:"boolean,omitempty"`
	Binary     []byte     `json:"binary,omitempty"`
	Dictionary Dictionary `json:"dictionary,omitempty"`
	Float      float64    `json:"float,omitempty"`
	Integer    int        `json:"integer,omitempty"`
	Literal    Literal    `json:"literal,omitempty"`
	String     string     `json:"string,omitempty"`
}

func (v Value) StringFromUTF16() (string, error) {
	if v.Type != ValueBinary {
		return "", fmt.Errorf("not a binary value")
	}

	utf16Dec := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder()

	r := transform.NewReader(bytes.NewReader(v.Binary),
		transform.Chain(utf16Dec.Transformer, norm.NFC))

	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("transform to utf8: %w", err)
	}

	return string(data), nil
}

type Dictionary map[Literal]Value

type Parameters map[string]Dictionary

func Parse(data []byte) (Parameters, error) {
	p := parser.NewScanner(data)

	params := make(Parameters)

	for p.Scan() {
		t := p.Token()

		if t.Type != parser.TypeStartDictionary {
			return nil, p.UnexpectedTokenError(parser.TypeStartDictionary, t)
		}

		d, err := parseDictionary(p)
		if err != nil {
			return nil, fmt.Errorf("parse parameter dictionary: %w", err)
		}

		if !p.Scan() {
			return nil, p.WrapErrorf("parse parameter dictionary name")
		}

		t = p.Token()
		if t.Type != parser.TypeIdentifier {
			return nil, p.UnexpectedTokenError(parser.TypeIdentifier, t)
		}

		params[string(t.Value)] = d
	}

	if p.Err() != nil {
		return nil, fmt.Errorf("parse data: %w", p.Err())
	}

	return params, nil
}

func parseDictionary(p *parser.Scanner) (Dictionary, error) {
	d := make(Dictionary)

	for p.Scan() {
		t := p.Token()

		if t.Type == parser.TypeEndDictionary {
			return d, nil
		}

		if t.Type != parser.TypeLiteral {
			return nil, p.UnexpectedTokenError(parser.TypeLiteral, t)
		}

		key := Literal(t.Value)

		if !p.Scan() {
			return nil, p.WrapErrorf("parse dictionary value")
		}

		t = p.Token()

		value, err := parseValue(p, t)
		if err != nil {
			return nil, fmt.Errorf("parse value of %q: %w", key, err)
		}

		d[key] = value
	}

	return nil, io.ErrUnexpectedEOF
}

func parseArray(p *parser.Scanner) (Value, error) {
	var a []Value

	var idx int

	for p.Scan() {
		t := p.Token()
		if t.Type == parser.TypeEndArray {
			return Value{
				Type:  ValueArray,
				Array: a,
			}, nil
		}

		value, err := parseValue(p, t)
		if err != nil {
			return Value{}, fmt.Errorf("parse value at index %d: %w", idx, err)
		}

		idx++

		a = append(a, value)
	}

	return Value{}, io.ErrUnexpectedEOF
}

func parseValue(p *parser.Scanner, t parser.Token) (Value, error) {
	switch t.Type {
	case parser.TypeBoolean:
		isTrue := bytes.Equal(t.Value, []byte("true"))

		return Value{
			Type:    ValueBoolean,
			Boolean: isTrue,
		}, nil
	case parser.TypeStartArray:
		return parseArray(p)
	case parser.TypeLiteral:
		return Value{
			Type:    ValueLiteral,
			Literal: Literal(t.Value),
		}, nil
	case parser.TypeString:
		quoted := `"` + string(t.Value) + `"`

		// A bit optimistic, won't handle unescaped " or '. Why the hell
		// didn't they choose a C string representation of strings?
		str, err := strconv.Unquote(quoted)
		if err != nil {
			return Value{}, p.WrapErrorf("unquote string value: %w", err)
		}

		return Value{
			Type:   ValueString,
			String: str,
		}, nil
	case parser.TypeNumber:
		s := string(t.Value)

		if strings.Contains(s, ".") {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return Value{}, p.WrapErrorf("parse float value: %w", err)
			}

			return Value{
				Type:  ValueFloat,
				Float: f,
			}, nil
		}

		n, err := strconv.Atoi(s)
		if err != nil {
			return Value{}, p.WrapErrorf("parse int value: %w", err)
		}

		return Value{
			Type:    ValueInteger,
			Integer: n,
		}, nil
	case parser.TypeBinary:
		cpy := make([]byte, hex.DecodedLen(len(t.Value)))

		_, err := hex.Decode(cpy, t.Value)
		if err != nil {
			return Value{}, p.WrapErrorf("decode hex data: %w", err)
		}

		return Value{
			Type:   ValueBinary,
			Binary: cpy,
		}, nil
	case parser.TypeStartDictionary:
		d, err := parseDictionary(p)
		if err != nil {
			return Value{}, err
		}

		return Value{
			Type:       ValueDictionary,
			Dictionary: d,
		}, nil
	default:
		return Value{}, p.UnexpectedTokenError(parser.TypeIdentifier, t)
	}
}
