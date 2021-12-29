package token

import "golox/scanner/token/tokentype"

type Token struct {
	Type   tokentype.TokenType
	Start  int
	Lexeme string
	Line   int
}
