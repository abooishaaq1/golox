package debug

import (
	"encoding/binary"
	"fmt"
	"golox/chunk"
	"golox/chunk/opcode"
)

func DisassembleChunk(c *chunk.Chunk, name string) {
	fmt.Printf("\n==== %s ====\n\n", name)

	for i := 0; i < len(c.Code); {
		i = DisassembleInstruction(c, i)
	}
}

func simpleInstruction(name string, offset int) int {
	fmt.Printf("%s\n", name)
	return offset + 1
}

func constantInstruction(name string, chunk *chunk.Chunk, offset int) int {
	constIndex := chunk.Code[offset+1]
	fmt.Printf("%-16s %4d ", name, constIndex)
	chunk.Constants[constIndex].Print()
	fmt.Println()
	return offset + 2
}

func byteInstruction(name string, chunk *chunk.Chunk, offset int) int {
	slot := chunk.Code[offset+1]
	fmt.Printf("%-16s %4d\n", name, slot)
	return offset + 2
}

func jumpInstruction(name string, sign int, chunk *chunk.Chunk, offset int) int {
	bytes := make([]byte, 2)
	bytes[0] = chunk.Code[offset+1]
	bytes[1] = chunk.Code[offset+2]
	jump := binary.LittleEndian.Uint16(bytes)
	fmt.Printf("%-16s %4d -> %d\n", name, offset, offset+3+sign*int(jump))
	return offset + 3
}

func DisassembleInstruction(chunk *chunk.Chunk, offset int) int {
	fmt.Printf("%04d ", offset)
	if offset > 0 && chunk.Lines[offset] == chunk.Lines[offset-1] {
		fmt.Printf("   | ")
	} else {
		fmt.Printf("%4d ", chunk.Lines[offset])
	}
	instruction := chunk.Code[offset]

	switch instruction {
	case opcode.OP_CONSTANT:
		return constantInstruction("OP_CONSTANT", chunk, offset)
	case opcode.OP_ADD:
		return simpleInstruction("OP_ADD", offset)
	case opcode.OP_SUBTRACT:
		return simpleInstruction("OP_SUBTRACT", offset)
	case opcode.OP_MULTIPLY:
		return simpleInstruction("OP_MULTIPLY", offset)
	case opcode.OP_DIVIDE:
		return simpleInstruction("OP_DIVIDE", offset)
	case opcode.OP_NEGATE:
		return simpleInstruction("OP_NEGATE", offset)
	case opcode.OP_RETURN:
		return simpleInstruction("OP_RETURN", offset)
	case opcode.OP_PRINT:
		return simpleInstruction("OP_PRINT", offset)
	case opcode.OP_NIL:
		return simpleInstruction("OP_NIL", offset)
	case opcode.OP_TRUE:
		return simpleInstruction("OP_TRUE", offset)
	case opcode.OP_FALSE:
		return simpleInstruction("OP_FALSE", offset)
	case opcode.OP_NOT:
		return simpleInstruction("OP_NOT", offset)
	case opcode.OP_POP:
		return simpleInstruction("OP_POP", offset)
	case opcode.OP_EQUAL:
		return simpleInstruction("OP_EQUAL", offset)
	case opcode.OP_LESS:
		return simpleInstruction("OP_LESS", offset)
	case opcode.OP_GREATER:
		return simpleInstruction("OP_GREATER", offset)
	case opcode.OP_DEFINE_GLOBAL:
		return constantInstruction("OP_DEFINE_GLOBAL", chunk, offset)
	case opcode.OP_GET_GLOBAL:
		return constantInstruction("OP_GET_GLOBAL", chunk, offset)
	case opcode.OP_SET_GLOBAL:
		return constantInstruction("OP_SET_GLOBAL", chunk, offset)
	case opcode.OP_CALL:
		return byteInstruction("OP_CALL", chunk, offset)
	case opcode.OP_GET_LOCAL:
		return byteInstruction("OP_GET_LOCAL", chunk, offset)
	case opcode.OP_SET_LOCAL:
		return byteInstruction("OP_SET_LOCAL", chunk, offset)
	case opcode.OP_GET_LOCAL_2:
		return byteInstruction("OP_GET_LOCAL_2", chunk, offset)
	case opcode.OP_SET_LOCAL_2:
		return byteInstruction("OP_SET_LOCAL_2", chunk, offset)
	case opcode.OP_LIST:
		return byteInstruction("OP_LIST", chunk, offset)
	case opcode.OP_STORE:
		return simpleInstruction("OP_STORE", offset)
	case opcode.OP_INDEX:
		return simpleInstruction("OP_INDEX", offset)
	case opcode.OP_JUMP:
		return jumpInstruction("OP_JUMP", 1, chunk, offset)
	case opcode.OP_JUMP_IF_FALSE:
		return jumpInstruction("OP_JUMP_IF_FALSE", 1, chunk, offset)
	case opcode.OP_LOOP:
		return jumpInstruction("OP_LOOP", -1, chunk, offset)
	case opcode.OP_CLOSURE:
		offset++
		constIndex := chunk.Code[offset]
		fmt.Printf("%-16s %4d ", "OP_CLOSURE", constIndex)
		funcVal := chunk.Constants[constIndex]
		funcVal.Print()
		function := funcVal.AsObjFunction()
		fmt.Println()
		for i := 0; i < function.UpvalueCount; i++ {
			isLocal := chunk.Code[offset+1]
			index := chunk.Code[offset+2]
			varDesc := ""
			if isLocal == 1 {
				varDesc = "local"
			} else {
				varDesc = "upvalue"
			}
			fmt.Printf("%04d      |                     %s %d\n", offset, varDesc, index)
			offset += 2
		}
		return offset + 1
	case opcode.OP_GET_UPVALUE:
		return byteInstruction("OP_GET_UPVALUE", chunk, offset)
	case opcode.OP_SET_UPVALUE:
		return byteInstruction("OP_SET_UPVALUE", chunk, offset)
	}

	fmt.Printf("Unknown opcode %d\n", instruction)
	return offset + 1
}
