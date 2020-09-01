package lexer

import (
	"bufio"
	"bytes"
	"io"
	"unicode"
)

type Scanner struct {
	r *bufio.Reader
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

//places the previously read rune back on the reader
func (s *Scanner) unread() { _ = s.r.UnreadRune() }

func (s *Scanner) peek() rune {
	peeked := s.read()
	s.unread()
	return peeked
}

func (s *Scanner) Read() (tok TokenType, text string) {
	ch := s.read()

	if ch == eof {
		return EOF, string(ch)
	}

	if isWhitespace(ch) {
		s.consumeWhitespace()
		return s.Read()
	}

	if isBracket(ch) {
		s.unread()
		return s.readBracket()
	}

	if isStartOfSymbol(ch) {
		s.unread()
		return s.readSymbol()
	}

	if isOperatorSymbol(ch) {
		s.unread()
		return s.readOperator()
	}

	if isNumerical(ch) {
		s.unread()
		return s.readNumber()
	}

	if ch == '"' {
		return s.readString()
	}

	if isValidIdentifier(ch) {
		s.unread()
		return s.readIdentifier()
	}

	return ILLEGAL, string(ch)
}

//Consume all whitespace until we reach an eof or a non-whitespace character
func (s *Scanner) consumeWhitespace() uint {
	count := uint(0)
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		}
		count++
	}
	return count
}

func (s *Scanner) readIdentifier() (tok TokenType, text string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isValidIdentifier(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	str := buf.String()
	switch str {
	case "let":
		return LET, str
	case "mut":
		return MUT, str
	}

	return IDENTIFIER, str
}

func (s *Scanner) readBracket() (tok TokenType, text string) {
	str := s.read()
	switch str {
	case '(':
		return LPAREN, string(str)
	case ')':
		return RPAREN, string(str)
	case '{':
		return LBRACE, string(str)
	case '}':
		return RBRACE, string(str)
	case '<':
		return LESSER, string(str)
	case '>':
		return GREATER, string(str)
	}
	return ILLEGAL, string(str)
}

func (s *Scanner) readSymbol() (tok TokenType, text string) {
	ch := s.read()

	switch ch {
	case '.':
		return DOT, string(ch)
	case '=':
		peeked := s.peek()
		if peeked == '>' {
			s.read()
			return ARROW, string(ch) + string(peeked)
		}
		return EQUAL, string(ch)
	}

	return ILLEGAL, string(ch)
}

func (s *Scanner) readOperator() (tok TokenType, text string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isOperatorSymbol(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}
	str := buf.String()
	switch str {
	case "==":
		return EQUALS, str
	case "+":
		return ADD, str
	}

	return ILLEGAL, str
}

func (s *Scanner) readString() (tok TokenType, text string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '"' {
			break
		} else {
			buf.WriteRune(ch)
		}
	}
	return STRING, buf.String()
}

func (s *Scanner) readNumber() (tok TokenType, text string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	numType := INT
	for {
		ch := s.read()
		if ch == eof {
			break
		}
		if ch == '.' {
			numType = FLOAT
		} else if !unicode.IsNumber(ch) {
			break
		}
		buf.WriteRune(ch)
	}

	return numType, buf.String()
}
