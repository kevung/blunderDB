package pdf

import (
	"bytes"
	"strconv"
)

// Object is a PDF object: nil (null), bool, int64, float64, String, Name,
// Array, Dict, *Stream or Ref.
type Object any

// Name is a PDF name, without its slash.
type Name string

// String is a PDF string's bytes, escapes decoded.
type String string

// Array is a PDF array.
type Array []Object

// Dict is a PDF dictionary.
type Dict map[Name]Object

// Ref is an indirect reference.
type Ref struct{ Num, Gen int }

// Stream is a stream object: its dictionary and its raw (still encoded) bytes.
type Stream struct {
	Dict Dict
	Raw  []byte
}

// keyword is a bare word: an operator in a content stream, obj, R, stream…
type keyword string

// delimiters ends a token without being part of it.
type delimiter byte

const maxNesting = 64

func isWhite(c byte) bool {
	return c == 0 || c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}

func isDelim(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

type lexer struct {
	b   []byte
	pos int
}

func (l *lexer) skipWhite() {
	for l.pos < len(l.b) {
		c := l.b[l.pos]
		if isWhite(c) {
			l.pos++
			continue
		}
		if c == '%' {
			for l.pos < len(l.b) && l.b[l.pos] != '\n' && l.b[l.pos] != '\r' {
				l.pos++
			}
			continue
		}
		return
	}
}

// eof is the token past the last byte.
type eof struct{}

// token returns the next token: a scalar Object (number, Name, String, bool,
// nil), a keyword, a delimiter ('[' ']' '{' '}', and '<' / '>' standing for
// "<<" and ">>"), or eof{}.
func (l *lexer) token() any {
	l.skipWhite()
	if l.pos >= len(l.b) {
		return eof{}
	}
	c := l.b[l.pos]
	switch {
	case c == '[' || c == ']' || c == '{' || c == '}':
		l.pos++
		return delimiter(c)
	case c == '<':
		if l.pos+1 < len(l.b) && l.b[l.pos+1] == '<' {
			l.pos += 2
			return delimiter('<')
		}
		return l.hexString()
	case c == '>':
		l.pos++
		if l.pos < len(l.b) && l.b[l.pos] == '>' {
			l.pos++
		}
		return delimiter('>')
	case c == '(':
		return l.litString()
	case c == ')':
		l.pos++
		return keyword(")")
	case c == '/':
		return l.name()
	case c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9'):
		return l.number()
	}
	start := l.pos
	for l.pos < len(l.b) && !isWhite(l.b[l.pos]) && !isDelim(l.b[l.pos]) {
		l.pos++
	}
	w := string(l.b[start:l.pos])
	switch w {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	return keyword(w)
}

func (l *lexer) number() any {
	start := l.pos
	if c := l.b[l.pos]; c == '+' || c == '-' {
		l.pos++
	}
	real := false
	for l.pos < len(l.b) {
		c := l.b[l.pos]
		if c == '.' {
			real = true
		} else if c < '0' || c > '9' {
			break
		}
		l.pos++
	}
	s := string(l.b[start:l.pos])
	if !real {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		// "-", "." or "--5": malformed numbers read as zero, as viewers do.
		return int64(0)
	}
	return f
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func (l *lexer) name() any {
	l.pos++ // '/'
	var buf []byte
	for l.pos < len(l.b) {
		c := l.b[l.pos]
		if isWhite(c) || isDelim(c) {
			break
		}
		if c == '#' && l.pos+2 < len(l.b) {
			h, ok1 := unhex(l.b[l.pos+1])
			lo, ok2 := unhex(l.b[l.pos+2])
			if ok1 && ok2 {
				buf = append(buf, h<<4|lo)
				l.pos += 3
				continue
			}
		}
		buf = append(buf, c)
		l.pos++
	}
	return Name(buf)
}

func (l *lexer) hexString() any {
	l.pos++ // '<'
	var buf []byte
	var half byte
	odd := false
	for l.pos < len(l.b) {
		c := l.b[l.pos]
		l.pos++
		if c == '>' {
			break
		}
		v, ok := unhex(c)
		if !ok {
			continue
		}
		if odd {
			buf = append(buf, half<<4|v)
		} else {
			half = v
		}
		odd = !odd
	}
	if odd {
		buf = append(buf, half<<4)
	}
	return String(buf)
}

func (l *lexer) litString() any {
	l.pos++ // '('
	var buf []byte
	depth := 1
	for l.pos < len(l.b) {
		c := l.b[l.pos]
		l.pos++
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return String(buf)
			}
		case '\\':
			if l.pos >= len(l.b) {
				return String(buf)
			}
			e := l.b[l.pos]
			l.pos++
			switch e {
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			case 'b':
				c = '\b'
			case 'f':
				c = '\f'
			case '\r':
				if l.pos < len(l.b) && l.b[l.pos] == '\n' {
					l.pos++
				}
				continue
			case '\n':
				continue
			default:
				if e >= '0' && e <= '7' {
					v := int(e - '0')
					for k := 0; k < 2 && l.pos < len(l.b) && l.b[l.pos] >= '0' && l.b[l.pos] <= '7'; k++ {
						v = v*8 + int(l.b[l.pos]-'0')
						l.pos++
					}
					c = byte(v)
				} else {
					c = e
				}
			}
		}
		buf = append(buf, c)
	}
	return String(buf)
}

// object parses one object from the lexer. With refs, "n g R" is a Ref; a
// content stream has no references and passes false. A keyword met where an
// object was expected is returned as is, for the caller to judge.
func (l *lexer) object(refs bool, depth int) (any, error) {
	if depth > maxNesting {
		return nil, malformed("objects nested deeper than %d", maxNesting)
	}
	t := l.token()
	switch v := t.(type) {
	case delimiter:
		switch v {
		case '[':
			var a Array
			for {
				save := l.pos
				if d, ok := l.token().(delimiter); ok && d == ']' {
					return a, nil
				}
				l.pos = save
				o, err := l.object(refs, depth+1)
				if err != nil {
					return nil, err
				}
				if _, ok := o.(eof); ok {
					return a, nil
				}
				if k, ok := o.(keyword); ok {
					if k == "endobj" || k == "stream" {
						return a, nil
					}
					continue
				}
				if _, ok := o.(delimiter); ok {
					continue
				}
				a = append(a, o)
			}
		case '<':
			d := Dict{}
			for {
				k := l.token()
				switch kk := k.(type) {
				case Name:
					o, err := l.object(refs, depth+1)
					if err != nil {
						return nil, err
					}
					switch ov := o.(type) {
					case eof:
						return d, nil
					case keyword:
						if ov == "endobj" || ov == "stream" {
							return d, nil
						}
					case delimiter:
						if ov == '>' {
							return d, nil
						}
					default:
						d[kk] = o
					}
				case delimiter:
					if kk == '>' {
						return d, nil
					}
				case eof:
					return d, nil
				case keyword:
					if kk == "endobj" || kk == "stream" {
						return d, nil
					}
				}
			}
		}
		return v, nil
	case int64:
		if refs {
			save := l.pos
			if g, ok := l.token().(int64); ok {
				if k, ok := l.token().(keyword); ok && k == "R" {
					return Ref{Num: int(v), Gen: int(g)}, nil
				}
			}
			l.pos = save
		}
		return v, nil
	}
	return t, nil
}

func (l *lexer) hasPrefix(s string) bool {
	return bytes.HasPrefix(l.b[l.pos:], []byte(s))
}
