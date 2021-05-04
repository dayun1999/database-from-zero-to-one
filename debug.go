package yunsql

import (
	"fmt"
)

// NOTE function debugRows 调试insert中获取到的rows
func debugRows(rows [][]MemoryCell) {
	for i := 0; i < len(rows); i++ {
		result := ""
		for j := 0; j < len(rows[0]); j++ {
			// 这里使用%v而不是%s的原因是,MemoryCell的值不一定是可以打印的
			result += fmt.Sprintf("  %v", rows[i][j])
		}
		fmt.Printf("第%d行的数据: %v\n", i, result)
	}
}

// NOTE function debugTokens 调试解析得到的tokens
func debugTokens(tokens []*Token) {
	if len(tokens) == 0 {
		return
	}
	result := ""
	fmt.Println("--------------------------------")
	fmt.Println("词法解析结果如下:")
	for i, token := range tokens {
		switch token.Kind {
		case StringKind:
			result = fmt.Sprintf("序号%d---%s----类型为  StringKind \n", i, token.Value)
		case NumericKind:
			result = fmt.Sprintf("序号%d---%s----类型为  NumericKind \n", i, token.Value)
		case IdentifierKind:
			result = fmt.Sprintf("序号%d---%s----类型为  IdentifierKind \n", i, token.Value)
		case BoolKind:
			result = fmt.Sprintf("序号%d---%s----类型为  BoolKind \n", i, token.Value)
		case KeywordKind:
			result = fmt.Sprintf("序号%d---%s----类型为  KeywordKind \n", i, token.Value)
		case SymbolKind:
			result = fmt.Sprintf("序号%d---%s----类型为  SymbolKind \n", i, token.Value)
		}
		fmt.Printf("%s", result)
	}
	fmt.Println("--------------------------------")
}

// FIXME function debugSelectItem 调试得到的selectItem
// !! EXPIREDhas some bugs
// func debugSelectItem(items *[]*SelectItem) {
// 	fmt.Println("======debugSelectItem=======")
// 	fmt.Printf("当前输入的len(*[]*SelectItem)=%d\n", len(*items))
// 	for i, item := range *items {
// 		token := item.Exp.Literal.Value
// 		fmt.Printf("第%d个slectItem的值为%v\n", i, token)
// 	}

// 	fmt.Println("================================")
// }
