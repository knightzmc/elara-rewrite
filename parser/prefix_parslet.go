package parser

import (
	"github.com/ElaraLang/elara/ast"
	"github.com/ElaraLang/elara/lexer"
	"strconv"
)

func (p *Parser) initPrefixParselets() {
	p.prefixParslets = make(map[lexer.TokenType]prefixParslet, 0)
	p.registerPrefix(lexer.Identifier, p.parseIdentifier)
	p.registerPrefix(lexer.LSquare, p.parseCollection)
	p.registerPrefix(lexer.LBrace, p.parseMap)
	p.registerPrefix(lexer.BinaryInt, p.parseInteger)
	p.registerPrefix(lexer.OctalInt, p.parseInteger)
	p.registerPrefix(lexer.HexadecimalInt, p.parseInteger)
	p.registerPrefix(lexer.DecimalInt, p.parseInteger)
	p.registerPrefix(lexer.Float, p.parseFloat)
	p.registerPrefix(lexer.Char, p.parseChar)
	p.registerPrefix(lexer.String, p.parseString)
	p.registerPrefix(lexer.LParen, p.resolvingPrefixParslet(p.functionGroupResolver()))
	p.registerPrefix(lexer.BooleanTrue, p.parseBoolean)
	p.registerPrefix(lexer.BooleanFalse, p.parseBoolean)
	p.registerPrefix(lexer.Subtract, p.parseUnaryExpression)
	p.registerPrefix(lexer.Not, p.parseUnaryExpression)
	p.registerPrefix(lexer.If, p.parseIfExpression)
}

func (p *Parser) parseIfExpression() ast.Expression {
	operator := p.Tape.Consume(lexer.If)
	condition := p.parseExpression(Lowest)
	var mainBranch ast.Statement
	var elseBranch ast.Statement
	if p.Tape.Match(lexer.Arrow) {
		mainBranch = p.parseExpressionStatement()
	} else {
		p.Tape.skipLineBreaks()
		mainBranch = p.parseBlockStatement()
	}
	p.Tape.skipLineBreaks()
	if p.Tape.Match(lexer.Else) {
		switch p.Tape.Current().TokenType {
		case lexer.Arrow:
			p.Tape.Match(lexer.Arrow)
			fallthrough
		case lexer.If:
			elseBranch = p.parseExpressionStatement()
		default:
			p.Tape.skipLineBreaks()
			elseBranch = p.parseBlockStatement()

		}
	}

	return &ast.IfExpression{
		Token:      operator,
		Condition:  condition,
		MainBranch: mainBranch,
		ElseBranch: elseBranch,
	}
}

func (p *Parser) parseUnaryExpression() ast.Expression {
	operator := p.Tape.Consume(lexer.Dot)
	expr := p.parseExpression(Prefix)
	return &ast.UnaryExpression{
		Token:    operator,
		Operator: operator,
		Right:    expr,
	}
}

func (p *Parser) parseIdentifier() ast.Expression {
	token := p.Tape.Consume(lexer.Identifier)
	return &ast.IdentifierLiteral{Token: token, Name: string(token.Data)}
}

func (p *Parser) parseInteger() ast.Expression {
	token := p.Tape.ConsumeAny()
	base := baseOf(token.TokenType)
	if base < 2 {
		p.error(token, "Invalid integer token received!")
	}
	value, err := strconv.ParseInt(string(token.Data), base, 64)
	if err != nil {
		p.error(token, "Error parsing integer token!")
	}
	return &ast.IntegerLiteral{Token: token, Value: value}
}

func baseOf(tokenType lexer.TokenType) int {
	switch tokenType {
	case lexer.HexadecimalInt:
		return 16
	case lexer.DecimalInt:
		return 10
	case lexer.OctalInt:
		return 8
	case lexer.BinaryInt:
		return 2
	}
	return -1
}

func (p *Parser) parseFloat() ast.Expression {
	token := p.Tape.Consume(lexer.Float)
	value, err := strconv.ParseFloat(string(token.Data), 10)
	if err != nil {
		p.error(token, "Error parsing float token!")
	}
	return &ast.FloatLiteral{Token: token, Value: value}
}

func (p *Parser) parseBoolean() ast.Expression {
	token := p.Tape.Consume(lexer.BooleanTrue, lexer.BooleanFalse)
	value := token.TokenType == lexer.BooleanTrue
	return &ast.BooleanLiteral{Token: token, Value: value}
}

func (p *Parser) parseChar() ast.Expression {
	token := p.Tape.Consume(lexer.Char)
	value := token.Data[0]
	return &ast.CharLiteral{Token: token, Value: value}
}

func (p *Parser) parseString() ast.Expression {
	token := p.Tape.Consume(lexer.String)
	value := string(token.Data)
	return &ast.StringLiteral{Token: token, Value: value}
}

func (p *Parser) parseFunction() ast.Expression {
	token := p.Tape.Consume(lexer.LParen)
	params := p.parseFunctionParameters()
	p.Tape.Expect(lexer.RParen)
	p.Tape.Expect(lexer.Arrow)
	p.Tape.skipLineBreaks()
	var typ ast.Type
	var body ast.Statement
	// !p.Tape.ValidationPeek(0, lexer.LBrace)
	if p.isReturnTypeProvided() {
		typ = p.parseType(TypeLowest)
	}
	p.Tape.skipLineBreaks()
	if p.Tape.ValidateHead(lexer.LBrace) {
		body = p.parseBlockStatement()
	} else {
		body = p.parseExpressionStatement()
	}
	return &ast.FunctionLiteral{
		Token:      token,
		ReturnType: typ,
		Parameters: params,
		Body:       body,
	}
}

func (p *Parser) parseGroupExpression() ast.Expression {
	p.Tape.Expect(lexer.LParen)
	p.Tape.skipLineBreaks()
	expr := p.parseExpression(Lowest)
	p.Tape.skipLineBreaks()
	p.Tape.Expect(lexer.RParen)
	return expr
}

func (p *Parser) parseCollection() ast.Expression {
	tok := p.Tape.Consume(lexer.LSquare)
	p.Tape.skipLineBreaks()
	elements := p.parseCollectionElements()
	p.Tape.skipLineBreaks()
	p.Tape.Expect(lexer.RSquare)
	return &ast.CollectionLiteral{Token: tok, Elements: elements}
}

func (p *Parser) parseMap() ast.Expression {
	tok := p.Tape.Consume(lexer.LBrace)
	p.Tape.skipLineBreaks()
	elements := p.parseMapEntries()
	p.Tape.skipLineBreaks()
	p.Tape.Consume(lexer.RBrace)
	return &ast.MapLiteral{Token: tok, Entries: elements}
}
