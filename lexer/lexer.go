package lexer

import (
	"fmt"
	"strings"
)

// 定义位置的结构体
type Location struct {
	//行
	Line uint
	//列
	Col uint
}

// 关键字类型
type Keyword string

// 定义默认的关键字
const (
	SelectKeyword Keyword = "select"
	IntoKeyword   Keyword = "into"
	FromKeyword   Keyword = "from"
	CreateKeyword Keyword = "create"
	CreatedKeyword Keyword = "created"
	AsKeyword     Keyword = "as"
	TableKeyword  Keyword = "table"
	InsertKeyword Keyword = "insert"
	WhereKeyword  Keyword = "where"
	ValuesKeyword Keyword = "values"
	IntKeyword    Keyword = "int"  // 代表支持int类型
	TextKeyword   Keyword = "text" // 代表支持text类型
)

// 定义标志(比如括号这种)
type Symbol string

const (
	SemicolonSymbol    Symbol = ";"
	AsterisSymbol      Symbol = "*"
	CommaSymbol        Symbol = ","
	LeftBracketSymbol  Symbol = "("
	RightBracketSymbol Symbol = ")"
)

// 定义token的各种类型
type TokenKind uint

const (
	KeywordKind    TokenKind = iota // 关键字
	SymbolKind                      // 标志
	IdentifierKind                  // 标识符
	StringKind                      // 字符串
	NumericKind                     // 数字
)

// 定义Token,一个token必须有值 类型 位置
type Token struct {
	Value string
	Kind  TokenKind
	Loc   Location
}

// 定义游标
type cursor struct {
	pointer uint
	loc     Location
}

// 判断两个标识符是否相等
func (t *Token) Equals(other *Token) bool {
	return t.Value == other.Value && t.Kind == other.Kind
}

// 定义词法解析器
type lexer func(string, cursor) (*Token, cursor, bool)

// function lex 的目的就是将SQL源解析为token数组
func Lex(source string) ([]*Token, error) {
	// select * from tablename;
	tokens := []*Token{}
	cur := cursor{}

lex:
	for cur.pointer < uint(len(source)) {
		// 将所有的解析函数放进一个数组
		lexers := []lexer{lexKeyword, lexSymbol, lexString, lexNumeric, lexIdentifier}
		for _, l := range lexers {
			if token, newCursor, ok := l(source, cur); ok {
				// 找到了为tokenKind中符合的那个
				// 当前的位置往后移动
				cur = newCursor
				// 并且忽略token有效但是语法类似于一个新行这样的（见函数lexSymbol）
				if token != nil {
					tokens = append(tokens, token)
				}
				continue lex // 继续从头开始
			}
		}
		hint := ""
		if len(tokens) > 0 {
			// 代表SQL源中出现无法解析的字符
			hint = " after " + tokens[len(tokens)-1].Value
		}
		return nil, fmt.Errorf("unable to lex Token %s at %d %d", hint, cur.loc.Line, cur.loc.Col)
	}

	return tokens, nil
}

// 接下来就要对基本的token们挨个写辅助函数
// function lexNumeric代表数字解析(这部分是最难的)
func lexNumeric(source string, ic cursor) (*Token, cursor, bool) {
	// "insert into tablename values(2,1.09e10)"
	// 有效的数字如下:
	// 42
	// 3.5
	// 4.
	// .001
	// 5e2
	// 1.925e-3
	cur := ic

	periodFound := false    // 看看是否发现了小数点'.'
	expMarkerFound := false // 看看是否发现了指数'e'

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer] // 先保存当前字符
		cur.loc.Col++            // 让列游标向后移动

		// 接下来进行判断
		isDigit := c >= '0' && c <= '9'
		isPeriod := c == '.'
		isExpMarker := c == 'e'

		// 必须以小数点或者数字开头,否则返回
		if cur.pointer == ic.pointer {
			// 不是以数字和 period开头的就返回,因为这是数字解析函数
			if !isDigit && !isPeriod {
				// return nil, cur, false //错误 2021年4月25日20:53:09
				return nil, ic, false
			}
			periodFound = isPeriod
			continue
		}
		// 如果是小数点开头
		if isPeriod {
			// 但是小数点后面又跟了一个小数点,就返回,证明这个token不是数字类型
			if periodFound {
				return nil, ic, false
			}
			periodFound = true
			continue
		}
		// 如果是e开头
		if isExpMarker {
			if expMarkerFound {
				return nil, ic, false
			}

			// e后面不能跟小数点
			periodFound = true
			expMarkerFound = true

			// e后面必须跟数字(可正可负)
			if cur.pointer == uint(len(source)-1) {
				return nil, ic, false
			}
			// 找到当前字符的后一个字符,也就是e后面的字符
			cNext := source[cur.pointer+1]
			// 判断是不是数字
			if cNext == '-' || cNext == '+' {
				cur.pointer++
				cur.loc.Col++
			}
			continue
		}

		// 如果不是数字
		if !isDigit {
			break
		}
	}

	// 如果没有发现数字
	if cur.pointer == ic.pointer {
		return nil, ic, false
	}
	return &Token{
		Value: source[ic.pointer:cur.pointer],
		Kind:  NumericKind,
		Loc:   ic.loc,
	}, cur, true
}

// function lexKeyword代表关键字解析
func lexKeyword(source string, ic cursor) (*Token, cursor, bool) {
	cur := ic
	// 先准备Keyword数组
	Keywords := []Keyword{
		SelectKeyword,
		InsertKeyword,
		ValuesKeyword,
		TableKeyword,
		IntoKeyword,
		FromKeyword,
		WhereKeyword,
		TextKeyword,
		CreateKeyword,
		CreatedKeyword,
		IntKeyword,
	}
	var options []string
	for _, k := range Keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)

	if match == "" {
		return nil, ic, false
	}

	// 比如match == "create"， 那pointer就要从原来的值(初始为0)增加到len("create"), ic.loc.Col也要增加
	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.Col = ic.loc.Col + uint(len(match))

	return &Token{
		Value: match,
		Kind:  KeywordKind,
		Loc:   ic.loc,
	}, cur, true
}

// function lexSymbol代表标志解析
func lexSymbol(source string, ic cursor) (*Token, cursor, bool) {
	c := source[ic.pointer]
	cur := ic
	// 先往后加一
	cur.pointer++
	cur.loc.Col++

	// 判断是不是已经定义的symbol
	switch c {
	// 需要被丢掉的
	case '\n':
		cur.loc.Line++
		cur.loc.Col = 0
		fallthrough // 继续往下
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true // 这样并不算错,但是得不到有效的token
	}
	// 应该被保留的语法
	// 有效的symbol集合
	symbols := []Symbol{
		CommaSymbol, 
		AsterisSymbol, 
		SemicolonSymbol, 
		LeftBracketSymbol, 
		RightBracketSymbol,
	}
	// TODO
	var options []string
	for _, s := range symbols {
		options = append(options, string(s))
	}
	// Use 'ic' not 'cur'
	match := longestMatch(source, ic, options)
	// 不认识的character
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.Col = ic.loc.Col + uint(len(match))

	return &Token{
		Value: match,
		Loc:   ic.loc,
		Kind:  SymbolKind,
	}, cur, true
}

// function lexIdentifier代表标识符解析
func lexIdentifier(source string, ic cursor) (*Token, cursor, bool) {
	// 将双引号的标识符分开讨论
	if token, newCursor, ok := lexCharacterDelimited(source, ic, '"'); ok {
		// 2021年4月24日16:48:15 补充
		token.Kind = IdentifierKind
		return token, newCursor, true
	}

	cur := ic
	c := source[cur.pointer]

	// 判断是否是字母
	isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	// 如果不是字母就返回错误
	if !isAlphabetical {
		return nil, ic, false
	}
	cur.pointer++
	cur.loc.Col++

	// var value []byte
	// value = append(value, c)
	value := []byte{c}
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c = source[cur.pointer]

		// 其他的字符也算
		isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumeric := c >= '0' && c <= '9'
		if isAlphabetical || isNumeric || c == '$' || c == '_' {
			value = append(value, c)
			cur.loc.Col++
			continue
		}
		// 如果不算我们所规定的字符,那么就不计入
		break
	}
	// 如果只计入一开始的一个字符,后面啥也没有,这次解析就是失败的
	if len(value) == 0 {
		return nil, ic, false
	}

	return &Token{
		Value: strings.ToLower(string(value)), // 双引号里面的标识符都是大小写敏感的
		Kind:  IdentifierKind,
		Loc:   ic.loc,
	}, cur, true
}

// function lexString 代表字符串解析
// 字符串必须以单引号开头,单引号结尾
func lexString(source string, ic cursor) (*Token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

// 这是解析字符串的辅助函数,之所以把它抽取出来是因为在标识符解析中我们还会用到
func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*Token, cursor, bool) {
	cur := ic

	// 如果到了结尾
	if len(source[cur.pointer:]) == 0 {
		return nil, ic, false
	}

	// 如果开头不是指定分隔符
	if source[cur.pointer] != delimiter {
		return nil, ic, false
	}

	// 证明第一个字符是分隔符 '
	cur.pointer++
	cur.loc.Col++

	var value []byte
	for ; cur.pointer < uint(len(source)); cur.pointer++ {

		c := source[cur.pointer]

		// 如果第一个字符(')的下一个字符还是(')
		if c == delimiter {
			// 如果已经越界了或者下下个字符不是(')
			if cur.pointer+1 >= uint(len(source)) || source[cur.pointer+1] != delimiter {
				// 补充 2021年4月25日21:36:55
				cur.pointer++
				cur.loc.Col++
				return &Token{
					Value: string(value),
					Loc:   ic.loc,  // 注意
					Kind:  StringKind,
				}, cur, true
			}
			// 如果下一个字符没有越界或者不是分隔符的话,继续添加到value
			value = append(value, delimiter)
			// 这次是分隔符,并且下一个也是分隔符,先把这次的append上去,
			// 然后索引继续往后
			cur.loc.Col++
			cur.pointer++
		}
		// 如果不是分隔符就一直append到value
		value = append(value, c)
		cur.loc.Col++
	}

	return nil, ic, false
}

// 辅助函数:最长匹配原则, 从给定的cursor遍历source从而找到和options中最为匹配的子字符串
func longestMatch(source string, ic cursor, options []string) string {
	// fmt.Printf("sourcr的长度是: %d\n", len(source))
	var value []byte
	var skipList []int
	var match string

	cur := ic

	for cur.pointer < uint(len(source)) {
		value = append(value, strings.ToLower(string(source[cur.pointer]))...)
		cur.pointer++

	match:
		for i, option := range options {
			for _, skip := range skipList {
				// 设想: 如果所有keyword的开头字母都不一样,下面这个if就永远执行不了
				if i == skip {
					continue match
				}
			}

			// 处理像INT vs INTO这样的情况
			if option == string(value) {
				skipList = append(skipList, i)
				if len(option) > len(match) {
					match = option
				}

				continue
			}

			sharesPrefix := string(value) == option[:cur.pointer-ic.pointer]
			tooLong := len(value) > len(option)
			if tooLong || !sharesPrefix {
				skipList = append(skipList, i)
			}
		}

		// 因为是最长匹配,按道理就算string(value) == option 了，也要继续下一个for loop
		if len(skipList) == len(options) {
			break
		}
	}
	fmt.Printf("+++++++skipList 的值是%v\n", skipList)
	fmt.Printf("-----------------------------------------------------------------------------------------------------cur.Pointer的值是 %d\n", cur.pointer)
	fmt.Printf("===============================================match的结果值是%s\n", match)
	return match
}
