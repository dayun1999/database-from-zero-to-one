package yunsql

import (
	"errors"
)

// 定义一些错误
var (
	ErrTableDoesNotExist  = errors.New("table does not exist")
	ErrColumnDoesNotExist = errors.New("column does not exist")
	ErrInvalidSelectItem  = errors.New("select Item is invalid")
	ErrInvalidDataType    = errors.New("invalid datatype")
	ErrMissingValue       = errors.New("missing values")
	ErrInvalidCell        = errors.New("invalid cell")
	ErrInvalidOperands    = errors.New("invalid operands")
)
