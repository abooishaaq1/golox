package value

import (
	"fmt"
	"golox/value/objtype"
	"golox/value/valuetype"
	"strconv"
	"unsafe"
)

type Value struct {
	Type valuetype.ValueType
	Data interface{}
}

type Obj struct {
	Type objtype.ObjType
}

type ObjList struct {
	Obj
	List []Value
}

type ObjString struct {
	Obj
	String string
}

type FuncChunk interface {
	Write(bits uint8, lines int)
	AddConstant(constant Value) int
}

type ObjFunction struct {
	Obj
	Arity        int
	UpvalueCount int
	Chunk        FuncChunk
	Name         *ObjString // TODO: why not just `string`?
}

type ObjClosure struct {
	Obj
	Function *ObjFunction
	Upvalues []*ObjUpvalue
}

type ObjUpvalue struct {
	Obj
	Closed   Value
	Location *Value
}

type NativeFn func(argCount int, args []Value) (Value, string)

type ObjNative struct {
	Obj
	Function NativeFn
}

func ValBool(val bool) Value {
	return Value{valuetype.VAL_BOOL, val}
}

func ValNumber(val float64) Value {
	return Value{valuetype.VAL_NUMBER, val}
}

func ValNil() Value {
	return Value{valuetype.VAL_NIL, nil}
}

func ValObjFunction(function *ObjFunction) Value {
	return Value{Type: valuetype.VAL_OBJ, Data: (*Obj)(unsafe.Pointer(function))}
}

func ValObjClosure(closure *ObjClosure) Value {
	return Value{Type: valuetype.VAL_OBJ, Data: (*Obj)(unsafe.Pointer(closure))}
}

func ValObjList(list []Value) Value {
	objList := NewObjList(list)
	return Value{Type: valuetype.VAL_OBJ, Data: (*Obj)(unsafe.Pointer(objList))}
}

func ValObjString(val string) Value {
	objStr := NewObjString(val)
	return Value{Type: valuetype.VAL_OBJ, Data: (*Obj)(unsafe.Pointer(objStr))}
}

func ValNative(function NativeFn) Value {
	nativeFunc := new(ObjNative)
	nativeFunc.Function = function
	nativeFunc.Obj.Type = objtype.OBJ_NATIVE
	return Value{Type: valuetype.VAL_OBJ, Data: (*Obj)(unsafe.Pointer(nativeFunc))}
}

func NewObjList(list []Value) *ObjList {
	objList := new(ObjList)
	objList.List = list
	objList.Obj.Type = objtype.OBJ_LIST
	return objList
}

func NewObjString(val string) *ObjString {
	objStr := ObjString{Obj: Obj{Type: objtype.OBJ_STRING}, String: val}
	return &objStr
}

func NewObjFunction(chunk FuncChunk) *ObjFunction {
	objFunc := new(ObjFunction)
	objFunc.Chunk = chunk
	objFunc.Type = objtype.OBJ_FUNCTION
	return objFunc
}

func NewObjClosure(function *ObjFunction) *ObjClosure {
	objClosure := new(ObjClosure)
	objClosure.Function = function
	objClosure.Type = objtype.OBJ_CLOSURE
	objClosure.Upvalues = make([]*ObjUpvalue, function.UpvalueCount)
	return objClosure
}

func NewObjUpvalue(slot *Value) *ObjUpvalue {
	upvalue := new(ObjUpvalue)
	upvalue.Closed = *slot
	upvalue.Location = &upvalue.Closed
	return upvalue
}

func (value Value) AsBool() bool {
	return value.Data.(bool)
}

func (value Value) AsNumber() float64 {
	return value.Data.(float64)
}

func (value Value) AsObj() *Obj {
	return value.Data.(*Obj)
}

func (value Value) AsGoString() string {
	return value.AsString().String
}

func (value Value) AsString() *ObjString {
	return (*ObjString)(unsafe.Pointer(value.AsObj()))
}

func (value Value) AsObjList() *ObjList {
	return (*ObjList)(unsafe.Pointer(value.AsObj()))
}

func (value Value) AsObjFunction() *ObjFunction {
	return (*ObjFunction)(unsafe.Pointer(value.AsObj()))
}

func (value Value) AsObjClosure() *ObjClosure {
	return (*ObjClosure)(unsafe.Pointer(value.AsObj()))
}

func (value Value) AsNative() NativeFn {
	return (*ObjNative)(unsafe.Pointer(value.AsObj())).Function
}

func (value Value) IsBool() bool {
	return value.Type == valuetype.VAL_BOOL
}

func (value Value) IsNumber() bool {
	return value.Type == valuetype.VAL_NUMBER
}

func (value Value) IsObj() bool {
	return value.Type == valuetype.VAL_OBJ
}

func (value Value) IsOBjType(typee objtype.ObjType) bool {
	return value.IsObj() && value.AsObj().Type == typee
}

func (value Value) IsString() bool {
	return value.IsOBjType(objtype.OBJ_STRING)
}

func (value Value) IsFunction() bool {
	return value.IsOBjType(objtype.OBJ_FUNCTION)
}

func (value Value) IsTruey() bool {
	switch value.Type {
	case valuetype.VAL_NIL:
		return false
	case valuetype.VAL_BOOL:
		return value.AsBool()
	case valuetype.VAL_NUMBER:
		if value.AsNumber() == 0 {
			return false
		}
	}
	return true
}

func AreEqual(a Value, b Value) bool {
	if a.Type != b.Type {
		return false
	}

	switch a.Type {
	case valuetype.VAL_NIL:
		if b.Type == valuetype.VAL_NIL {
			return true
		}
		return false
	case valuetype.VAL_BOOL:
		return a.AsBool() == b.AsBool()
	case valuetype.VAL_NUMBER:
		return a.AsNumber() == b.AsNumber()
	case valuetype.VAL_OBJ:
		return a.AsGoString() == b.AsGoString()
	}
	return false
}

func (value Value) Stringify() string {
	switch value.Type {
	case valuetype.VAL_NIL:
		return "nil"
	case valuetype.VAL_BOOL:
		return fmt.Sprint(value.AsBool())
	case valuetype.VAL_NUMBER:
		return strconv.FormatFloat(value.AsNumber(), 'f', -1, 64)
	case valuetype.VAL_OBJ:
		switch value.AsObj().Type {
		case objtype.OBJ_STRING:
			return value.AsGoString()
		case objtype.OBJ_LIST:
			result := "[ "
			for _, v := range value.AsObjList().List {
				result = result + v.Stringify() + ", "
			}
			result += "]"
			return result
		case objtype.OBJ_FUNCTION:
			return fmt.Sprintf("<fn %s>", value.AsObjFunction().Name.String)
		case objtype.OBJ_CLOSURE:
			return fmt.Sprintf("<fn %s>", value.AsObjClosure().Function.Name.String)
		case objtype.OBJ_NATIVE:
			return "<native fn>"
		}
	}
	return "<undefined>"
}

func (value Value) Print() {
	fmt.Print(value.Stringify())
}
