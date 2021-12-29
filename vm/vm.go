package vm

import (
	"encoding/binary"
	"fmt"
	"golox/builtins"
	"golox/chunk"
	"golox/chunk/opcode"
	"golox/compiler"
	"golox/config"
	"golox/value"
	"golox/value/objtype"
	"golox/value/valuetype"
	"golox/vm/interpretresult"
	"os"
	"unsafe"
)

const (
	FRAMES_INITIAL_SIZE int = 128
	STACK_INITIAL_SIZE  int = FRAMES_INITIAL_SIZE * 256
)

type VM struct {
	stackTop     int
	stack        []value.Value
	frames       []CallFrame
	openUpvalues []*value.ObjUpvalue
	globals      map[string]value.Value
}

type CallFrame struct {
	slots   int
	ip      *byte
	closure *value.ObjClosure
}

func (vm *VM) resetStack() {
	vm.stackTop = 0
	vm.stack = make([]value.Value, STACK_INITIAL_SIZE)
	vm.frames = make([]CallFrame, 0, FRAMES_INITIAL_SIZE)
	vm.globals = make(map[string]value.Value)
}

func (vm *VM) initBuiltins() {
	vm.defineNative("clock", builtins.Clock)

	vm.defineNative("mod", builtins.Mod)

	vm.defineNative("list", builtins.List)
	vm.defineNative("append", builtins.Append)
	vm.defineNative("len", builtins.Len)
	vm.defineNative("pop", builtins.Pop)
}

func (vm *VM) Init() {
	vm.resetStack()
	vm.initBuiltins()
}

func (vm *VM) push(value value.Value) {
	vm.stack[vm.stackTop] = value
	vm.stackTop++
}

func (vm *VM) pop() value.Value {
	vm.stackTop--
	return vm.stack[vm.stackTop]
}

func (vm *VM) peek(distance int) value.Value {
	return vm.stack[vm.stackTop-distance-1]
}

func (vm *VM) readByte() uint8 {
	frame := &vm.frames[len(vm.frames)-1]
	instruction := *frame.ip
	frame.ip = incr(frame.ip, 1)
	return instruction
}

func (vm *VM) readTwoBytes() uint16 {
	frame := &vm.frames[len(vm.frames)-1]
	bytes := make([]uint8, 2)
	bytes[0] = *frame.ip
	frame.ip = incr(frame.ip, 1)
	bytes[1] = *frame.ip
	frame.ip = incr(frame.ip, 1)
	return binary.LittleEndian.Uint16(bytes)
}

func (vm *VM) readFourBytes() uint32 {
	frame := &vm.frames[len(vm.frames)-1]
	bytes := make([]uint8, 4)
	for i := 0; i < 4; i++ {
		bytes[i] = *frame.ip
		frame.ip = incr(frame.ip, 1)
	}
	return binary.LittleEndian.Uint32(bytes)
}

func (vm *VM) readConstant() value.Value {
	frame := &vm.frames[len(vm.frames)-1]
	chunk := frame.closure.Function.Chunk.(*chunk.Chunk)
	return chunk.Constants[vm.readByte()]
}

func incr(pointer *byte, steps int) *byte {
	addressHolder := uintptr(unsafe.Pointer(pointer))
	addressHolder = addressHolder + unsafe.Sizeof(*(pointer))*uintptr(steps)
	return (*byte)(unsafe.Pointer(addressHolder))
}

func decr(pointer *byte, steps int) *byte {
	addressHolder := uintptr(unsafe.Pointer(pointer))
	addressHolder = addressHolder - unsafe.Sizeof(*(pointer))*uintptr(steps)
	return (*byte)(unsafe.Pointer(addressHolder))
}

func diff(p1 *byte, p2 *byte) int {
	return int(uintptr(unsafe.Pointer(p1)) - uintptr(unsafe.Pointer(p2)))
}

func (vm *VM) runtimeError(err string) {
	fmt.Println(err)

	for i := len(vm.frames) - 1; i >= 0; i-- {
		frame := &vm.frames[i]
		function := frame.closure.Function
		// -1 because the IP is sitting on the next instruction to be
		// executed.
		chunk := frame.closure.Function.Chunk.(*chunk.Chunk)
		offset := diff(frame.ip, &((chunk.Code)[0]))
		line := (chunk.Lines)[offset]
		fmt.Fprintf(os.Stderr, "[line %d] in ", line)
		if function.Name == nil {
			fmt.Fprintf(os.Stderr, "script\n")
		} else {
			fmt.Fprintf(os.Stderr, "%s()\n", function.Name.String)
		}
	}

	vm.resetStack()
}

func (vm *VM) call(closure *value.ObjClosure, argCount int) bool {
	if argCount != closure.Function.Arity {
		vm.runtimeError(fmt.Sprintf("Expect %d arguments but got %d.", closure.Function.Arity, argCount))
		return false
	}

	if len(vm.frames) == FRAMES_INITIAL_SIZE {
		vm.runtimeError("Stack overflow.")
		return false
	}

	chunk := closure.Function.Chunk.(*chunk.Chunk)
	frame := CallFrame{closure: closure, ip: &((chunk.Code)[0]), slots: vm.stackTop - argCount - 1}
	vm.frames = append(vm.frames, frame)

	return true
}

func (vm *VM) callValue(callee value.Value, argCount int) bool {
	if callee.IsObj() {
		switch callee.AsObj().Type {
		case objtype.OBJ_FUNCTION:
			return vm.call(value.NewObjClosure(callee.AsObjFunction()), argCount)
		case objtype.OBJ_NATIVE:
			native := callee.AsNative()
			result, err := (native)(argCount, vm.stack[vm.stackTop-argCount:])
			if len(err) > 0 {
				vm.runtimeError(err)
				return false
			}
			vm.stackTop -= argCount + 1
			vm.push(result)
			return true
		case objtype.OBJ_CLOSURE:
			closure := callee.AsObjClosure()
			return vm.call(closure, argCount)
		}
	}

	vm.runtimeError("Can only call functions and classes.")
	return false
}

func (vm *VM) binaryOp(op rune) {
	if vm.peek(0).IsNumber() && vm.peek(1).IsNumber() {
		a := vm.pop().AsNumber()
		b := vm.pop().AsNumber()
		switch op {
		case '>':
			vm.push(value.ValBool(b > a))
		case '<':
			vm.push(value.ValBool(b < a))
		case '+':
			vm.push(value.ValNumber(b + a))
		case '-':
			vm.push(value.ValNumber(b - a))
		case '*':
			vm.push(value.ValNumber(b * a))
		case '/':
			vm.push(value.ValNumber(b / a))
		}
	} else {
		vm.runtimeError("Operands must be numbers.")
	}
}

func (vm *VM) defineNative(name string, function value.NativeFn) {
	vm.globals[name] = value.ValNative(function)
}

func (vm *VM) captureUpvalue(l *value.Value) *value.ObjUpvalue {
	for _, up := range vm.openUpvalues {
		if up.Location == l {
			return up
		}
	}

	upvalue := value.NewObjUpvalue(l)
	upvalue.Closed = *upvalue.Location
	upvalue.Location = &upvalue.Closed
	vm.openUpvalues = append(vm.openUpvalues, upvalue)

	return upvalue
}

func (vm *VM) run() interpretresult.InterpretResult {
	frame := &vm.frames[len(vm.frames)-1]

	for {
		if config.DEBUG_TRACE_EXECUTION {
			fmt.Printf("          ")
			for i := 0; i < vm.stackTop; i += 1 {
				fmt.Printf("[ ")
				vm.stack[i].Print()
				fmt.Printf(" %p", &vm.stack[i])
				fmt.Printf(" ]")
			}
			fmt.Println()
		}

		switch vm.readByte() {

		case opcode.OP_CONSTANT:
			constant := vm.readConstant()
			vm.push(constant)

		case opcode.OP_ADD:

			if vm.peek(0).IsString() && vm.peek(1).IsString() {
				a := vm.pop().AsGoString()
				b := vm.pop().AsGoString()
				vm.push(value.ValObjString(b + a))
			} else if vm.peek(1).IsString() {
				strfied := vm.pop().Stringify()
				str := vm.pop().AsGoString()
				vm.push(value.ValObjString(str + strfied))
			} else if vm.peek(0).IsString() {
				str := vm.pop().AsGoString()
				strfied := vm.pop().Stringify()
				vm.push(value.ValObjString(strfied + str))
			} else {
				vm.binaryOp('+')
			}

		case opcode.OP_SUBTRACT:
			vm.binaryOp('-')
		case opcode.OP_MULTIPLY:
			vm.binaryOp('*')
		case opcode.OP_DIVIDE:
			vm.binaryOp('/')
		case opcode.OP_GREATER:
			vm.binaryOp('>')
		case opcode.OP_LESS:
			vm.binaryOp('<')
		case opcode.OP_NIL:
			vm.push(value.ValNil())
		case opcode.OP_TRUE:
			vm.push(value.ValBool(true))
		case opcode.OP_FALSE:
			vm.push(value.ValBool(false))
		case opcode.OP_EQUAL:
			vm.push(value.ValBool(value.AreEqual(vm.pop(), vm.pop())))
		case opcode.OP_NEGATE:
			vm.stack[vm.stackTop-1] = value.ValNumber(-vm.stack[vm.stackTop-1].AsNumber())
		case opcode.OP_NOT:
			vm.push(value.ValBool(!vm.pop().IsTruey()))
		case opcode.OP_POP:
			vm.pop()
		case opcode.OP_PRINT:
			vm.pop().Print()
			fmt.Println()
		case opcode.OP_DEFINE_GLOBAL:
			name := vm.readConstant().AsGoString()
			_, ok := vm.globals[name]
			if ok {
				vm.runtimeError(fmt.Sprintf("Variable %s is already defined.", name))
			} else {
				vm.globals[name] = vm.pop()
			}
		case opcode.OP_GET_GLOBAL:
			name := vm.readConstant().AsGoString()
			val, ok := vm.globals[name]
			if !ok {
				vm.runtimeError(fmt.Sprintf("Undefined variable '%s'.", name))
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}
			vm.push(val)
		case opcode.OP_SET_GLOBAL:
			name := vm.readConstant().AsGoString()
			_, ok := vm.globals[name]
			if !ok {
				vm.runtimeError(fmt.Sprintf("Undefined variable '%s'.", name))
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}
			vm.globals[name] = vm.peek(0)
		case opcode.OP_GET_LOCAL:
			slot := vm.readByte()
			vm.push(vm.stack[frame.slots+int(slot)])
		case opcode.OP_SET_LOCAL:
			slot := vm.readByte()
			vm.stack[frame.slots+int(slot)] = vm.peek(0)
		case opcode.OP_LIST:
			count := int(vm.readByte())
			list := make([]value.Value, count)
			for i := count; i > 0; i-- {
				list[count-i] = vm.peek(i - 1)
			}
			for count > 0 {
				count--
				vm.pop()
			}
			vm.push(value.ValObjList(list))
		case opcode.OP_INDEX:
			valueIndex := vm.pop()
			valueList := vm.pop()

			if valueList.AsObj().Type != objtype.OBJ_LIST {
				vm.runtimeError("Invalid type to index into.")
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}

			objList := valueList.AsObjList()

			if valueIndex.Type != valuetype.VAL_NUMBER {
				vm.runtimeError("List index is not a number.")
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}

			index := valueIndex.AsNumber()

			if int(index) >= len(objList.List) {
				vm.runtimeError("List index out of range.")
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}

			vm.push(objList.List[int(index)])

		case opcode.OP_STORE:

			newValue := vm.pop()
			valueIndex := vm.pop()
			valueList := vm.pop()

			if valueList.AsObj().Type != objtype.OBJ_LIST {
				vm.runtimeError("Invalid type to index into.")
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}

			objList := valueList.AsObjList()

			if valueIndex.Type != valuetype.VAL_NUMBER {
				vm.runtimeError("List index is not a number.")
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}

			index := valueIndex.AsNumber()

			if int(index) >= len(objList.List) {
				vm.runtimeError("List index out of range.")
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}

			objList.List[int(index)] = newValue

			vm.push(newValue)

		case opcode.OP_JUMP_IF_FALSE:

			offset := vm.readTwoBytes()
			if !vm.peek(0).IsTruey() {
				frame.ip = incr(frame.ip, int(offset))
			}

		case opcode.OP_JUMP:

			offset := vm.readTwoBytes()
			frame.ip = incr(frame.ip, int(offset))

		case opcode.OP_LOOP:

			offset := vm.readTwoBytes()
			frame.ip = decr(frame.ip, int(offset))

		case opcode.OP_CALL:

			argCount := vm.readByte()
			if !vm.callValue(vm.peek(int(argCount)), int(argCount)) {
				return interpretresult.INTERPRET_RUNTIME_ERROR
			}
			frame = &vm.frames[len(vm.frames)-1]

		case opcode.OP_CLOSURE:

			function := vm.readConstant().AsObjFunction()
			closure := value.NewObjClosure(function)

			for i := 0; i < len(closure.Upvalues); i++ {
				isLocal := vm.readByte()
				index := vm.readByte()
				if isLocal == 1 {
					closure.Upvalues[i] = vm.captureUpvalue(&vm.stack[frame.slots+int(index)])
				} else {
					closure.Upvalues[i] = frame.closure.Upvalues[index]
				}
			}

			vm.push(value.ValObjClosure(closure))

		case opcode.OP_GET_UPVALUE:

			slot := vm.readByte()
			vm.push(*frame.closure.Upvalues[slot].Location)

		case opcode.OP_SET_UPVALUE:

			slot := vm.readByte()
			*(frame.closure.Upvalues[slot].Location) = vm.peek(0)

		case opcode.OP_RETURN:

			result := vm.pop()
			vm.frames = vm.frames[:len(vm.frames)-1]
			if len(vm.frames) == 0 {
				return interpretresult.INTERPRET_OK
			}

			for vm.stackTop != frame.slots {
				vm.pop()
			}

			vm.push(result)
			frame = &vm.frames[len(vm.frames)-1]
		}
	}
}

func (vm *VM) Interpret(source string) interpretresult.InterpretResult {

	function := compiler.Compile(&source)

	if function == nil {
		return interpretresult.INTERPRET_COMPILE_ERROR
	}

	closure := value.NewObjClosure(function)
	valClosure := value.ValObjClosure(closure)
	// vm.push(valClosure) // useless?
	vm.callValue(valClosure, 0)

	return vm.run()
}
