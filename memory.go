package yunsql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
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

// function LiteralToMemoryCell maps a go value into a memeory cell
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

// tables
type table struct {
	Columns    []string
	ColumnType []ColumnType
	Rows       [][]MemoryCell
}

func (t *table) evaluateLiteralCell(rowIndex uint, exp Expression) (MemoryCell, string, ColumnType, error) {
	if exp.Kind != LiteralKind {
		return nil, "", 0, ErrInvalidCell
	}

	lit := exp.Literal
	if lit.Kind == IdentifierKind {
		for i, tableCol := range t.Columns {
			if tableCol == lit.Value {
				return t.Rows[rowIndex][i], tableCol, t.ColumnType[i], nil
			}
		}
		return nil, "", 0, ErrColumnDoesNotExist
	}

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
	l, _, lt, err := t.evaluateCell(rowIndex, bexp.A)
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
				return TrueMemoryCell, "?column?", BoolType, nil
			}
			if lt == IntType && rt == IntType && eq {
				return TrueMemoryCell, "?column?", BoolType, nil
			}
			if lt == BoolType && rt == BoolType && eq {
				return TrueMemoryCell, "?column?", BoolType, nil
			}

			return FalseMemoryCell, "?column?", BoolType, nil
		case NeqSymbol:
			if lt != rt || !l.Equals(r) {
				return TrueMemoryCell, "?column?", BoolType, nil
			}

			return FalseMemoryCell, "?column?", BoolType, nil
		case ConcatSymbol:
			if lt != TextType || rt != TextType {
				return nil, "", 0, ErrInvalidOperands
			}

			return literalToMemoryCell(&Token{Kind: StringKind, Value: l.AsText() + r.AsText()}), "?column?", TextType, nil
		case PlusSymbol:
			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			iValue := int(l.AsInt() + r.AsInt())
			return literalToMemoryCell(&Token{Kind: NumericKind, Value: strconv.Itoa(iValue)}), "?column?", IntType, nil
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

			return res, "?column?", BoolType, nil
		case OrKeyword:
			if lt != BoolType || rt != BoolType {
				return nil, "", 0, ErrInvalidOperands
			}
			res := FalseMemoryCell
			if l.AsBool() || r.AsBool() {
				res = TrueMemoryCell
			}

			return res, "?column?", BoolType, nil
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

// MemoryBackend
type MemoryBackend struct {
	tables map[string]*table
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}

// Implementing Select statement
func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	// 查看表名是否存在
	// table, ok := mb.tables[slct.From.Value]
	t := &table{}

	if slct.From != nil {
		var ok bool
		t, ok = mb.tables[slct.From.Value]
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

	if slct.From == nil {
		t = &table{}
		t.Rows = [][]MemoryCell{}
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
			if col.Asterisk {
				fmt.Println("Skipping *")
				continue
			}

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
func (mb *MemoryBackend) CreateTable(crt *CreateStatement) error {
	t := table{}
	mb.tables[crt.Table.Value] = &t
	if crt.Cols == nil {
		return nil
	}

	for _, col := range *crt.Cols {
		t.Columns = append(t.Columns, col.Name.Value)

		var dt ColumnType
		switch col.Datatype.Value {
		case "int":
			dt = IntType
		case "text":
			dt = TextType
		default:
			return ErrInvalidDataType
		}
		t.ColumnType = append(t.ColumnType, dt)
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

	// 插入的值与列的数量对应不上
	if len(*inst.Values) != len(t.Columns) {
		return ErrMissingValue
	}

	for _, value := range *inst.Values {
		if value.Kind != LiteralKind {
			fmt.Println("Skipp non-literal")
			continue
		}
		emptyTable := &table{}
		value, _, _, err := emptyTable.evaluateCell(0, *value)
		if err != nil {
			return err
		}
		row = append(row, value)
	}
	t.Rows = append(t.Rows, row)
	return nil
}
