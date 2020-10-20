package lexer

import (
	"strings"
)

func lex(file *string, code string) []Token {
	reader := strings.NewReader(code)
	scanner := NewScanner(reader)

	tokens := make([]Token, 0)
	for {
		tok, str, line, col := scanner.Read()
		if tok == EOF {
			break
		}

		tokens = append(tokens, Token{
			TokenType: tok,
			Text:      str,
			Position:  CreatePosition(file, line, col),
		})
	}
	return tokens
}
