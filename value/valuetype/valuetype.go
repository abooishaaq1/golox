package valuetype

type ValueType uint8

const (
	VAL_NIL    = iota
	VAL_BOOL   = iota
	VAL_NUMBER = iota
	VAL_OBJ    = iota
)
