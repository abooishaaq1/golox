package chunk

import "golox/value"

type Chunk struct {
	Code      []uint8
	Lines     []int
	Constants []value.Value
}

func (chunk *Chunk) Write(bits uint8, lines int) {
	chunk.Code = append(chunk.Code, bits)
	chunk.Lines = append(chunk.Lines, lines)
}

func (chunk *Chunk) AddConstant(constant value.Value) int {
	chunk.Constants = append(chunk.Constants, constant)
	return len(chunk.Constants) - 1
}
