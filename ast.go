package yunsql

import (
	"fmt"
)

// 抽象语法树
type Ast struct {
	Statements []*Statement
}

type AstKind uint

const (
	SelectKind AstKind = iota
	CreateKind
	CreateIndexKind
	DropTableKind
	InsertKind
)

type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	CreateIndexStatement *CreateIndexStatement
	InsertStatement      *InsertStatement
	Kind                 AstKind
}

// Insert语句目前只有一个表名和一列值来插入
type InsertStatement struct {
	Table  Token
	Values *[]*Expression
}

type ExpressionKind uint

const (
	LiteralKind ExpressionKind = iota // 只有IdentifierKind, NumericKind, StringKind, BoolKind, NullKind才算
	BinaryKind
)

// 新增二进制表达式
type BinaryExpression struct {
	A  Expression
	B  Expression
	Op Token
}

// 一个表达式就是一系列的字面token或者未来可能加入的函数调用或者内联操作
type Expression struct {
	Literal *Token
	Binary  *BinaryExpression
	Kind    ExpressionKind
}

// Create语句有一个表名和一列列名和类型
type CreateTableStatement struct {
	Name Token                // 表名
	Cols *[]*ColumnDefinition // 列的信息
}

// 用于描述列的信息
type ColumnDefinition struct {
	Name       Token // 列名
	Datatype   Token // 每列的类型
	PrimaryKey bool  // 是否是主键
}

// 创建带有索引的表
// NOTE 具体见memory.go#CreateTableStatement
type CreateIndexStatement struct {
	Name       Token  // 这个不是表名,而是Token{Value: t.Name + "_pkey"},
	Unique     bool
	PrimaryKey bool
	Table      Token  // 这个才是表名代表的Token
	Exp        Expression
}

// Select语句有一个表名和一列列的名字
// TODO 弄清楚SelectStatement每个字段的具体含义
type SelectStatement struct {
	// table Token // 表的名字
	// colnames *[]*Token // 列的名字集合
	Item   *[]*SelectItem //列的名字
	From   *Token         // 表名
	Where  *Expression
	Limit  *Expression
	Offset *Expression
}

// TODO 弄清楚SelectItem每个字段的具体含义
type SelectItem struct {
	Exp      *Expression  // 一个SelectItem对应的表达式的值就是一个id或者一个age或者一个name 
	Asterisk bool // 是否是select * from 语句
	As       *Token
}

func (e Expression) generateCode() string {
	switch e.Kind {
	case LiteralKind:
		switch e.Literal.Kind {
		case IdentifierKind:
			return fmt.Sprintf("\"%s\"", e.Literal.Value)
		case StringKind:
			return fmt.Sprintf("'%s'", e.Literal.Value)
		default:
			return fmt.Sprintf(e.Literal.Value)
		}
	case BinaryKind:
		return e.Binary.generateCode()
	}

	return ""
}

func (be BinaryExpression) generateCode() string {
	return fmt.Sprintf("(%s %s %s)", be.A.generateCode(), be.Op.Value, be.B.generateCode())
}
