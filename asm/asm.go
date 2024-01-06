package asm

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

/*
#include <sys/mman.h>
#include <unistd.h>
#include <stdint.h>
#include <string.h>

void* alloc_mem_exec(size_t length, uint8_t* code) {
   void* mem = mmap(0, length, PROT_READ | PROT_WRITE | PROT_EXEC, MAP_ANON | MAP_PRIVATE, -1, 0);
   memcpy(mem, code, length);
   mprotect(mem, length, PROT_READ | PROT_EXEC);
   return mem;
}

int run_mem_exec(void* mem) {
	int (*f)() = mem;
	int res = f();
	return res;
}
*/
import "C"

type Reg uint8

const (
	RAX Reg = iota
	RCX
	RDX
	RBX
	RSP
	RBP
	RSI
	RDI
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15
)

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// Opcode      Instruction     Op/  64-bit Compat/   Description
//                             En   Mode   Leg Mode
// REX.W+03/r  ADD r64,r/m64   RM   Valid  N.E.      Add r/m64 to r64.

type X86_64 struct {
	REX   []uint8 // 1 byte // 0100WRXB
	Op    []uint8 // 1-3 bytes
	ModRm []uint8 // 1 byte
	SIB   []uint8 // 1 byte
	Disp  []uint8 // 1, 2, or 4 bytes
	Imm   []uint8 // 1, 2, or 4 bytes
}

func (x *X86_64) Encode() []uint8 {
	var out []uint8
	out = append(out, x.REX...)
	out = append(out, x.Op...)
	out = append(out, x.ModRm...)
	out = append(out, x.SIB...)
	out = append(out, x.Disp...)
	out = append(out, x.Imm...)
	return out
}

func (x *X86_64) Init() {
	x.REX = []uint8{0b01000000}
}

func (x *X86_64) SetRex_W(w uint8) {
	x.REX[0] |= w << 3
}

func (x *X86_64) SetRex_R(r uint8) {
	x.REX[0] |= r << 2
}

func (x *X86_64) SetRex_X(x_ uint8) {
	x.REX[0] |= x_ << 1
}

func (x *X86_64) setOp(opCode uint8) {
	x.Op = []uint8{opCode}
}

func (x *X86_64) setOp2(op1, op2 uint8) {
	x.Op = []uint8{op1, op2}
}

func (x *X86_64) setModRm(dst, src Reg, mode uint8) {
	x.ModRm = []uint8{mode | uint8(dst)<<3 | uint8(src)}
}

func (x *X86_64) setImm64(imm uint64) {
	x.Imm = []uint8{uint8(imm), uint8(imm >> 8), uint8(imm >> 16), uint8(imm >> 24)}
}

func (x *X86_64) setImm32(imm int) {
	x.Imm = make([]uint8, 4)
	binary.LittleEndian.PutUint32(x.Imm, uint32(imm))
}

func (x *X86_64) setImm16(imm int16) {
	x.Imm = []uint8{uint8(imm), uint8(imm >> 8)}
}

func (x *X86_64) setImm8(imm int8) {
	x.Imm = []uint8{uint8(imm)}
}

func (x *X86_64) AddRegReg(dst, src Reg) {
	x.Init()
	x.SetRex_W(1)
	x.setOp(0x03)
	x.setModRm(dst, src, 0xc0)
}

func (x *X86_64) SubRegReg(dst, src Reg) {
	x.Init()
	x.SetRex_W(1)
	x.setOp(0x2B)
	x.setModRm(dst, src, 0xc0)
}

func (x *X86_64) MulReg(dst Reg) {
	x.Init()
	x.SetRex_W(1)
	x.setOp(0xF7)
	x.setModRm(dst, 4, 0xE0)
}

func (x *X86_64) DivReg(dst Reg) {
	x.Init()
	x.SetRex_W(1)
	x.setOp(0xF7)
	x.setModRm(dst, 6, 0xE0)
}

func (x *X86_64) Jmp(target int32) {
	x.Init()
	x.setOp(0xE9)
	x.setImm64(uint64(target))
}

func (x *X86_64) Jle(target int) {
	x.setOp2(0x0F, 0x8E)
	x.setImm32(target)
}

func (x *X86_64) Ret() {
	x.setOp(0xC3)
}

func (x *X86_64) AddRegImm(dst Reg, imm int) {
	x.Init()
	x.SetRex_W(1)
	x.setOp(0x81)
	x.setModRm(0, dst, 0xc0)
	x.setImm32(imm)
}

func (x *X86_64) CmpRegImm(reg Reg, imm int) {
	// REX.W + 81 /7 id
	x.Init()
	x.SetRex_W(1)
	x.setOp(0x81)
	x.setModRm(7, reg, 0xc0)
	x.setImm32(imm)
}

func (x *X86_64) XorRegReg(dst, src Reg) {
	x.Init()
	x.SetRex_W(1)
	x.setOp(0x33)
	x.setModRm(dst, src, 0xc0)
}

func ExtendCode(code []uint8, x []X86_64) []uint8 {
	for _, insn := range x {
		code = append(code, insn.Encode()...)
	}
	return code
}

func Run(code []uint8) int {
	// Allocate memory
	mem := C.alloc_mem_exec(C.size_t(len(code)), (*C.uint8_t)(unsafe.Pointer(&code[0])))
	defer C.munmap(mem, C.size_t(len(code)))

	// Run
	res := C.run_mem_exec(mem)
	return int(res)
}

func Test() {
	// 1:
	//  xor rax, rax
	//  add rax, 1111
	//  ret
	x := make([]X86_64, 3)

	x[0].Init()
	x[0].XorRegReg(RAX, RAX)
	x[1].Init()
	x[1].AddRegImm(RAX, 1111)
	x[2].Init()
	x[2].Ret()
	codeBytes := ExtendCode(make([]uint8, 0), x)
	fmt.Println(Run(codeBytes))

	// 2:
	// xor rax, rax
	// label:
	// add rax, 1
	// cmp rax, 10
	// jne label
	// ret

	var y [6]X86_64
	y[0].XorRegReg(RAX, RAX)
	codeBytes = ExtendCode(make([]uint8, 0), y[:1])
	y[1].AddRegImm(RAX, 1)
	y[2].AddRegImm(RAX, 1)
	y[3].CmpRegImm(RAX, 10)
	loopBytes := y[1].Encode()
	loopBytes = append(loopBytes, y[2].Encode()...)
	// take in account the length of the loopBytes and the length of the jne instruction
	y[4].Jle(-len(loopBytes) - 6)
	y[5].Ret()
	codeBytes = append(codeBytes, loopBytes...)
	codeBytes = ExtendCode(codeBytes, y[3:])
	fmt.Println(Run(codeBytes))

	// 3:
	// xor rdi, rdi
	// xor rax, rax
	// label:
	// add rdi, 1
	// add rdi, 1
	// cmp rdi, 10
	// jne label
	// add rax, rdi
	// ret

	var z [8]X86_64
	z[0].XorRegReg(RDI, RDI)
	z[1].XorRegReg(RAX, RAX)
	codeBytes = ExtendCode(make([]uint8, 0), z[:2])
	z[2].AddRegImm(RDI, 1)
	z[3].AddRegImm(RDI, 1)
	z[4].CmpRegImm(RDI, 10)
	loopBytes = ExtendCode(make([]uint8, 0), z[2:5])
	z[5].Jle(-len(loopBytes) - 6)
	z[6].Init()
	z[6].AddRegReg(RAX, RDI)
	z[7].Ret()
	codeBytes = append(codeBytes, loopBytes...)
	codeBytes = ExtendCode(codeBytes, z[5:])
	fmt.Println(Run(codeBytes))
}
