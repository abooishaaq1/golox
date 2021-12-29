package objtype

type ObjType uint8

const (
	OBJ_STRING   ObjType = iota
	OBJ_LIST     ObjType = iota
	OBJ_NATIVE   ObjType = iota
	OBJ_FUNCTION ObjType = iota
	OBJ_CLOSURE  ObjType = iota
	OBJ_UPVALUE  ObjType = iota
	OBJ_CLASS    ObjType = iota
)
