package yunsql

import (
	"errors"
	"fmt"
)

func TokenFromKeyword(k Keyword) Token {
	return Token{
		Kind:  KeywordKind,
		Value: string(k),
	}
}

func TokenFromSymbol(s Symbol) Token {
	return Token{
		Kind:  SymbolKind,
		Value: string(s),
	}
}

// 检查tokens里面是不是期望的token(t)
func expectToken(tokens []*Token, cursor uint, t Token) bool {
	// cursor 代表的是 tokens数组中的索引,不能大于数组长度,否则返回false
	if cursor >= uint(len(tokens)) {
		return false
	}
	return t.Equals(tokens[cursor])
}

// 消息打印辅助函数
func helpMessage(tokens []*Token, cursor uint, msg string) {
	var c *Token
	if cursor+1 < uint(len(tokens)) {
		c = tokens[cursor+1]
	} else {
		c = tokens[cursor]
	}
	// fmt.Printf("[%d, %d]: %s, got: %s\n", c.Loc.Line, c.Loc.Col, msg, c.Value)
	fmt.Printf("[%d, %d]: %s, near: %s\n", c.Loc.Line, c.Loc.Col, msg, c.Value)
}

func Parse(source string) (*Ast, error) {
	// 先利用词法解析
	// source 对应 select * from test;
	// tokens 对应 [select, *, from, test]
	tokens, err := Lex(source)
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
		stmt, newCursor, ok := parseStatement(tokens, cursor, TokenFromSymbol(SemicolonSymbol))
		if !ok {
			helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("failed to parse, expect statement")
		}
		cursor = newCursor // 更新cursor

		a.Statements = append(a.Statements, stmt)

		// 找找有没有分号
		atLeastOneSemicolon := false
		for expectToken(tokens, cursor, TokenFromSymbol(SemicolonSymbol)) {
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
func parseStatement(tokens []*Token, initialCursor uint, _ Token) (*Statement, uint, bool) {
	// 分别调动每个statement类型的解析函数
	cursor := initialCursor

	// 寻找SELECT
	semicolonToken := TokenFromSymbol(SemicolonSymbol)
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

	create, newCursor, ok := parseCreateTableStatement(tokens, cursor, semicolonToken)
	if ok {
		// 证明找到了Selec语句
		return &Statement{
			Kind:                 CreateKind,
			CreateTableStatement: create,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

////////////////////////////////
// 解析select 语句
// SELECT
// $expression [, ...]
// FROM
// $table-name
func parseSelectStatement(tokens []*Token, initialCursor uint, delimiter Token) (*SelectStatement, uint, bool) {
	var ok bool
	cursor := initialCursor
	// 验证第一个token是不是SELECT关键字
	_, cursor, ok = parseToken(tokens, cursor, TokenFromKeyword(SelectKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	slct := SelectStatement{}

	// 从SELECT 到 FROM 关键字这中间都被解析成item
	fromToken := TokenFromKeyword(FromKeyword)
	item, newCursor, ok := parseSelectItem(tokens, cursor, []Token{fromToken, delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	// FIXME debug一下解析到的selectItem
	debugSelectItem(item)

	slct.Item = item
	cursor = newCursor

	whereToken := TokenFromKeyword(WhereKeyword)
	limitToken := TokenFromKeyword(LimitKeyword)
	offsetToken := TokenFromKeyword(OffsetKeyword)

	// 找到from token
	_, cursor, ok = parseToken(tokens, cursor, fromToken)
	if ok {
		// 找到from之后接下来就要找表名
		tableName, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected table name after from token")
			return nil, initialCursor, false
		}

		slct.From = tableName
		cursor = newCursor
	}

	// 找到where token
	_, cursor, ok = parseToken(tokens, cursor, whereToken)
	if ok {
		// 遇到offset / limit / other delimiter的时候,就返回
		where, newCursor, ok := parseExpression(tokens, cursor, []Token{limitToken, offsetToken, delimiter}, 0)
		if !ok {
			helpMessage(tokens, cursor, "Expected WHERE conditionals")
			return nil, initialCursor, false
		}
		// 这个where可以为例如age=20这样的expression
		slct.Where = where
		cursor = newCursor
	}

	// 找到limit token
	_, cursor, ok = parseToken(tokens, cursor, limitToken)
	if ok {
		limit, newCursor, ok := parseExpression(tokens, cursor, []Token{offsetToken, delimiter}, 0)
		// 证明有limit token但是后面没有对应的值
		if !ok {
			helpMessage(tokens, cursor, "Expected LIMIT VALUE")
			return nil, initialCursor, false
		}

		slct.Limit = limit
		cursor = newCursor
	}

	// 找到offset token
	_, cursor, ok = parseToken(tokens, cursor, offsetToken)
	if ok {
		offset, newCursor, ok := parseExpression(tokens, cursor, []Token{delimiter}, 0)
		if !ok {
			helpMessage(tokens, cursor, "Expected OFFSET VALUE")
			return nil, initialCursor, false
		}
		slct.Offset = offset
		cursor = newCursor
	}

	return &slct, cursor, true
}

// REVIEW  function parseSelectItem的作用就是返回的select关键字到from关键字中间的[id, name, age]对应的某种形式
func parseSelectItem(tokens []*Token, initialCursor uint, delimiters []Token) (*[]*SelectItem, uint, bool) {
	cursor := initialCursor
	var s []*SelectItem
outer:
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		// NOTE 这一段代码检查是否遇到了分隔符/结束符
		// NOTE 举例:select id, name, age from users; 中分隔符就是分号(;)和from对应的token
		for _, delimiter := range delimiters {
			// 如果当前的
			if delimiter.Equals(current) {
				break outer
			}
		}

		var ok bool
		// 如果已经找到了部分item,就保持
		if len(s) > 0 {
			// 找到一个selectItem了,就查看下一个是不是逗号,如果不是逗号,也不是分隔符,那么就报错
			_, cursor, ok = parseToken(tokens, cursor, TokenFromSymbol(CommaSymbol))
			if !ok {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		var si SelectItem
		// REVIEW 寻找标志 * 或者as token
		_, cursor, ok = parseToken(tokens, cursor, TokenFromSymbol(AsterisSymbol))
		if ok {
			si = SelectItem{
				Asterisk: true,
			}
		} else {
			// 寻找as token
			asToken := TokenFromKeyword(AsKeyword)
			delimiters := append(delimiters, TokenFromSymbol(CommaSymbol), asToken)
			exp, newCursor, ok := parseExpression(tokens, cursor, delimiters, 0)
			if !ok {
				helpMessage(tokens, cursor, "Expected expression")
				return nil, initialCursor, false
			}

			cursor = newCursor
			si.Exp = exp

			_, cursor, ok = parseToken(tokens, cursor, asToken)
			if ok {
				id, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
				if !ok {
					helpMessage(tokens, cursor, "Expected Identifier after AS")
					return nil, initialCursor, false
				}

				cursor = newCursor
				si.As = id
			}
		}

		s = append(s, &si)
	}

	return &s, cursor, true
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
func parseInsertStatement(tokens []*Token, initialCursor uint, _ Token) (*InsertStatement, uint, bool) {
	cursor := initialCursor
	ok := false
	// 验证是否是insert语句
	_, cursor, ok = parseToken(tokens, cursor, TokenFromKeyword(InsertKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	// 找到into关键字
	_, cursor, ok = parseToken(tokens, cursor, TokenFromKeyword(IntoKeyword))
	if !ok {
		helpMessage(tokens, cursor, "Expected into")
		return nil, initialCursor, false
	}

	// 找到table Name
	table, cursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}

	// 找到VALUES关键字
	_, cursor, ok = parseToken(tokens, cursor, TokenFromKeyword(ValuesKeyword))
	if !ok {
		helpMessage(tokens, cursor, "Expected values keyword")
		return nil, initialCursor, false
	}

	// 找到左括号
	_, cursor, ok = parseToken(tokens, cursor, TokenFromSymbol(LeftBracketSymbol))
	if !ok {
		helpMessage(tokens, cursor, "Expected left paren")
		return nil, initialCursor, false
	}

	// 左括号开始就需要解析expression了
	rightParenToken := TokenFromSymbol(RightBracketSymbol)
	values, newCursor, ok := parseExpressions(tokens, cursor, []Token{rightParenToken})
	if !ok {
		helpMessage(tokens, cursor, "Expected expressions")
		return nil, initialCursor, false
	}
	cursor = newCursor

	// 找到右括号
	_, cursor, ok = parseToken(tokens, cursor, rightParenToken)
	if !ok {
		helpMessage(tokens, cursor, "Expected right paren")
		return nil, initialCursor, false
	}

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
func parseCreateTableStatement(tokens []*Token, initialCursor uint, delimiter Token) (*CreateTableStatement, uint, bool) {
	cursor := initialCursor
	// 找到CREATE
	if !expectToken(tokens, cursor, TokenFromKeyword(CreateKeyword)) {
		return nil, initialCursor, false
	}
	cursor++

	// 找到TABLE关键字
	if !expectToken(tokens, cursor, TokenFromKeyword(TableKeyword)) {
		return nil, initialCursor, false
	}
	cursor++
	// 找打tablename
	// table, newCursor, ok := parseExpression(tokens, cursor, delimiter)
	name, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor
	// 找到"("
	if !expectToken(tokens, cursor, TokenFromSymbol(LeftBracketSymbol)) {
		helpMessage(tokens, cursor, "Expected '('")
		return nil, initialCursor, false
	}
	cursor++
	// 找到column list
	cloums, newCursor, ok := parseColumnDefinitions(tokens, cursor, TokenFromSymbol(RightBracketSymbol))
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	// 找打")"
	if !expectToken(tokens, cursor, TokenFromSymbol(RightBracketSymbol)) {
		helpMessage(tokens, cursor, "Expected ')'")
		return nil, initialCursor, false
	}
	cursor++

	return &CreateTableStatement{
		Name: *name,
		Cols: cloums,
	}, cursor, true
}

// 辅助函数,用于找到列名和跟在后面的列类型
func parseColumnDefinitions(tokens []*Token, initialCursor uint, delimiter Token) (*[]*ColumnDefinition, uint, bool) {
	cursor := initialCursor
	cds := []*ColumnDefinition{}
	// 循环找, 直到遇到分隔符(这里指的是右括号)位置
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
			if !expectToken(tokens, cursor, TokenFromSymbol(CommaSymbol)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
			cursor++
		}

		// 找列名
		id, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}
		cursor = newCursor
		// 找列的类型
		ty, newCursor, ok := parseTokenKind(tokens, cursor, KeywordKind)
		if !ok {
			return nil, initialCursor, false
		}
		cursor = newCursor

		primaryKey := false
		// 寻找有没有主键关键字
		_, cursor, ok = parseToken(tokens, cursor, TokenFromKeyword(PrimaryKeyKeyword))
		if ok {
			primaryKey = true
		}

		cds = append(cds, &ColumnDefinition{
			Name:       *id,
			Datatype:   *ty,
			PrimaryKey: primaryKey,
		})
	}
	return &cds, cursor, true
}

// function parseExpressions的作用就是找到所有token对应的表达式, id对应一个表达式,age+2对应一个表达式
func parseExpressions(tokens []*Token, initialCursor uint, delimiters []Token) (*[]*Expression, uint, bool) {
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
			// 如果发现这个token就是分隔符,直接退出outer标签
			if d.Equals(current) {
				break outer
			}
		}

		// 找逗号(前提是有逗号)
		if len(exps) > 0 {
			// 不是逗号
			if !expectToken(tokens, cursor, TokenFromSymbol(CommaSymbol)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}

			cursor++
		}
		// 找expression
		exp, newCursor, ok := parseExpression(tokens, cursor, []Token{TokenFromSymbol(CommaSymbol), TokenFromSymbol(RightBracketSymbol)}, 0)
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

// parseExpression 会找到等等单个token封装的expression或者age+2这样连着的expression
func parseExpression(tokens []*Token, initialCursor uint, delimiters []Token, minBp uint) (*Expression, uint, bool) {

	cursor := initialCursor

	var exp *Expression
	// 先找有么有左括号
	_, newCursor, ok := parseToken(tokens, cursor, TokenFromSymbol(LeftBracketSymbol))
	if ok {
		cursor = newCursor
		// 右括号
		rightParenToken := TokenFromSymbol(RightBracketSymbol)
		// 有了左括号,就去寻找expression
		exp, cursor, ok = parseExpression(tokens, cursor, append(delimiters, rightParenToken), minBp)
		// 没找到
		if !ok {
			helpMessage(tokens, cursor, "Expected expression after opening paren")
			return nil, initialCursor, false
		}

		// 找到了expression,下面继续寻找右括号
		_, cursor, ok = parseToken(tokens, cursor, rightParenToken)
		if !ok {
			helpMessage(tokens, cursor, "Expected right paren after expression")
			return nil, initialCursor, false
		}

	} else {
		// 就是一开始没有找到左括号
		exp, cursor, ok = parseLiteralExpression(tokens, cursor)
		if !ok {
			return nil, initialCursor, false
		}
	}

	// 找到左边和右边括号之后
	lastCursor := cursor
outer:
	for cursor < uint(len(tokens)) {
		// 寻找分隔符
		for _, d := range delimiters {
			_, _, ok := parseToken(tokens, cursor, d)
			if ok {
				break outer
			}
		}
		// 不是分隔符的情况
		binOps := []Token{
			TokenFromKeyword(AndKeyword),
			TokenFromKeyword(OrKeyword),
			TokenFromSymbol(EqSymbol),
			TokenFromSymbol(NeqSymbol),
			TokenFromSymbol(ConcatSymbol),
			TokenFromSymbol(PlusSymbol),
		}

		var op *Token = nil
		for _, bo := range binOps {
			var t *Token
			t, cursor, ok = parseToken(tokens, cursor, bo)
			if ok {
				op = t
				break
			}
		}
		// 如果没有找到token
		if op == nil {
			helpMessage(tokens, cursor, "Expected binary operator")
			return nil, initialCursor, false
		}

		bp := op.BindingPower()
		if bp < minBp {
			cursor = lastCursor
			break
		}
		// 如果新的操作符拥有更大的binding power,我们将进行递归调用parseExpression
		// 将新的binding power作为minBp的实参
		b, newCursor, ok := parseExpression(tokens, cursor, delimiters, bp)
		if !ok {
			helpMessage(tokens, cursor, "Expected right operand")
			return nil, initialCursor, false
		}
		// REVIEW 比如select age+2, name form users;
		// 这里exp的最终结果就是{age, 2, +}
		exp = &Expression{
			Binary: &BinaryExpression{
				*exp,
				*b,
				*op,
			},
			Kind: BinaryKind,
		}

		cursor = newCursor
		lastCursor = cursor
	}

	return exp, cursor, true
}

// 找到字面量(表名users, 数字1, 字符'hello', 布尔, null)的一个Token,并封装成Expression
func parseLiteralExpression(tokens []*Token, initialCursor uint) (*Expression, uint, bool) {
	cursor := initialCursor

	kinds := []TokenKind{
		IdentifierKind,
		NumericKind,
		StringKind,
		BoolKind,
		NullKind,
	}

	for _, kind := range kinds {
		t, newCursor, ok := parseTokenKind(tokens, cursor, kind)
		if ok {
			return &Expression{
				Kind:    LiteralKind,
				Literal: t,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}

// parseToken辅助函数会找到特定kind的token消费掉,相比parseToken范围更大,找的是一类Token
func parseTokenKind(tokens []*Token, initialCursor uint, kind TokenKind) (*Token, uint, bool) {
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

// function parseToken 将会把匹配的token给消费掉,相比parseTokenKind范围更精确
func parseToken(tokens []*Token, initialCursor uint, t Token) (*Token, uint, bool) {
	cursor := initialCursor
	// 检查有么有越界
	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	if p := tokens[cursor]; t.Equals(p) {
		return p, cursor + 1, true
	}

	return nil, initialCursor, false
}
