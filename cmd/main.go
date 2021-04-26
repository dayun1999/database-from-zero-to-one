package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/database-from-zero-to-one/lexer"
	"github.com/database-from-zero-to-one/parser"
)

// An in-memory backend
// 列的类型
type ColumnType uint

const (
	TextType ColumnType = iota
	IntType
)

type Cell interface {
	AsText() string
	AsInt() int32
}

// 返回的结果
type Results struct {
	Columns []struct {
		Type ColumnType // 一个列既要有类型,也要有名字
		Name string
	}
	Rows [][]Cell // 一个行/记录
}

// 定义一些错误
var (
	ErrTableDoesNotExist  = errors.New("table does not exist")
	ErrColumnDoesNotExist = errors.New("column does not exist")
	ErrInvalidSelectItem  = errors.New("select Item is invalid")
	ErrInvalidDataType    = errors.New("invalid datatype")
	ErrMissingValue       = errors.New("missing values")
)

type Backend interface {
	CreateTable(*parser.CreateStatement) error
	Insert(*parser.InsertStatement) error
	Select(*parser.SelectStatement) (*Results, error)
}

////////////////////////////////
// memory layout
type MemoryCell []byte

// 实现Cell
func (mc MemoryCell) AsInt() int32 {
	var i int32
	err := binary.Read(bytes.NewBuffer(mc), binary.BigEndian, &i)
	if err != nil {
		panic(err)
	}
	return i
}

func (mc MemoryCell) AsText() string {
	return string(mc)
}

type table struct {
	Columns     []string
	ColumnTypes []ColumnType
	rows        [][]MemoryCell
}

type MemoryBackend struct {
	tables map[string]*table // 多张table
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}

// Implementing create table support
func (mb *MemoryBackend) CreateTable(crt *parser.CreateStatement) error {
	t := table{}
	mb.tables[crt.Table.Value] = &t
	if crt.Cols == nil {
		return nil
	}

	for _, col := range *crt.Cols {
		t.Columns = append(t.Columns, col.Name.Value)

		var datatype ColumnType
		switch col.Datatype.Value {
		case "int":
			datatype = IntType
		case "text":
			datatype = TextType
		default:
			return ErrInvalidDataType
		}
		t.ColumnTypes = append(t.ColumnTypes, datatype)
	}
	return nil
}

// Implementing insert support
func (mb *MemoryBackend) Insert(inst *parser.InsertStatement) error {
	// 查看表名是否存在
	table, ok := mb.tables[inst.Table.Value]
	// 没有这个表,就返回一个错误
	if !ok {
		return ErrTableDoesNotExist
	}

	if inst.Values == nil {
		return nil
	}

	row := []MemoryCell{}

	// 插入的值与列的数量对应不上
	if len(*inst.Values) != len(table.Columns) {
		return ErrMissingValue
	}

	for _, value := range *inst.Values {
		// if value.Kind == parser.LiteralKind {
		// 	fmt.Println("Skipp non-literal")
		// 	continue
		// }
		if value.Kind != parser.LiteralKind {
			fmt.Println("Skipp non-literal")
			continue
		}
		row = append(row, mb.tokenToCell(value.Literal))
	}

	table.rows = append(table.rows, row)
	return nil
}

// Implementing select support
func (mb *MemoryBackend) Select(slct *parser.SelectStatement) (*Results, error) {
	// 查看表名是否存在
	// table, ok := mb.tables[slct.From.Value]
	table, ok := mb.tables[slct.From.Value]
	if !ok {
		return nil, ErrTableDoesNotExist
	}

	results := [][]Cell{}
	columns := []struct {
		Type ColumnType
		Name string
	}{}

	// 遍历所有的行
	for i, row := range table.rows {
		result := []Cell{}
		// 是否是第一行
		isFirstRow := i == 0

		for _, exp := range slct.Item {
			if exp.Kind != parser.LiteralKind {
				fmt.Println("Skipping non-literal expression")
				continue
			}
			// 字面量
			lit := exp.Literal
			// 看看是不是标识符类型
			if lit.Kind == lexer.IdentifierKind {
				found := false
				// 遍历所有的列
				for i, tableCol := range table.Columns {
					if tableCol == lit.Value {
						// 第一行就是列名什么的
						if isFirstRow {
							columns = append(columns, struct {
								Type ColumnType
								Name string
							}{
								Type: table.ColumnTypes[i],
								Name: lit.Value,
							})
						}

						result = append(result, row[i])
						found = true
						// 找到一个，跳出最近的这个for循环
						break
					}
				}
				// 如果一个item都没有找到
				if !found {
					return nil, ErrColumnDoesNotExist
				}

				continue
			}
			return nil, ErrColumnDoesNotExist
			// 错误:
			// results = append(results, result)
		}
		results = append(results, result)
	}

	return &Results{
		Columns: columns,
		Rows:    results,
	}, nil
}

func (mb *MemoryBackend) tokenToCell(t *lexer.Token) MemoryCell {
	if t.Kind == lexer.NumericKind {
		buf := new(bytes.Buffer)
		// string converted into int
		i, err := strconv.Atoi(t.Value)
		if err != nil {
			panic(err)
		}

		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			panic(err)
		}
		return MemoryCell(buf.Bytes())
	}

	if t.Kind == lexer.StringKind {
		return MemoryCell(t.Value)
	}

	return nil
}

func main() {
	mb := NewMemoryBackend()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("欢迎来到云云数据库")
	for {
		fmt.Print("# ")
		text, err := reader.ReadString('\n')
		// fmt.Printf("你输入的数据是: %s", text)
		if err != nil {
			panic(err)
		}
		text = strings.Replace(text, "\n", "", -1)
		// fmt.Printf("你输入的数据是: %s", text)
		if text == "exit" {
			fmt.Printf("Bye Bye!")
			os.Exit(0)
		}

		// 尝试，trim space 失败
		// text = strings.Replace(text, " ", "", -1)

		ast, err := parser.Parse(text)
		if err != nil {
			panic(err)
		}

		for _, stmt := range ast.Statements {
			// 判断statement类型
			switch stmt.Kind {
			case parser.CreateKind:
				err = mb.CreateTable(ast.Statements[0].CreateStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("ok")
			case parser.InsertKind:
				err = mb.Insert(stmt.InsertStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("ok")
			case parser.SelectKind:
				results, err := mb.Select(stmt.SelectStatement)
				if err != nil {
					panic(err)
				}
				// 打印每一列
				for _, col := range results.Columns {
					fmt.Printf("| %s ", col.Name)
				}
				fmt.Println("|")

				// 打印分割线
				for j := 0; j < 20; j++ {
					fmt.Printf("=")
				}
				fmt.Println()

				// 然后打印行
				for _, result := range results.Rows {
					fmt.Printf("|")

					for i, cell := range result {
						typ := results.Columns[i].Type
						s := ""
						switch typ {
						case IntType:
							// s = strconv.Itoa(int(cell.AsInt()))
							s = fmt.Sprintf("%d", cell.AsInt())
						case TextType:
							s = cell.AsText()
						}

						fmt.Printf(" %s | ", s)
					}
					fmt.Println()
				}
				fmt.Println("ok")
			}
		}
	}

}
