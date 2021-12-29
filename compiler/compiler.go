package compiler

import (
	"encoding/binary"
	"fmt"
	"golox/chunk"
	"golox/chunk/opcode"
	"golox/config"
	"golox/debug"
	"golox/scanner"
	"golox/scanner/token"
	"golox/scanner/token/tokentype"
	"golox/value"
	"golox/value/functype"
	"math"
	"os"
	"strconv"
)

type Parser struct {
	previous  token.Token
	current   token.Token
	scanner   *scanner.Scanner
	compiler  *Compiler
	panicMode bool
	hadError  bool
}

type Compiler struct {
	enclosing  *Compiler
	function   *value.ObjFunction
	funcType   functype.FuncType
	locals     []Local
	upvalues   [256]Upvalue
	scopeDepth int
}

type Upvalue struct {
	index   uint8
	isLocal bool
}

type Precedence uint8

const (
	PREC_NONE       Precedence = iota
	PREC_ASSIGNMENT Precedence = iota // =
	PREC_OR         Precedence = iota // or
	PREC_AND        Precedence = iota // and
	PREC_EQUALITY   Precedence = iota // == !=
	PREC_COMPARISON Precedence = iota // < > <= >=
	PREC_TERM       Precedence = iota // + -
	PREC_FACTOR     Precedence = iota // * /
	PREC_UNARY      Precedence = iota // ! -
	PREC_CALL       Precedence = iota // . ()
	PREC_SUBSR      Precedence = iota // []
	PREC_PRIMARY    Precedence = iota
)

type ParseRule struct {
	prefix     ParseFn
	infix      ParseFn
	precedence Precedence
}

type ParseFn func(receiver *Parser, canAssign bool)

type Local struct {
	name  token.Token
	depth int
}

var rules map[tokentype.TokenType]ParseRule

func initRules() {
	rules = make(map[tokentype.TokenType]ParseRule)
	rules[tokentype.TOKEN_LEFT_BRACKET] = ParseRule{(*Parser).list, (*Parser).subscr, PREC_SUBSR}
	rules[tokentype.TOKEN_RIGHT_BRACKET] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_LEFT_PAREN] = ParseRule{(*Parser).grouping, (*Parser).call, PREC_CALL}
	rules[tokentype.TOKEN_RIGHT_PAREN] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_LEFT_BRACE] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_RIGHT_BRACE] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_COMMA] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_DOT] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_MINUS] = ParseRule{(*Parser).unary, (*Parser).binary, PREC_TERM}
	rules[tokentype.TOKEN_MINUS_MINUS] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_PLUS] = ParseRule{nil, (*Parser).binary, PREC_TERM}
	rules[tokentype.TOKEN_PLUS_PLUS] = ParseRule{(*Parser).unary, nil, PREC_TERM}
	rules[tokentype.TOKEN_SEMICOLON] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_SLASH] = ParseRule{nil, (*Parser).binary, PREC_FACTOR}
	rules[tokentype.TOKEN_STAR] = ParseRule{nil, (*Parser).binary, PREC_FACTOR}
	rules[tokentype.TOKEN_BANG] = ParseRule{(*Parser).unary, nil, PREC_NONE}
	rules[tokentype.TOKEN_BANG_EQUAL] = ParseRule{(*Parser).binary, (*Parser).binary, PREC_EQUALITY}
	rules[tokentype.TOKEN_EQUAL] = ParseRule{(*Parser).binary, nil, PREC_EQUALITY}
	rules[tokentype.TOKEN_EQUAL_EQUAL] = ParseRule{(*Parser).binary, (*Parser).binary, PREC_COMPARISON}
	rules[tokentype.TOKEN_GREATER] = ParseRule{(*Parser).binary, (*Parser).binary, PREC_COMPARISON}
	rules[tokentype.TOKEN_GREATER_EQUAL] = ParseRule{(*Parser).binary, (*Parser).binary, PREC_COMPARISON}
	rules[tokentype.TOKEN_VAR] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_LESS] = ParseRule{nil, (*Parser).binary, PREC_COMPARISON}
	rules[tokentype.TOKEN_LESS_EQUAL] = ParseRule{nil, (*Parser).binary, PREC_COMPARISON}
	rules[tokentype.TOKEN_IDENTIFIER] = ParseRule{(*Parser).variable, nil, PREC_NONE}
	rules[tokentype.TOKEN_STRING] = ParseRule{(*Parser).stringg, nil, PREC_NONE}
	rules[tokentype.TOKEN_NUMBER] = ParseRule{(*Parser).number, nil, PREC_NONE}
	rules[tokentype.TOKEN_AND] = ParseRule{nil, (*Parser).and, PREC_AND}
	rules[tokentype.TOKEN_ELSE] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_FALSE] = ParseRule{(*Parser).literal, nil, PREC_NONE}
	rules[tokentype.TOKEN_FOR] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_FUN] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_IF] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_NIL] = ParseRule{(*Parser).literal, nil, PREC_NONE}
	rules[tokentype.TOKEN_OR] = ParseRule{nil, (*Parser).or, PREC_OR}
	rules[tokentype.TOKEN_PRINT] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_RETURN] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_THIS] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_TRUE] = ParseRule{(*Parser).literal, nil, PREC_NONE}
	rules[tokentype.TOKEN_WHILE] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_ERROR] = ParseRule{nil, nil, PREC_NONE}
	rules[tokentype.TOKEN_EOF] = ParseRule{nil, nil, PREC_NONE}
}

func (parser *Parser) errorAtCurrent(msg string) {
	parser.errorAt(&parser.current, msg)
}

func (parser *Parser) advance() {
	parser.previous = parser.current

	for {
		parser.current = parser.scanner.ScanToken()
		if parser.current.Type != tokentype.TOKEN_ERROR {
			break
		}

		parser.errorAtCurrent(parser.current.Lexeme)
	}
}

func (parser *Parser) consume(typee tokentype.TokenType, msg string) {
	if parser.current.Type == typee {
		parser.advance()
		return
	}

	parser.errorAtCurrent(msg)
}

func (parser *Parser) check(typee tokentype.TokenType) bool {
	return parser.current.Type == typee
}

func (parser *Parser) match(typee tokentype.TokenType) bool {
	if !parser.check(typee) {
		return false
	}
	parser.advance()
	return true
}

func (parser *Parser) error(msg string) {
	parser.errorAt(&parser.previous, msg)
}

func (parser *Parser) errorAt(token *token.Token, msg string) {
	if parser.panicMode {
		return
	}
	parser.panicMode = true

	fmt.Fprintf(os.Stderr, "[line %d] Error", token.Line)

	if token.Type == tokentype.TOKEN_EOF {
		fmt.Fprintf(os.Stderr, " at end")
	} else if token.Type == tokentype.TOKEN_ERROR {
		// nothing
	} else {
		fmt.Fprintf(os.Stderr, " at '%s'", token.Lexeme)
	}

	fmt.Fprintf(os.Stderr, ": %s\n", msg)
}

func (parser *Parser) currentChunk() *chunk.Chunk {
	return parser.compiler.function.Chunk.(*chunk.Chunk)
}

func (parser *Parser) emitByte(b uint8) {
	parser.currentChunk().Write(b, parser.previous.Line)
}

func (parser *Parser) emitBytes(b1 uint8, b2 uint8) {
	parser.emitByte(b1)
	parser.emitByte(b2)
}

func (parser *Parser) emitReturn() {
	parser.emitByte(opcode.OP_NIL)
	parser.emitByte(opcode.OP_RETURN)
}

func (parser *Parser) makeConstant(val value.Value) uint8 {
	constIndex := parser.currentChunk().AddConstant(val)
	if constIndex > 256 {
		parser.error("Too many constants in one chunk.")
		return 0
	}

	return uint8(constIndex)
}

func (parser *Parser) emitConstant(val value.Value) {
	parser.emitBytes(opcode.OP_CONSTANT, parser.makeConstant(val))
}

func (parser *Parser) endCompiler() *value.ObjFunction {
	parser.emitReturn()

	funcName := parser.compiler.function.Name.String

	if len(funcName) == 0 {
		funcName = "script"
	}

	if config.DEBUG_PRINT_CODE {
		debug.DisassembleChunk(parser.compiler.function.Chunk.(*chunk.Chunk), funcName)
	}

	function := parser.compiler.function
	parser.compiler = parser.compiler.enclosing
	return function
}

func (parser *Parser) or(_ bool) {
	elseJump := parser.emitJump(opcode.OP_JUMP_IF_FALSE)
	endJump := parser.emitJump(opcode.OP_JUMP)

	parser.patchJump(elseJump)
	parser.emitByte(opcode.OP_POP)

	parser.parsePrecedence(PREC_OR)
	parser.patchJump(endJump)
}

func (parser *Parser) and(_ bool) {
	endJump := parser.emitJump(opcode.OP_JUMP_IF_FALSE)
	parser.emitByte(opcode.OP_POP)
	parser.parsePrecedence(PREC_AND)
	parser.patchJump(endJump)
}

func (parser *Parser) number(_ bool) {
	numStr := parser.previous.Lexeme
	val, _ := strconv.ParseFloat(numStr, len(numStr))
	parser.emitConstant(value.ValNumber(val))
}

func (parser *Parser) stringg(_ bool) {
	lexeme := parser.previous.Lexeme
	parser.emitConstant(value.ValObjString(lexeme[1 : len(lexeme)-1]))
}

func (parser *Parser) variable(canAssign bool) {
	parser.namedVariable(&parser.previous, canAssign)

}

func (parser *Parser) literal(_ bool) {
	switch parser.previous.Type {
	case tokentype.TOKEN_FALSE:
		parser.emitByte(opcode.OP_FALSE)
	case tokentype.TOKEN_TRUE:
		parser.emitByte(opcode.OP_TRUE)
	case tokentype.TOKEN_NIL:
		parser.emitByte(opcode.OP_NIL)
	}
}

func (parser *Parser) grouping(_ bool) {
	parser.expression()
	parser.consume(tokentype.TOKEN_RIGHT_PAREN, "Expect ')' after expression.")
}

func (parser *Parser) call(_ bool) {
	argCount := parser.argumentList()
	parser.emitBytes(opcode.OP_CALL, argCount)
}

func (parser *Parser) unary(_ bool) {
	operatorType := parser.previous.Type

	parser.expression()

	switch operatorType {
	case tokentype.TOKEN_BANG:
		parser.emitByte(opcode.OP_NOT)
	case tokentype.TOKEN_MINUS:
		parser.emitByte(opcode.OP_NEGATE)
	}
}

func (parser *Parser) binary(_ bool) {
	operatorType := parser.previous.Type
	rule := rules[operatorType]
	parser.parsePrecedence(rule.precedence + 1)

	switch operatorType {
	case tokentype.TOKEN_PLUS:
		parser.emitByte(opcode.OP_ADD)
	case tokentype.TOKEN_MINUS:
		parser.emitByte(opcode.OP_SUBTRACT)
	case tokentype.TOKEN_STAR:
		parser.emitByte(opcode.OP_MULTIPLY)
	case tokentype.TOKEN_SLASH:
		parser.emitByte(opcode.OP_DIVIDE)
	case tokentype.TOKEN_EQUAL_EQUAL:
		parser.emitByte(opcode.OP_EQUAL)
	case tokentype.TOKEN_BANG_EQUAL:
		parser.emitBytes(opcode.OP_EQUAL, opcode.OP_NOT)
	case tokentype.TOKEN_GREATER:
		parser.emitByte(opcode.OP_GREATER)
	case tokentype.TOKEN_GREATER_EQUAL:
		parser.emitBytes(opcode.OP_LESS, opcode.OP_NOT)
	case tokentype.TOKEN_LESS:
		parser.emitByte(opcode.OP_LESS)
	case tokentype.TOKEN_LESS_EQUAL:
		parser.emitBytes(opcode.OP_GREATER, opcode.OP_NOT)
	}
}

func (parser *Parser) list(_ bool) {
	count := 0
	if !parser.check(tokentype.TOKEN_RIGHT_BRACKET) {
		for ok := true; ok; ok = parser.match(tokentype.TOKEN_COMMA) {
			if parser.check(tokentype.TOKEN_RIGHT_BRACKET) {
				break // trailing comma case
			}

			parser.parsePrecedence(PREC_OR)

			if count == 256 {
				parser.error("Cannot have more than 256 items in a list literal.")
			}

			count++
		}
	}

	parser.consume(tokentype.TOKEN_RIGHT_BRACKET, "Expect ']' after list literal.")

	parser.emitBytes(opcode.OP_LIST, uint8(count))
}

func (parser *Parser) subscr(canAssign bool) {
	parser.parsePrecedence(PREC_OR)
	parser.consume(tokentype.TOKEN_RIGHT_BRACKET, "Expect ']' after index.")

	if canAssign && parser.match(tokentype.TOKEN_EQUAL) {
		parser.expression()
		parser.emitByte(opcode.OP_STORE)
	} else {
		parser.emitByte(opcode.OP_INDEX)
	}
}

func (parser *Parser) argumentList() uint8 {
	argCount := 0
	if !parser.check(tokentype.TOKEN_RIGHT_PAREN) {
		for ok := true; ok; ok = parser.match(tokentype.TOKEN_COMMA) {
			parser.expression()
			if argCount == 255 {
				parser.error("Can't have more than 255 arguments.")
			}
			argCount++
		}
	}

	parser.consume(tokentype.TOKEN_RIGHT_PAREN, "Expect ')' after arguments.")
	return uint8(argCount)
}

func (parser *Parser) parsePrecedence(prec Precedence) {
	parser.advance()
	prefixRule := rules[parser.previous.Type].prefix

	if prefixRule == nil {
		parser.error("Expect expression.")
		return
	}

	canAssign := prec <= PREC_ASSIGNMENT
	prefixRule(parser, canAssign)

	for prec <= rules[parser.current.Type].precedence {
		parser.advance()
		infixRule := rules[parser.previous.Type].infix
		infixRule(parser, canAssign)
	}
}

func (parser *Parser) expression() {
	parser.parsePrecedence(PREC_ASSIGNMENT)
}

func (parser *Parser) block() {
	for !parser.check(tokentype.TOKEN_RIGHT_BRACE) && !parser.check(tokentype.TOKEN_EOF) {
		parser.declaration()
	}

	parser.consume(tokentype.TOKEN_RIGHT_BRACE, "Expect '}' after block.")
}

func (parser *Parser) expressionStatement() {
	parser.expression()
	parser.consume(tokentype.TOKEN_SEMICOLON, "Expect ';' after expression.")
	parser.emitByte(opcode.OP_POP)
}

func (parser *Parser) printStatement() {
	parser.expression()
	parser.consume(tokentype.TOKEN_SEMICOLON, "Expect ';' after value.")
	parser.emitByte(opcode.OP_PRINT)
}

func (parser *Parser) emitJump(instruction uint8) int {
	parser.emitByte(instruction)
	parser.emitByte(0xff)
	parser.emitByte(0xff)
	return len(parser.currentChunk().Code) - 2
}

func (parser *Parser) patchJump(offset int) {
	jump := len(parser.currentChunk().Code) - offset - 2

	if jump > math.MaxUint16 {
		parser.error("Too much code to jump over.")
	}

	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, uint16(jump))
	parser.currentChunk().Code[offset] = bytes[0]
	parser.currentChunk().Code[offset+1] = bytes[1]
}

func (parser *Parser) ifStatement() {
	parser.consume(tokentype.TOKEN_LEFT_PAREN, "Expect '(' after 'if'.")
	parser.expression()
	parser.consume(tokentype.TOKEN_RIGHT_PAREN, "Expect ')' after condition.")

	thenJump := parser.emitJump(opcode.OP_JUMP_IF_FALSE)
	parser.emitByte(opcode.OP_POP)

	//then branch
	parser.statement()

	elseJump := parser.emitJump(opcode.OP_JUMP)
	parser.patchJump(thenJump)

	//else branch
	parser.emitByte(opcode.OP_POP)
	if parser.match(tokentype.TOKEN_ELSE) {
		parser.statement()
	}

	parser.patchJump(elseJump)
}

func (parser *Parser) emitLoop(loopStart int) {
	parser.emitByte(opcode.OP_LOOP)

	offset := len(parser.currentChunk().Code) - loopStart + 2
	if offset > math.MaxUint16 {
		parser.error("Loop body too large.")
	}

	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, uint16(offset))
	parser.emitBytes(bytes[0], bytes[1])
}

func (parser *Parser) whileStatement() {
	loopStart := len(parser.currentChunk().Code)
	parser.consume(tokentype.TOKEN_LEFT_PAREN, "Expect '(' after while.")
	parser.expression()
	parser.consume(tokentype.TOKEN_RIGHT_PAREN, "Expect ')' after while.")

	exitJump := parser.emitJump(opcode.OP_JUMP_IF_FALSE)
	parser.emitByte(opcode.OP_POP)
	parser.statement()
	parser.emitLoop(loopStart)

	parser.patchJump(exitJump)
	parser.emitByte(opcode.OP_POP)
}

func (parser *Parser) forStatement() {
	parser.beginScope()
	parser.consume(tokentype.TOKEN_LEFT_PAREN, "Expect '(' after 'for'.")

	if parser.match(tokentype.TOKEN_VAR) {
		parser.varDeclaration()
	} else {
		if parser.match(tokentype.TOKEN_IDENTIFIER) {
			parser.variable(true)
		}
		parser.consume(tokentype.TOKEN_SEMICOLON, "Expect ';' after loop initializer.")
	}

	loopStart := len(parser.currentChunk().Code)
	exitJump := -1

	if !parser.match(tokentype.TOKEN_SEMICOLON) {
		parser.expression()
		parser.consume(tokentype.TOKEN_SEMICOLON, "Expect ';' after loop condition.")

		exitJump = parser.emitJump(opcode.OP_JUMP_IF_FALSE)
		parser.emitByte(opcode.OP_POP)
	}

	if !parser.match(tokentype.TOKEN_RIGHT_PAREN) {
		bodyJump := parser.emitJump(opcode.OP_JUMP)
		incrementStart := len(parser.currentChunk().Code)
		parser.expression()
		parser.emitByte(opcode.OP_POP)
		parser.consume(tokentype.TOKEN_RIGHT_PAREN, "Expect ')' after for clauses.")

		parser.emitLoop(loopStart)
		loopStart = incrementStart
		parser.patchJump(bodyJump)
	}

	parser.statement()
	parser.emitLoop(loopStart)

	if exitJump != -1 {
		parser.patchJump(exitJump)
		parser.emitByte(opcode.OP_POP)
	}

	parser.endScope()
}

func (parser *Parser) returnStatement() {
	if parser.compiler.funcType == functype.TYPE_SCRIPT {
		parser.error("Can't return from top-level code.")
	}

	if parser.match(tokentype.TOKEN_SEMICOLON) {
		parser.emitReturn()
	} else {
		parser.expression()
		parser.consume(tokentype.TOKEN_SEMICOLON, "Expect ';' after return value.")
		parser.emitByte(opcode.OP_RETURN)
	}
}

func (parser *Parser) statement() (breakJump, continueJump int) {
	breakJump, continueJump = -1, -1

	if parser.match(tokentype.TOKEN_PRINT) {
		parser.printStatement()
	} else if parser.match(tokentype.TOKEN_LEFT_BRACE) {
		parser.beginScope()
		parser.block()
		parser.endScope()
	} else if parser.match(tokentype.TOKEN_IF) {
		parser.ifStatement()
	} else if parser.match(tokentype.TOKEN_WHILE) {
		parser.whileStatement()
	} else if parser.match(tokentype.TOKEN_FOR) {
		parser.forStatement()
	} else if parser.match(tokentype.TOKEN_RETURN) {
		parser.returnStatement()
	} else {
		parser.expressionStatement()
	}

	return
}

func (parser *Parser) function() {
	compiler := parser.initCompiler(functype.TYPE_FUNCTION)
	parser.beginScope()

	compiler.function.Name = value.ValObjString(parser.previous.Lexeme).AsString()

	parser.consume(tokentype.TOKEN_LEFT_PAREN, "Expect '(' after function name.")

	if !parser.check(tokentype.TOKEN_RIGHT_PAREN) {
		for ok := true; ok; ok = parser.match(tokentype.TOKEN_COMMA) {
			compiler.function.Arity++
			if compiler.function.Arity > 255 {
				parser.errorAtCurrent("Can't have more than 255 parameters.")
			}
			arg := parser.parseVariable("Expect parameter name.")
			parser.defineVaraible(arg)
		}
	}

	parser.consume(tokentype.TOKEN_RIGHT_PAREN, "Expect ')' after parameters.")
	parser.consume(tokentype.TOKEN_LEFT_BRACE, "Expect '{' before function body.")
	parser.block()

	function := parser.endCompiler()
	funcConstIndex := parser.makeConstant(value.ValObjFunction(function))
	parser.emitBytes(opcode.OP_CLOSURE, funcConstIndex)

	for i := 0; i < function.UpvalueCount; i++ {
		if compiler.upvalues[i].isLocal {
			parser.emitByte(1)
		} else {
			fmt.Println("emiting upvalue")
			parser.emitByte(0)
		}
		parser.emitByte(compiler.upvalues[i].index)
	}
}

func (parser *Parser) beginScope() {
	parser.compiler.scopeDepth++
}

func (parser *Parser) endScope() {
	parser.compiler.scopeDepth--

	localCount := len(parser.compiler.locals)
	for localCount > 0 && parser.compiler.locals[localCount-1].depth > parser.compiler.scopeDepth {
		parser.emitByte(opcode.OP_POP)
		parser.compiler.locals = parser.compiler.locals[:localCount-1]
		localCount = len(parser.compiler.locals)
	}
}

func (parser *Parser) addLocal(name token.Token) {
	if len(parser.compiler.locals) > math.MaxUint8 {
		parser.error("Too many local variables in function.")
	}

	local := Local{name: name, depth: -1}
	parser.compiler.locals = append(parser.compiler.locals, local)
}

func (parser *Parser) resolveLocal(compiler *Compiler, name *token.Token) int {
	localCount := len(compiler.locals)
	for i := localCount - 1; i >= 0; i-- {
		local := &compiler.locals[i]
		if local.name.Lexeme == name.Lexeme {
			if local.depth == -1 {
				parser.error("Can't read local variable in its own initializer.")
			}
			return i
		}
	}

	return -1
}

func (parser *Parser) addUpvalue(compiler *Compiler, index uint8, isLocal bool) int {
	upvalueCount := compiler.function.UpvalueCount

	for i := 0; i < upvalueCount; i++ {
		upvalue := compiler.upvalues[i]
		if upvalue.index == index && upvalue.isLocal == isLocal {
			return i
		}
	}

	if upvalueCount == 256 {
		parser.error("Too many closure variables in function.")
		return 0
	}

	compiler.upvalues[upvalueCount].isLocal = isLocal
	compiler.upvalues[upvalueCount].index = index

	compiler.function.UpvalueCount++
	return compiler.function.UpvalueCount - 1
}

func (parser *Parser) resolveUpvalue(compiler *Compiler, name *token.Token) int {
	if compiler.enclosing == nil {
		return -1
	}

	resolved := -1

	local := parser.resolveLocal(compiler.enclosing, name)
	if local != -1 {
		resolved = parser.addUpvalue(compiler, uint8(local), true)
	}

	upvalue := parser.resolveUpvalue(compiler.enclosing, name)
	if upvalue != -1 {
		resolved = parser.addUpvalue(compiler, uint8(upvalue), false)
	}

	return resolved
}

func (parser *Parser) namedVariable(name *token.Token, canAssign bool) {
	var getOp, setOp uint8

	arg := parser.resolveLocal(parser.compiler, name)
	if arg != -1 {
		getOp = opcode.OP_GET_LOCAL
		setOp = opcode.OP_SET_LOCAL
	} else {
		arg = parser.resolveUpvalue(parser.compiler, name)
		if arg != -1 {
			getOp = opcode.OP_GET_UPVALUE
			setOp = opcode.OP_SET_UPVALUE
		} else {
			arg = int(parser.identifierConstant(name))
			getOp = opcode.OP_GET_GLOBAL
			setOp = opcode.OP_SET_GLOBAL
		}
	}

	if canAssign && parser.match(tokentype.TOKEN_EQUAL) {
		parser.expression()
		parser.emitBytes(setOp, uint8(arg))
	} else if canAssign && parser.match(tokentype.TOKEN_MINUS_MINUS) {
		parser.emitBytes(getOp, uint8(arg))
		parser.emitBytes(getOp, uint8(arg))
		one := parser.makeConstant(value.ValNumber(1))
		parser.emitBytes(opcode.OP_CONSTANT, one)
		parser.emitByte(opcode.OP_SUBTRACT)
		parser.emitBytes(setOp, uint8(arg))
		parser.emitByte(opcode.OP_POP)
	} else if canAssign && parser.match(tokentype.TOKEN_PLUS_PLUS) {
		parser.emitBytes(getOp, uint8(arg))
		parser.emitBytes(getOp, uint8(arg))
		one := parser.makeConstant(value.ValNumber(1))
		parser.emitBytes(opcode.OP_CONSTANT, one)
		parser.emitByte(opcode.OP_ADD)
		parser.emitBytes(setOp, uint8(arg))
		parser.emitByte(opcode.OP_POP)
	} else {
		parser.emitBytes(getOp, uint8(arg))
	}
}

func (parser *Parser) markInitialized() {
	if parser.compiler.scopeDepth == 0 {
		return
	}

	localCount := len(parser.compiler.locals)
	parser.compiler.locals[localCount-1].depth = parser.compiler.scopeDepth
}

func (parser *Parser) defineVaraible(global uint8) {
	if parser.compiler.scopeDepth > 0 {
		parser.markInitialized()
		return
	}

	parser.emitBytes(opcode.OP_DEFINE_GLOBAL, global)
}

func (parser *Parser) declareVariable() {
	if parser.compiler.scopeDepth == 0 {
		return
	}

	name := &parser.previous

	for i := len(parser.compiler.locals) - 1; i >= 0; i-- {
		local := parser.compiler.locals[i]
		if local.depth != -1 && local.depth > parser.compiler.scopeDepth {
			break
		}

		if local.name.Lexeme == name.Lexeme {
			parser.error("Already a variable with this name in this scope.")
		}
	}

	parser.addLocal(*name)
}

func (parser *Parser) identifierConstant(name *token.Token) uint8 {
	return parser.makeConstant(value.ValObjString(name.Lexeme))
}

func (parser *Parser) parseVariable(err string) uint8 {
	parser.consume(tokentype.TOKEN_IDENTIFIER, err)

	parser.declareVariable()
	if parser.compiler.scopeDepth > 0 {
		return 0
	}

	return parser.identifierConstant(&parser.previous)
}

func (parser *Parser) varDeclaration() {
	global := parser.parseVariable("Expect variable name.")

	if parser.match(tokentype.TOKEN_EQUAL) {
		parser.expression()
	} else {
		parser.emitByte(opcode.OP_NIL)
	}

	parser.consume(tokentype.TOKEN_SEMICOLON, "Expect ; after variable declaration")
	parser.defineVaraible(global)
}

func (parser *Parser) funDeclaration() {
	global := parser.parseVariable("Expect function name.")
	parser.markInitialized()
	parser.function()
	parser.defineVaraible(global)
}

func (parser *Parser) declaration() {
	if parser.match(tokentype.TOKEN_VAR) {
		parser.varDeclaration()
	} else if parser.match(tokentype.TOKEN_FUN) {
		parser.funDeclaration()
	} else {
		parser.statement()
	}

	if parser.panicMode {
		parser.synchronize()
	}
}

func (parser *Parser) synchronize() {
	parser.panicMode = false
	for parser.current.Type != tokentype.TOKEN_EOF {
		switch parser.current.Type {
		case tokentype.TOKEN_FUN:
			return
		case tokentype.TOKEN_VAR:
			return
		case tokentype.TOKEN_FOR:
			return
		case tokentype.TOKEN_IF:
			return
		case tokentype.TOKEN_WHILE:
			return
		case tokentype.TOKEN_PRINT:
			return
		case tokentype.TOKEN_RETURN:
			return
		}

		parser.advance()
	}
}

func (parser *Parser) initCompiler(funcType functype.FuncType) *Compiler {
	compiler := new(Compiler)
	compiler.enclosing = parser.compiler
	compiler.function = value.NewObjFunction(new(chunk.Chunk))
	compiler.funcType = funcType
	compiler.scopeDepth = 0
	compiler.locals = make([]Local, 0)

	if compiler.funcType == functype.TYPE_SCRIPT {
		compiler.function.Name = value.ValObjString("<script>").AsString()
	}

	parser.compiler = compiler

	if funcType != functype.TYPE_SCRIPT {
		compiler.function.Name = value.ValObjString(parser.previous.Lexeme).AsString()
	}

	local := Local{depth: 0, name: token.Token{Lexeme: ""}}
	compiler.locals = append(compiler.locals, local)

	return compiler
}

func Compile(source *string) *value.ObjFunction {
	var scanner scanner.Scanner
	scanner.Init(source)

	parser := new(Parser)
	parser.hadError = false
	parser.panicMode = false
	parser.scanner = &scanner
	parser.compiler = parser.initCompiler(functype.TYPE_SCRIPT)

	initRules()

	parser.advance()

	for !parser.match(tokentype.TOKEN_EOF) {
		parser.declaration()
	}

	parser.consume(tokentype.TOKEN_EOF, "Expect end of expression.")

	function := parser.endCompiler()

	if parser.hadError {
		return nil
	}

	return function
}
