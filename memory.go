package yunsql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/petar/GoLLRB/llrb"
)

type MemoryCell []byte

func (mc MemoryCell) AsInt() int32 {
	var i int32
	err := binary.Read(bytes.NewBuffer(mc), binary.BigEndian, &i)
	if err != nil {
		fmt.Printf("Corrupted data [%s]: %s\n", mc, err)
		return 0
	}
	return i
}

func (mc MemoryCell) AsText() string {
	return string(mc)
}

func (mc MemoryCell) AsBool() bool {
	return len(mc) != 0
}

func (mc MemoryCell) Equals(b MemoryCell) bool {
	if mc == nil || b == nil {
		return mc == nil && b == nil
	}

	return bytes.Equal(mc, b)
}

var (
	TrueToken  = Token{Kind: BoolKind, Value: "true"}
	FalseToken = Token{Kind: BoolKind, Value: "false"}

	TrueMemoryCell  = literalToMemoryCell(&TrueToken)
	FalseMemoryCell = literalToMemoryCell(&FalseToken)
)

type TreeItem struct {
	Value MemoryCell
	Index uint
}

// 实现Item接口
func (te TreeItem) Less(than llrb.Item) bool {
	return bytes.Equal(te.Value, than.(TreeItem).Value)
}

// 将字面量(1, 'wdy' etc.)转换为MemoeyCell
func literalToMemoryCell(t *Token) MemoryCell {
	// 根据token的kind来解析
	switch t.Kind {
	case NumericKind:
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.Value)
		if err != nil {
			fmt.Printf("Currupted data [%s]: %s\n", t.Value, err)
			return MemoryCell(nil)
		}

		// handle big int
		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			fmt.Printf("Currupted data [%s]: %s\n", buf.String(), err)
			return MemoryCell(nil)
		}
		return MemoryCell(buf.Bytes())
	case StringKind:
		return MemoryCell(t.Value)
	case BoolKind:
		if t.Value == "true" {
			return MemoryCell([]byte{1})
		} else {
			return MemoryCell(nil)
		}
	}
	return nil
}

// 表的结构体
type table struct {
	Name        string  // 表名称
	Columns     []string //列名,比如[]string{id, name, age}之类的字符串切片
	ColumnTypes []ColumnType // 列的类型
	Rows        [][]MemoryCell // 行的集合,每一行又是一个MemoryCell切片,且每个MemoryCell对应转换过来的插入的值,比如(1, 20, 'wdy')中的'wdy'
	Indexes     []*Index
}

func createTable() *table {
	return &table{}
}

// NOTE function evaluateCell 的作用就是将传进来的表达式解构,提取出具体的字面量,然后再处理变为MemoryCell,返回的结果是{值,列名,列的类型,error}
func (t *table) evaluateLiteralCell(rowIndex uint, exp Expression) (MemoryCell, string, ColumnType, error) {
	// REVIEW 如果表达式的类型不是IdentifierKind, NumericKind, StringKind, BoolKind, NullKind其中的一个的话,直接返回
	if exp.Kind != LiteralKind {
		return nil, "", 0, ErrInvalidCell
	}

	// 取出表达式的字面量
	lit := exp.Literal
	// FIXME 打印lit的值来看看
	fmt.Printf("来自memory.go#evaluateLiteralCell,lit的值为%s\n", lit.Value)
	if lit.Kind == IdentifierKind {
		for i, tableCol := range t.Columns {
			if tableCol == lit.Value {
				return t.Rows[rowIndex][i], tableCol, t.ColumnTypes[i], nil
			}
		}
		return nil, "", 0, ErrColumnDoesNotExist
	}
	
	// 如果字面量是1,就是IntType
	columnType := IntType
	if lit.Kind == StringKind {
		columnType = TextType
	} else if lit.Kind == BoolKind {
		columnType = BoolType
	}
	return literalToMemoryCell(lit), "?column?", columnType, nil
}

func (t *table) evaluateBinaryCell(rowIndex uint, exp Expression) (MemoryCell, string, ColumnType, error) {
	if exp.Kind != BinaryKind {
		return nil, "", 0, ErrInvalidCell
	}

	bexp := exp.Binary

	// 左边的被操作数
	l, columnName, lt, err := t.evaluateCell(rowIndex, bexp.A)
	if err != nil {
		return nil, "", 0, err
	}

	// 右边的被操作数
	r, _, rt, err := t.evaluateCell(rowIndex, bexp.B)
	if err != nil {
		return nil, "", 0, err
	}

	// 操作符
	switch bexp.Op.Kind {
	case SymbolKind:
		switch Symbol(bexp.Op.Value) {
		case EqSymbol:
			eq := l.Equals(r)
			if lt == TextType && rt == TextType && eq {
				return TrueMemoryCell, columnName, BoolType, nil
			}
			if lt == IntType && rt == IntType && eq {
				return TrueMemoryCell, columnName, BoolType, nil
			}
			if lt == BoolType && rt == BoolType && eq {
				return TrueMemoryCell, columnName, BoolType, nil
			}

			return FalseMemoryCell, columnName, BoolType, nil
		case NeqSymbol:
			if lt != rt || !l.Equals(r) {
				return TrueMemoryCell, columnName, BoolType, nil
			}

			return FalseMemoryCell, columnName, BoolType, nil
		case ConcatSymbol:
			if lt != TextType || rt != TextType {
				return nil, "", 0, ErrInvalidOperands
			}

			return literalToMemoryCell(&Token{Kind: StringKind, Value: l.AsText() + r.AsText()}), columnName, TextType, nil
		case PlusSymbol:
			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			iValue := int(l.AsInt() + r.AsInt())
			return literalToMemoryCell(&Token{Kind: NumericKind, Value: strconv.Itoa(iValue)}), columnName, IntType, nil
		default:
			// TODO
			break
		}
	case KeywordKind:
		switch Keyword(bexp.Op.Value) {
		case AndKeyword:
			if lt != BoolType || rt != BoolType {
				return nil, "", 0, ErrInvalidOperands
			}
			res := FalseMemoryCell
			if l.AsBool() && r.AsBool() {
				res = TrueMemoryCell
			}

			return res, columnName, BoolType, nil
		case OrKeyword:
			if lt != BoolType || rt != BoolType {
				return nil, "", 0, ErrInvalidOperands
			}
			res := FalseMemoryCell
			if l.AsBool() || r.AsBool() {
				res = TrueMemoryCell
			}

			return res, columnName, BoolType, nil
		default:
			break
		}
	}

	return nil, "", 0, ErrInvalidCell
}

func (t *table) evaluateCell(rowIndex uint, exp Expression) (MemoryCell, string, ColumnType, error) {
	switch exp.Kind {
	case LiteralKind:
		return t.evaluateLiteralCell(rowIndex, exp)
	case BinaryKind:
		return t.evaluateBinaryCell(rowIndex, exp)
	default:
		return nil, "", 0, ErrInvalidCell
	}
}

type IndexAndExpression struct {
	i *Index
	e Expression
}

// 目前只支持where后面是boolean表达式,比如select * from users where age=23 or age=25;
// 目前只支持AND
func (t *table) getApplicableIndexes(where *Expression) []IndexAndExpression {

	var linearizeExpressions func(where *Expression, exps []Expression) []Expression

	linearizeExpressions = func(where *Expression, exps []Expression) []Expression {
		// 如果没有where语句
		if where == nil || where.Kind != BinaryKind {
			return exps
		}

		// 目前只支持and
		if where.Binary.Op.Value == string(OrKeyword) {
			return exps
		}

		if where.Binary.Op.Value == string(AndKeyword) {
			exps := linearizeExpressions(&where.Binary.A, exps)
			return linearizeExpressions(&where.Binary.B, exps)
		}
		return append(exps, *where)
	}

	exps := linearizeExpressions(where, []Expression{})

	iAndE := []IndexAndExpression{}
	for _, exp := range exps {
		for _, index := range t.Indexes {
			if index.applicableValue(exp) != nil {
				iAndE = append(iAndE, IndexAndExpression{
					i: index,
					e: exp,
				})
			}
		}
	}

	return iAndE
}

// function addRow为行添加索引
func (i *Index) addRow(t *table, rowIndex uint) error {
	// i.Exp的值 &Expression{&"id", LiteralKind}
	indexValue, _, _, err := t.evaluateCell(rowIndex, i.Exp)
	if err != nil {
		return err
	}

	// 如果value为空,就触犯了not null的限制
	if indexValue == nil {
		return ErrViolatesNotNullConstraint
	}

	if i.Unique && i.Tree.Has(TreeItem{Value: indexValue}) {
		return ErrViolatesUniqueConstraint
	}

	i.Tree.InsertNoReplace(TreeItem{
		Value: indexValue,
		Index: rowIndex,
	})
	return nil
}

func (i *Index) applicableValue(exp Expression) *Expression {
	// 如果不是二进制表达式,就返回,证明要找的是二进制表达式
	if exp.Kind != BinaryKind {
		return nil
	}

	be := exp.Binary

	// 目前只要找到boolean expression的column and value即可
	columnExp := be.A
	valueExp := be.B

	// 只有kind相同generateCode才会相等
	if columnExp.generateCode() != i.Exp.generateCode() {
		columnExp = be.B
		valueExp = be.A
	}

	// 交换后两边还是不等
	if columnExp.generateCode() != i.Exp.generateCode() {
		return nil
	}

	supportChecks := []Symbol{EqSymbol, NeqSymbol, NeqSymbol2, GtSymbol, LtSymbol, GteSymbol, LteSymbol}
	supported := false
	for _, sym := range supportChecks {
		// 找到支持的二进制符号了,就退出遍历
		if string(sym) == be.Op.Value {
			supported = true
			break
		}
	}

	// 如果不支持,就返回nil
	if !supported {
		return nil
	}

	if valueExp.Kind != LiteralKind {
		fmt.Println("Only index checks on literal supported")
		return nil
	}
	return &valueExp
}

// function newTableFromSubset根据where过滤返回原来table的一个子集
func (i *Index) newTableFromSubset(t *table, exp Expression) *table {
	valueExp := i.applicableValue(exp)
	if valueExp == nil {
		return t
	}

	value, _, _, err := createTable().evaluateCell(0, *valueExp)
	if err != nil {
		fmt.Println(err)
		return t
	}

	tiValue := TreeItem{
		Value: value,
	}

	indexes := []uint{}
	switch Symbol(exp.Binary.Op.Value) {
	case EqSymbol:
		i.Tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(TreeItem)

			if !bytes.Equal(ti.Value, value) {
				return false
			}
			indexes = append(indexes, ti.Index)
			return true
		})
	case NeqSymbol:
		fallthrough
	case NeqSymbol2:
		i.Tree.AscendGreaterOrEqual(llrb.Inf(-1), func(i llrb.Item) bool {
			ti := i.(TreeItem)
			if bytes.Equal(ti.Value, value) {
				indexes = append(indexes, ti.Index)
			}

			return true
		})
	case LtSymbol:
		i.Tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(TreeItem)
			if bytes.Compare(ti.Value, value) < 0 {
				indexes = append(indexes, ti.Index)
			}

			return true
		})
	case LteSymbol:
		i.Tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(TreeItem)
			if bytes.Compare(ti.Value, value) <= 0 {
				indexes = append(indexes, ti.Index)
			}

			return true
		})
	case GtSymbol:
		i.Tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(TreeItem)
			if bytes.Compare(ti.Value, value) > 0 {
				indexes = append(indexes, ti.Index)
			}

			return true
		})
	case GteSymbol:
		i.Tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(TreeItem)
			if bytes.Compare(ti.Value, value) >= 0 {
				indexes = append(indexes, ti.Index)
			}

			return true
		})
	}

	newT := createTable()
	newT.Columns = t.Columns
	newT.ColumnTypes = t.ColumnTypes
	newT.Indexes = t.Indexes
	newT.Rows = [][]MemoryCell{}

	for _, index := range indexes {
		newT.Rows = append(newT.Rows, t.Rows[index])
	}

	return newT
}

// MemoryBackend
type MemoryBackend struct {
	tables map[string]*table
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}

// MemoryBackend也实现了Backend接口
// Implementing Select statement
func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	// 查看表名是否存在
	// table, ok := mb.tables[slct.From.Value]
	t := createTable()

	// slct.From代表的是表名
	if slct.From != nil {
		var ok bool
		t, ok = mb.tables[slct.From.Value]
		// 如果表不存在,就返回错误
		if !ok {
			return nil, ErrTableDoesNotExist
		}
	}

	if slct.Item == nil || len(*slct.Item) == 0 {
		return &Results{}, nil
	}

	results := [][]Cell{}
	columns := []struct {
		Type ColumnType
		Name string
	}{}

	// TODO 不知道这行是干什么的
	// if slct.From == nil {
	// 	t = &table{}
	// 	t.Rows = [][]MemoryCell{}
	// }

	// 增加
	for _, iAndE := range t.getApplicableIndexes(slct.Where) {
		index := iAndE.i
		exp := iAndE.e
		t = index.newTableFromSubset(t, exp)
	}
	// 遍历所有的行
	for i := range t.Rows {
		result := []Cell{}
		// 是否是第一行
		isFirstRow := len(results) == 0

		if slct.Where != nil {
			val, _, _, err := t.evaluateCell(uint(i), *slct.Where)
			if err != nil {
				return nil, err
			}

			if !val.AsBool() {
				continue
			}
		}

		for _, col := range *slct.Item {
			value, columName, columnType, err := t.evaluateCell(uint(i), *col.Exp)
			if err != nil {
				return nil, err
			}
			// 第一行就是列名什么的
			if isFirstRow {
				columns = append(columns, struct {
					Type ColumnType
					Name string
				}{
					Type: columnType,
					Name: columName,
				})
			}

			result = append(result, value)
		}
		results = append(results, result)
	}

	return &Results{
		Columns: columns,
		Rows:    results,
	}, nil
}

// Implementing Create statement
func (mb *MemoryBackend) CreateTable(crt *CreateTableStatement) error {
	// 查看表名是否存在
	if _, ok := mb.tables[crt.Name.Value]; ok {
		return ErrTableAlreadyExists
	}

	t := createTable()
	t.Name = crt.Name.Value
	mb.tables[t.Name] = t
	// 如果只创建了表名但是没有列名,那么就直接返回
	if crt.Cols == nil {
		return nil
	}

	var primaryKey *Expression = nil
	// NOTE 注意遍历的目的是什么
	// 比如CREATE TABLE users(id INT PRIMARY KEY, name TEXT);
	// 这里col就代表了成员变量为(比如)&ColumDefinition{Name: id, Datatype: int, PrimaryKey: true}的结构体
	// 表的结构体
	// type table struct {
	// 	Name        string
	// 	Columns     []string  //列名,比如[]string{id, name, age}之类的字符串切片
	// 	ColumnTypes []ColumnType
	// 	Rows        [][]MemoryCell
	// 	Indexes     []*Index
	// }

	for _, col := range *crt.Cols {
		// 遍历的目的就是为了寻找列名、列的类型
		t.Columns = append(t.Columns, col.Name.Value)

		var dt ColumnType
		switch col.Datatype.Value {
		case "int":
			dt = IntType
		case "text":
			dt = TextType
		case "bool":
			dt = BoolType
		default:
			// 遇到不存在的类型,建立表失败
			delete(mb.tables, t.Name)
			return ErrInvalidDataType
		}
		// 创建表的时候带有主键,比如id int primary key,那么id就是主键
		if col.PrimaryKey {
			// 如果第一次primary的表达式就不为nil，删除表
			// NOTE 可理解为只有一个列能成为主键
			if primaryKey != nil {
				delete(mb.tables, t.Name)
				return ErrPrimaryKeyAlreadyExists
			}
			// REVIEW 这里主键表达式的内容就是{列名, 表达式类型}
			primaryKey = &Expression{
				Literal: &col.Name, // 一般为 "id"
				Kind:    LiteralKind,
			}
		}
		t.ColumnTypes = append(t.ColumnTypes, dt)
	}

	// 有主键了
	if primaryKey != nil {
		// REVIEW 创建索引从这里开始
		err := mb.CreateIndex(&CreateIndexStatement{
			Table:      crt.Name,
			Name:       Token{Value: t.Name + "_pkey"},
			Unique:     true,
			PrimaryKey: true,
			Exp:        *primaryKey,
		})

		if err != nil {
			delete(mb.tables, t.Name)
			return err
		}
	}
	return nil
}

// Implementing Insert statement
func (mb *MemoryBackend) Insert(inst *InsertStatement) error {
	// 查看表名是否存在
	t, ok := mb.tables[inst.Table.Value]
	// 没有这个表,就返回一个错误
	if !ok {
		return ErrTableDoesNotExist
	}

	if inst.Values == nil {
		return nil
	}

	row := []MemoryCell{}

	// 插入的时候没有给每一列赋值,就返回错误
	// fmt.Printf("#########len(*inst.Values)的值为%d\n", len(*inst.Values))

	if len(*inst.Values) != len(t.Columns) {
		return ErrMissingValue
	}

	for _, value := range *inst.Values {
		if value.Kind != LiteralKind {
			fmt.Println("Skipp non-literal")
			continue
		}
		emptyTable := createTable()
		// NOTE 注意下面exvaluateCell返回的第一个参数value不一定是可以打印出来的
		value, _, _, err := emptyTable.evaluateCell(0, *value)
		if err != nil {
			return err
		}
		row = append(row, value)
	}
	t.Rows = append(t.Rows, row)

	// FIXME 调试t.Rows一下
	debugRows(t.Rows)

	// 为当前行(新增加的行)添加索引
	for _, index := range t.Indexes {
		err := index.addRow(t, uint(len(t.Rows)-1))
		if err != nil {
			// drop the row on failure
			t.Rows = t.Rows[:len(t.Rows)-1]
			return err
		}
	}
	return nil
}

// Implementing CreateIndex statement
func (mb *MemoryBackend) CreateIndex(ci *CreateIndexStatement) error {
	// REVIEW 得到CreateIndexStatement的内容
	// CREATE TABLE users (id INT PRIMARY KEY, name TEXT);
	// &CreateIndexStatement{
	// 	Table: "users",
	// 	NAME: "users_pkey",
	// 	Unique: true,
	// 	PrimaryKey: true,
	// 	Exp: &Expression{&"id", LiteralKind},
	// }
	// 验证表是否存在
	table, ok := mb.tables[ci.Table.Value]
	if !ok {
		return ErrTableDoesNotExist
	}

	// 检查列对应的索引是否已经存在,如果已经存在就返回错误
	for _, index := range table.Indexes {
		// 索引已经存在
		if index.Name == ci.Name.Value {
			return ErrIndexAlreadyExists
		}
	}

	// NOTE 真正建立索引
	index := &Index{
		Name:       ci.Name.Value,
		Exp:        ci.Exp,
		PrimaryKey: ci.PrimaryKey,
		Tree:       llrb.New(), // 索引的存储数据结构,底层是一个左倾的红黑2-3树
		Type:       "red_black_tree",   // NOTE 索引的类型,这里索引的创建只用到了红黑树一种(postgres支持多种索引),所以暂时写死
	}
	table.Indexes = append(table.Indexes, index)

	// FIXME 这一段代码有必要吗？
	for i := range table.Rows {
		err := index.addRow(table, uint(i))
		if err != nil {
			return err
		}
	}
	return nil
}
