package yunsql

// 抽象语法树
type Ast struct {
	Statements []*Statement
}

type AstKind uint

const (
	SelectKind AstKind = iota
	CreateKind
	InsertKind
)

type Statement struct {
	SelectStatement *SelectStatement
	CreateStatement *CreateStatement
	InsertStatement *InsertStatement
	Kind            AstKind
}

// Insert语句目前只有一个表名和一列值来插入
type InsertStatement struct {
	Table  Token
	Values *[]*Expression
}

type ExpressionKind uint

const (
	LiteralKind ExpressionKind = iota
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
type CreateStatement struct {
	Table Token                // 表名
	Cols  *[]*ColumnDefinition // 列的信息
}

type ColumnDefinition struct {
	Name     Token // 列名
	Datatype Token // 每列的类型
}

type CreateIndexStatement struct {
	Name       Token
	Unique     bool
	PrimaryKey bool
	Table      Token
	Exp        Expression
}

// Select语句有一个表名和一列列的名字
type SelectStatement struct {
	// table Token // 表的名字
	// colnames *[]*Token // 列的名字集合
	Item   *[]*SelectItem //列的名字
	From   *Token         // 表名
	Where  *Expression
	Limit  *Expression
	Offset *Expression
}

type SelectItem struct {
	Exp      *Expression
	Asterisk bool // *
	As       *Token
}
