package tokentype

type TokenType uint8

const (
	// Single-character tokens.
	TOKEN_LEFT_BRACKET  TokenType = iota
	TOKEN_RIGHT_BRACKET TokenType = iota
	TOKEN_LEFT_PAREN    TokenType = iota
	TOKEN_RIGHT_PAREN   TokenType = iota
	TOKEN_LEFT_BRACE    TokenType = iota
	TOKEN_RIGHT_BRACE   TokenType = iota
	TOKEN_COMMA         TokenType = iota
	TOKEN_DOT           TokenType = iota
	TOKEN_SEMICOLON     TokenType = iota
	TOKEN_SLASH         TokenType = iota
	TOKEN_STAR          TokenType = iota

	// One or two character tokens.
	TOKEN_MINUS         TokenType = iota
	TOKEN_MINUS_MINUS   TokenType = iota
	TOKEN_PLUS          TokenType = iota
	TOKEN_PLUS_PLUS     TokenType = iota
	TOKEN_BANG          TokenType = iota
	TOKEN_BANG_EQUAL    TokenType = iota
	TOKEN_EQUAL         TokenType = iota
	TOKEN_EQUAL_EQUAL   TokenType = iota
	TOKEN_GREATER       TokenType = iota
	TOKEN_GREATER_EQUAL TokenType = iota
	TOKEN_LESS          TokenType = iota
	TOKEN_LESS_EQUAL    TokenType = iota

	// Literals.
	TOKEN_IDENTIFIER TokenType = iota
	TOKEN_STRING     TokenType = iota
	TOKEN_NUMBER     TokenType = iota

	// Keywords.
	TOKEN_AND    TokenType = iota
	TOKEN_ELSE   TokenType = iota
	TOKEN_FALSE  TokenType = iota
	TOKEN_FOR    TokenType = iota
	TOKEN_FUN    TokenType = iota
	TOKEN_IF     TokenType = iota
	TOKEN_NIL    TokenType = iota
	TOKEN_OR     TokenType = iota
	TOKEN_PRINT  TokenType = iota
	TOKEN_RETURN TokenType = iota
	TOKEN_THIS   TokenType = iota
	TOKEN_TRUE   TokenType = iota
	TOKEN_VAR    TokenType = iota
	TOKEN_WHILE  TokenType = iota

	TOKEN_ERROR TokenType = iota
	TOKEN_EOF   TokenType = iota
)
