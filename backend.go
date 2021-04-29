package yunsql

import (
	"errors"
)

type ColumnType uint

const (
	TextType ColumnType = iota
	IntType
	BoolType
)

type Cell interface {
	AsInt() int32
	AsText() string
	AsBool() bool
}

// 返回的结果
type Results struct {
	Columns []struct {
		Type ColumnType // 一个列既要有类型,也要有名字
		Name string
	}
	Rows [][]Cell // 一个行/记录
}

type Backend interface {
	CreateTable(*CreateStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
}

type EmptyBackend struct{}

func (eb EmptyBackend) CreateTable(_ *CreateStatement) error {
	return errors.New("create not support")
}

func (eb EmptyBackend) Insert(_ *InsertStatement) error {
	return errors.New("insert not supported")
}

func (eb EmptyBackend) Select(_ *SelectStatement) (*Results, error) {
	return nil, errors.New("select not supported")
}
