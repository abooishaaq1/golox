package opcode

const (
	OP_CONSTANT      uint8 = iota
	OP_NEGATE        uint8 = iota
	OP_ADD           uint8 = iota
	OP_SUBTRACT      uint8 = iota
	OP_MULTIPLY      uint8 = iota
	OP_DIVIDE        uint8 = iota
	OP_RETURN        uint8 = iota
	OP_NIL           uint8 = iota
	OP_TRUE          uint8 = iota
	OP_FALSE         uint8 = iota
	OP_PRINT         uint8 = iota
	OP_NOT           uint8 = iota
	OP_POP           uint8 = iota
	OP_EQUAL         uint8 = iota
	OP_GREATER       uint8 = iota
	OP_LESS          uint8 = iota
	OP_DEFINE_GLOBAL uint8 = iota
	OP_GET_GLOBAL    uint8 = iota
	OP_SET_GLOBAL    uint8 = iota
	OP_GET_LOCAL     uint8 = iota
	OP_SET_LOCAL     uint8 = iota
	OP_GET_LOCAL_2   uint8 = iota
	OP_SET_LOCAL_2   uint8 = iota
	OP_JUMP          uint8 = iota
	OP_JUMP_IF_FALSE uint8 = iota
	OP_LOOP          uint8 = iota
	OP_CALL          uint8 = iota
	OP_LIST          uint8 = iota
	OP_STORE         uint8 = iota
	OP_INDEX         uint8 = iota
	OP_CLOSURE       uint8 = iota
	OP_GET_UPVALUE   uint8 = iota
	OP_SET_UPVALUE   uint8 = iota
)
