package scanner

import (
	"golox/scanner/token"
	"golox/scanner/token/tokentype"
)

type Scanner struct {
	start   int
	current int
	line    int
	source  *string
}

func (scanner *Scanner) Init(source *string) {
	scanner.start = 0
	scanner.current = 0
	scanner.line = 1
	scanner.source = source
}

func (scanner *Scanner) SourceSubStr(start int, len int) string {
	return (*scanner.source)[start : start+len]
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}

func (scanner *Scanner) isAtEnd() bool {
	return len(*scanner.source) == scanner.current
}

func (scanner *Scanner) advance() rune {
	scanner.current++
	return rune((*scanner.source)[scanner.current-1])
}

func (scanner *Scanner) currChar() rune {
	return rune((*scanner.source)[scanner.current])
}

func (scanner *Scanner) nextChar() rune {
	return rune((*scanner.source)[scanner.current+1])
}

func (scanner *Scanner) match(expected rune) bool {
	if scanner.isAtEnd() {
		return false
	}
	if scanner.currChar() != expected {
		return false
	}
	scanner.current += 1
	return true
}

func (scanner *Scanner) makeToken(t tokentype.TokenType) token.Token {
	return token.Token{
		Type:   t,
		Line:   scanner.line,
		Start:  scanner.start,
		Lexeme: scanner.SourceSubStr(scanner.start, scanner.current-scanner.start)}
}

func (scanner *Scanner) errorToken(msg string) token.Token {
	return token.Token{
		Type:   tokentype.TOKEN_ERROR,
		Start:  scanner.start,
		Line:   scanner.line,
		Lexeme: msg}
}

func (scanner *Scanner) skipWhitespace() {
	for {
		switch scanner.currChar() {
		case ' ', '\t', '\r':
			scanner.advance()
		case '\n':
			scanner.line += 1
			scanner.advance()
		case '#':
			for scanner.currChar() != '\n' && !scanner.isAtEnd() {
				scanner.advance()
			}
		default:
			return
		}
	}
}

func (scanner *Scanner) identifierType() tokentype.TokenType {
	t := scanner.SourceSubStr(scanner.start, scanner.current-scanner.start)
	switch t {
	case "and":
		return tokentype.TOKEN_AND
	case "else":
		return tokentype.TOKEN_ELSE
	case "false":
		return tokentype.TOKEN_FALSE
	case "for":
		return tokentype.TOKEN_FOR
	case "fun":
		return tokentype.TOKEN_FUN
	case "if":
		return tokentype.TOKEN_IF
	case "var":
		return tokentype.TOKEN_VAR
	case "nil":
		return tokentype.TOKEN_NIL
	case "or":
		return tokentype.TOKEN_OR
	case "print":
		return tokentype.TOKEN_PRINT
	case "return":
		return tokentype.TOKEN_RETURN
	case "this":
		return tokentype.TOKEN_THIS
	case "true":
		return tokentype.TOKEN_TRUE
	case "while":
		return tokentype.TOKEN_WHILE
	}

	return tokentype.TOKEN_IDENTIFIER
}

func (scanner *Scanner) identifier() token.Token {
	for isAlpha(scanner.currChar()) || isDigit(scanner.currChar()) {
		scanner.advance()
	}
	return scanner.makeToken(scanner.identifierType())
}

func (scanner *Scanner) number() token.Token {
	for isDigit(scanner.currChar()) {
		scanner.advance()
	}

	if scanner.currChar() == '.' && isDigit(scanner.nextChar()) {
		scanner.advance()

		for isDigit(scanner.currChar()) {
			scanner.advance()
		}
	}

	return scanner.makeToken(tokentype.TOKEN_NUMBER)
}

func (scanner *Scanner) string() token.Token {
	for scanner.currChar() != '"' && !scanner.isAtEnd() {
		if scanner.currChar() == '\n' {
			scanner.line += 1
		}
		scanner.advance()
	}

	if scanner.isAtEnd() {
		return scanner.errorToken("Unterminated string.")
	}

	scanner.advance()
	return scanner.makeToken(tokentype.TOKEN_STRING)
}

func (scanner *Scanner) Scan(tokens chan token.Token) {
	for !scanner.isAtEnd() {
		scanner.skipWhitespace()
		scanner.start = scanner.current

		var token token.Token

		c := scanner.advance()

		if isAlpha(c) {
			token = scanner.identifier()
		}

		if isDigit(c) {
			token = scanner.number()
		}

		switch c {
		case '[':
			token = scanner.makeToken(tokentype.TOKEN_LEFT_BRACKET)
		case ']':
			token = scanner.makeToken(tokentype.TOKEN_RIGHT_BRACKET)
		case '(':
			token = scanner.makeToken(tokentype.TOKEN_LEFT_PAREN)
		case ')':
			token = scanner.makeToken(tokentype.TOKEN_RIGHT_PAREN)
		case '{':
			token = scanner.makeToken(tokentype.TOKEN_LEFT_BRACE)
		case '}':
			token = scanner.makeToken(tokentype.TOKEN_RIGHT_BRACE)
		case ';':
			token = scanner.makeToken(tokentype.TOKEN_SEMICOLON)
		case ',':
			token = scanner.makeToken(tokentype.TOKEN_COMMA)
		case '.':
			token = scanner.makeToken(tokentype.TOKEN_DOT)
		case '-':
			if scanner.match('-') {
				token = scanner.makeToken(tokentype.TOKEN_MINUS_MINUS)
			} else {
				token = scanner.makeToken(tokentype.TOKEN_MINUS)
			}
		case '+':
			if scanner.match('+') {
				token = scanner.makeToken(tokentype.TOKEN_PLUS_PLUS)
			} else {
				token = scanner.makeToken(tokentype.TOKEN_PLUS)
			}
		case '/':
			token = scanner.makeToken(tokentype.TOKEN_SLASH)
		case '*':
			token = scanner.makeToken(tokentype.TOKEN_STAR)
		case '!':
			if scanner.match('=') {
				token = scanner.makeToken(tokentype.TOKEN_BANG_EQUAL)
			} else {
				token = scanner.makeToken(tokentype.TOKEN_BANG)
			}
		case '=':
			if scanner.match('=') {
				token = scanner.makeToken(tokentype.TOKEN_EQUAL_EQUAL)
			} else {
				token = scanner.makeToken(tokentype.TOKEN_EQUAL)
			}
		case '<':
			if scanner.match('=') {
				token = scanner.makeToken(tokentype.TOKEN_LESS_EQUAL)
			} else {
				token = scanner.makeToken(tokentype.TOKEN_LESS)
			}
		case '>':
			if scanner.match('=') {
				token = scanner.makeToken(tokentype.TOKEN_GREATER_EQUAL)
			} else {
				token = scanner.makeToken(tokentype.TOKEN_GREATER)
			}
		case '"':
			token = scanner.string()
		}

		if scanner.isAtEnd() {
			token = scanner.makeToken(tokentype.TOKEN_EOF)
		} else if token.Lexeme == "" {
			token = scanner.errorToken("Unexpected character!")
		}

		tokens <- token
	}

	close(tokens)
}
