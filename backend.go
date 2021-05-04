package yunsql

import (
	"errors"

	"github.com/petar/GoLLRB/llrb"
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
	CreateTable(*CreateTableStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
	CreateIndex(*CreateIndexStatement) error
}

// 索引的解构
type Index struct {
	Name       string
	Exp        Expression
	Type       string
	Unique     bool
	PrimaryKey bool
	Tree       *llrb.LLRB
}

type EmptyBackend struct{}

// EmptyBackend 实现了接口Backend
func (eb EmptyBackend) CreateTable(_ *CreateTableStatement) error {
	return errors.New("create not support")
}

func (eb EmptyBackend) Insert(_ *InsertStatement) error {
	return errors.New("insert not supported")
}

func (eb EmptyBackend) Select(_ *SelectStatement) (*Results, error) {
	return nil, errors.New("select not supported")
}

func (eb EmptyBackend) CreateIndex(_ *CreateIndexStatement) error {
	return errors.New("create index not supported")
}
