package builtins

import (
	"fmt"
	"golox/value"
	"golox/value/objtype"
	"golox/value/valuetype"
	"time"
)

func Clock(argCount int, args []value.Value) (value.Value, string) {
	return value.ValNumber(float64(time.Now().UnixMicro()) / 1000), ""
}

func Mod(argCount int, args []value.Value) (value.Value, string) {
	if argCount != 2 {
		return value.ValNil(), fmt.Sprintf("Required 2 arguments but got %d", argCount)
	}
	a := args[0]
	b := args[1]

	if a.Type != valuetype.VAL_NUMBER {
		return value.ValNil(), "Required 1st argument to be of type number."
	}
	if b.Type != valuetype.VAL_NUMBER {
		return value.ValNil(), "Required 2nd argument to be of type number."
	}

	result := value.ValNumber(float64(int(a.AsNumber()) % int(b.AsNumber())))

	return result, ""
}

func List(argCount int, args []value.Value) (value.Value, string) {
	if argCount != 1 {
		return value.ValNil(), fmt.Sprintf("Required 2 arguments but got %d", argCount)
	}

	a := args[0]

	if a.Type != valuetype.VAL_NUMBER {
		return value.ValNil(), "Required 1st argument to be of type number."
	}

	list := make([]value.Value, int(a.AsNumber()))

	return value.ValObjList(list), ""
}

func Append(argCount int, args []value.Value) (value.Value, string) {
	if argCount != 2 {
		return value.ValNil(), fmt.Sprintf("Required 2 arguments but got %d", argCount)
	}

	a := args[0]
	b := args[1]

	if a.Type != valuetype.VAL_OBJ && a.AsObj().Type != objtype.OBJ_LIST {
		return value.ValNil(), "Required 1st argument to be of type list."
	}

	objList := a.AsObjList()
	objList.List = append(objList.List, b)

	return a, ""
}

func Pop(argCount int, args []value.Value) (value.Value, string) {
	if argCount != 1 {
		return value.ValNil(), fmt.Sprintf("Required 2 arguments but got %d", argCount)
	}

	a := args[0]

	if a.Type != valuetype.VAL_OBJ && a.AsObj().Type != objtype.OBJ_LIST {
		return value.ValNil(), "Required 1st argument to be of type list."
	}

	objList := a.AsObjList()
	result := objList.List[len(objList.List)-1]
	objList.List = objList.List[:len(objList.List)-1]

	return result, ""
}

func Len(argCount int, args []value.Value) (value.Value, string) {
	if argCount != 1 {
		return value.ValNil(), fmt.Sprintf("Required 2 arguments but got %d", argCount)
	}

	a := args[0]

	if a.Type != valuetype.VAL_OBJ && a.AsObj().Type != objtype.OBJ_LIST {
		return value.ValNil(), "Required 1st argument to be of type list."
	}

	objList := a.AsObjList()
	result := value.ValNumber(float64(len(objList.List)))

	return result, ""
}
