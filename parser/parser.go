package parser

import (
	"errors"
	"fmt"

	"github.com/database-from-zero-to-one/lexer"
)

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
	Table  lexer.Token
	Values *[]*Expression
}

type ExpressionKind uint

const (
	LiteralKind ExpressionKind = iota
)

// 一个表达式就是一系列的字面token或者未来可能加入的函数调用或者内联操作
type Expression struct {
	Literal *lexer.Token
	Kind    ExpressionKind
}

// Create语句有一个表名和一列列名和类型
type CreateStatement struct {
	Table lexer.Token          // 表名
	Cols  *[]*ColumnDefinition // 列的信息
}

type ColumnDefinition struct {
	Name     lexer.Token // 列名
	Datatype lexer.Token // 每列的类型
}

// Select语句有一个表名和一列列的名字
type SelectStatement struct {
	// table lexer.Token // 表的名字
	// colnames *[]*Token // 列的名字集合
	Item []*Expression //列的名字
	From lexer.Token   // 表名
}

// parseing
func TokenFromKeyword(k lexer.Keyword) lexer.Token {
	return lexer.Token{
		Kind:  lexer.KeywordKind,
		Value: string(k),
	}
}

func TokenFromSymbol(s lexer.Symbol) lexer.Token {
	return lexer.Token{
		Kind:  lexer.SymbolKind,
		Value: string(s),
	}
}

// 检查tokens里面是不是期望的token(t)
func expectToken(tokens []*lexer.Token, cursor uint, t lexer.Token) bool {
	// cursor 代表的是 tokens数组中的索引,不能大于数组长度,否则返回false
	if cursor >= uint(len(tokens)) {
		return false
	}
	return t.Equals(tokens[cursor])
}

// 消息打印辅助函数
func helpMessage(tokens []*lexer.Token, cursor uint, msg string) {
	var c *lexer.Token
	if cursor < uint(len(tokens)) {
		c = tokens[cursor]
	} else {
		c = tokens[cursor-1]
	}
	fmt.Printf("[%d, %d]: %s, got: %s\n", c.Loc.Line, c.Loc.Col, msg, c.Value)
}

func Parse(source string) (*Ast, error) {
	// 先利用词法解析
	// source 对应 select * from test;
	// tokens 对应 [select, *, from, test]
	tokens, err := lexer.Lex(source)
	// 打印tokens数组
	// fmt.Printf("\n--------------------------------\n")
	// for i, t := range tokens {
	// 	fmt.Printf("索引%d--%s\n",i, t.Value)
	// }
	if err != nil {
		return nil, err
	}
	a := Ast{}
	cursor := uint(0)
	// 开始遍历
	for cursor < uint(len(tokens)) {
		// 解析statement
		stmt, newCursor, ok := parseStatement(tokens, cursor, TokenFromSymbol(lexer.SemicolonSymbol))
		if !ok {
			helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("failed to parse, expect statement")
		}
		cursor = newCursor // 更新cursor

		a.Statements = append(a.Statements, stmt)

		// 找找有没有分号
		atLeastOneSemicolon := false
		for expectToken(tokens, cursor, TokenFromSymbol(lexer.SemicolonSymbol)) {
			cursor++
			atLeastOneSemicolon = true
		}

		if !atLeastOneSemicolon {
			// 没有分号,也就是没有结束标志
			helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("missing semi-colon between statements")
		}
	}
	return &a, nil
}

// 解析语句辅助函数,每个statement将会是SELECT, INSERT, CREATE
func parseStatement(tokens []*lexer.Token, initialCursor uint, delimiter lexer.Token) (*Statement, uint, bool) {
	// 分别调动每个statement类型的解析函数
	cursor := initialCursor

	// 寻找SELECT
	semicolonToken := TokenFromSymbol(lexer.SemicolonSymbol)
	slct, newCursor, ok := parseSelectStatement(tokens, cursor, semicolonToken)
	if ok {
		// 证明找到了Selec语句
		return &Statement{
			Kind:            SelectKind,
			SelectStatement: slct,
		}, newCursor, true
	}

	// 不是SELECT语句，寻找INSERT
	insert, newCursor, ok := parseInsertStatement(tokens, cursor, semicolonToken)
	if ok {
		// 证明找到了Selec语句
		return &Statement{
			Kind:            InsertKind,
			InsertStatement: insert,
		}, newCursor, true
	}
	// 不是SELECT 和 INSERT, 寻找Create

	create, newCursor, ok := parseCreateStatement(tokens, cursor, semicolonToken)
	if ok {
		// 证明找到了Selec语句
		return &Statement{
			Kind:            CreateKind,
			CreateStatement: create,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

////////////////////////////////
// 解析select 语句
// Parsing SELECT statements is easy. We'll look for the following token pattern:

// SELECT
// $expression [, ...]
// FROM
// $table-name
func parseSelectStatement(tokens []*lexer.Token, initialCursor uint, delimiter lexer.Token) (*SelectStatement, uint, bool) {
	cursor := initialCursor
	// 如果token数组中当前索引对应的这个token不是Select的话,就返回错误
	if !expectToken(tokens, cursor, TokenFromKeyword(lexer.SelectKeyword)) {
		// return nil, cursor, false //错误 2021年4月25日21:16:31
		return nil, initialCursor, false
	}
	// 如果当前的这个token真的是select对应的token
	// 也就是当前tokens[cursor]就是我们要的Select
	// 那么就要看下一个token
	cursor++
	// 这就是我们要返回的结果
	slct := SelectStatement{}

	exps, newCursor, ok := parseExpressions(tokens, cursor, []lexer.Token{TokenFromKeyword(lexer.FromKeyword), delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.Item = *exps // 列名
	cursor = newCursor

	// 检查是不是from关键字
	if expectToken(tokens, cursor, TokenFromKeyword(lexer.FromKeyword)) {
		cursor++

		from, newCursor, ok := parseToken(tokens, cursor, lexer.IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected FROM token")
			return nil, initialCursor, false
		}
		slct.From = *from
		cursor = newCursor
	}

	return &slct, cursor, true
}

////////////////////////////////
// 解析Insert 语句
// We'll look for the following token pattern:
// INSERT
// INTO
// $table-name
// VALUES
// (
// $expression [, ...]
// )
func parseInsertStatement(tokens []*lexer.Token, initialCursor uint, delimiter lexer.Token) (*InsertStatement, uint, bool) {
	cursor := initialCursor
	// 找打INSERT
	if !expectToken(tokens, cursor, TokenFromKeyword(lexer.InsertKeyword)) {
		return nil, initialCursor, false
	}
	cursor++
	// 找到INTO
	if !expectToken(tokens, cursor, TokenFromKeyword(lexer.IntoKeyword)) {
		return nil, initialCursor, false
	}
	cursor++
	// 找到tablename
	table, newCursor, ok := parseToken(tokens, cursor, lexer.IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	// 找到VALUES
	if !expectToken(tokens, cursor, TokenFromKeyword(lexer.ValuesKeyword)) {
		helpMessage(tokens, cursor, "Expected VALUES")
		return nil, initialCursor, false
	}
	cursor++
	// 找到"("
	if !expectToken(tokens, cursor, TokenFromSymbol(lexer.LeftBracketSymbol)) {
		helpMessage(tokens, cursor, "Expected '(' ")
		return nil, initialCursor, false
	}
	cursor++
	// 找到表达式list
	values, newCursor, ok := parseExpressions(tokens, cursor, []lexer.Token{TokenFromSymbol(lexer.RightBracketSymbol)})
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor
	// 找到 ")"
	if !expectToken(tokens, cursor, TokenFromSymbol(lexer.RightBracketSymbol)) {
		helpMessage(tokens, cursor, "Expected ')'")
		return nil, initialCursor, false
	}

	cursor++ // 别忘了最后找到)的时候cursor要往后加一个
	return &InsertStatement{
		Table:  *table,
		Values: values,
	}, cursor, true
}

////////////////////////////////
// 解析Create语句
// Finally, for create statements we'll look for the following token pattern:

// CREATE
// $table-name
// (
// [$column-name $column-type [, ...]]
// )
func parseCreateStatement(tokens []*lexer.Token, initialCursor uint, delimiter lexer.Token) (*CreateStatement, uint, bool) {
	cursor := initialCursor
	// 找到CREATE
	if !expectToken(tokens, cursor, TokenFromKeyword(lexer.CreateKeyword)) {
		return nil, initialCursor, false
	}
	cursor++

	// 找到TABLE关键字
	if !expectToken(tokens, cursor, TokenFromKeyword(lexer.TableKeyword)) {
		return nil, initialCursor, false
	}
	cursor++
	// 找打tablename
	// table, newCursor, ok := parseExpression(tokens, cursor, delimiter)
	name, newCursor, ok := parseToken(tokens, cursor, lexer.IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor
	// 找到"("
	if !expectToken(tokens, cursor, TokenFromSymbol(lexer.LeftBracketSymbol)) {
		helpMessage(tokens, cursor, "Expected '('")
		return nil, initialCursor, false
	}
	cursor++
	// 找到column list
	cloums, newCursor, ok := parseColumnDefinitions(tokens, cursor, TokenFromSymbol(lexer.RightBracketSymbol))
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	// 找打")"
	if !expectToken(tokens, cursor, TokenFromSymbol(lexer.RightBracketSymbol)) {
		helpMessage(tokens, cursor, "Expected ')'")
		return nil, initialCursor, false
	}
	cursor++

	return &CreateStatement{
		Table: *name,
		Cols:  cloums,
	}, cursor, true
}

// 辅助函数,用于找到列名和跟在后面的列类型
func parseColumnDefinitions(tokens []*lexer.Token, initialCursor uint, delimiter lexer.Token) (*[]*ColumnDefinition, uint, bool) {
	cursor := initialCursor
	cds := []*ColumnDefinition{}
	// 循环找, 直到遇到分隔符(delimiter)位置
	for {
		// 每次循环都要检查cursor是否越界
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}
		// 查找delimiter
		current := tokens[cursor]
		// 找到delimiter，跳出循环
		if delimiter.Equals(current) {
			break
		}

		// 看看有么有逗号，有逗号的情景下证明cds数组里面已经有了一个元素
		if len(cds) > 0 {
			// check if there is a comma
			if !expectToken(tokens, cursor, TokenFromSymbol(lexer.CommaSymbol)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
			cursor++
		}

		// 找列名
		id, newCursor, ok := parseToken(tokens, cursor, lexer.IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}
		cursor = newCursor
		// 找列的类型
		ty, newCursor, ok := parseToken(tokens, cursor, lexer.KeywordKind)
		if !ok {
			return nil, initialCursor, false
		}
		cursor = newCursor

		cds = append(cds, &ColumnDefinition{
			Name:     *id,
			Datatype: *ty,
		})
	}
	return &cds, cursor, true
}

// The parseExpressions helper will look for tokens separated by a comma until a delimiter is found.
// It will use existing helpers plus parseExpression.
func parseExpressions(tokens []*lexer.Token, initialCursor uint, delimiters []lexer.Token) (*[]*Expression, uint, bool) {
	cursor := initialCursor
	exps := []*Expression{}
outer:
	for {
		// 先判断索引cursor有么有越界
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}
		// 寻找分隔符
		current := tokens[cursor]
		for _, d := range delimiters {
			// 如果发现这个token就是分隔符,直接退出循环
			if d.Equals(current) {
				break outer
			}
		}

		// 找逗号(前提是有逗号)
		if len(exps) > 0 {
			// 不是逗号
			if !expectToken(tokens, cursor, TokenFromSymbol(lexer.CommaSymbol)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}

			cursor++
		}
		// 找expression
		exp, newCursor, ok := parseExpression(tokens, cursor, TokenFromSymbol(lexer.CommaSymbol))
		// 说明氮
		if !ok {
			helpMessage(tokens, cursor, "Expected expression")
			return nil, initialCursor, false
		}
		// 说明遇到了逗号
		cursor = newCursor

		exps = append(exps, exp)
	}

	return &exps, cursor, true
}

// parseExpression 会找到数字, 字符串, 标识符等等
func parseExpression(tokens []*lexer.Token, initialCursor uint, _ lexer.Token) (*Expression, uint, bool) {
	cursor := initialCursor

	// 下面就是要找的种类
	kinds := []lexer.TokenKind{lexer.IdentifierKind, lexer.NumericKind, lexer.StringKind}
	for _, kind := range kinds {
		t, newCursor, ok := parseToken(tokens, cursor, kind)
		// 如果找到了特定kind的token
		if ok {
			return &Expression{
				Literal: t,
				Kind:    LiteralKind, // 字面量,目前只有一种ExpressionKind
			}, newCursor, true
		}
	}
	return nil, initialCursor, false
}

// parseToken辅助函数会找到特定kind的token
func parseToken(tokens []*lexer.Token, initialCursor uint, kind lexer.TokenKind) (*lexer.Token, uint, bool) {
	cursor := initialCursor
	// 检查有么有越界
	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}
	current := tokens[cursor]
	if current.Kind == kind {
		return current, cursor + 1, true
	}
	return nil, initialCursor, false
}
