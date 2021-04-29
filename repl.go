package yunsql

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/olekukonko/tablewriter"
)

func RunRepl(b Backend) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("<(*^▽^*)>")
	fmt.Println("欢迎来到云云数据库!")
repl:
	for {
		fmt.Print("# ")
		text, err := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)
		if err == readline.ErrInterrupt {
			if len(text) == 0 {
				break
			} else {
				continue repl
			}
		} else if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error while reading line:", err)
			continue repl
		}

		// parser := Parser{}

		trimmed := strings.TrimSpace(text)
		if trimmed == "quit" || trimmed == "exit" || trimmed == "\\q" {
			break
		}

		// if trimmed == "\\dt" {
		// 	debugTables(b)
		// 	continue
		// }

		// if strings.HasPrefix(trimmed, "\\d") {
		// 	name := strings.TrimSpace(trimmed[len("\\d"):])
		// 	debugTable(b, name)
		// 	continue
		// }

		//parseOnly := false
		if strings.HasPrefix(trimmed, "\\p") {
			text = strings.TrimSpace(trimmed[len("\\p"):])
			//parseOnly = true
		}

		ast, err := Parse(text)
		if err != nil {
			fmt.Println("Error while parsing:", err)
			continue repl
		}

		for _, stmt := range ast.Statements {
			// if parseOnly {
			// 	fmt.Println(stmt.GenerateCode())
			// 	continue
			// }

			switch stmt.Kind {
			// case CreateIndexKind:
			// 	err = b.CreateIndex(ast.Statements[0].CreateIndexStatement)
			// 	if err != nil {
			// 		fmt.Println("Error adding index on table:", err)
			// 		continue repl
			// 	}
			case CreateKind:
				err = b.CreateTable(ast.Statements[0].CreateStatement)
				if err != nil {
					fmt.Println("Error creating table:", err)
					continue repl
				}
			// case DropTableKind:
			// 	err = b.DropTable(ast.Statements[0].DropTableStatement)
			// 	if err != nil {
			// 		fmt.Println("Error dropping table:", err)
			// 		continue repl
			// 	}
			case InsertKind:
				err = b.Insert(stmt.InsertStatement)
				if err != nil {
					fmt.Println("Error inserting values:", err)
					continue repl
				}
			case SelectKind:
				err := doSelect(b, stmt.SelectStatement)
				if err != nil {
					fmt.Println("Error selecting values:", err)
					continue repl
				}
			}
		}

		fmt.Println("ok")
	}
}

func doSelect(mb Backend, slct *SelectStatement) error {
	results, err := mb.Select(slct)
	if err != nil {
		return err
	}

	if len(results.Rows) == 0 {
		fmt.Println("(no results)")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{}
	for _, col := range results.Columns {
		header = append(header, col.Name)
	}
	table.SetHeader(header)
	table.SetAutoFormatHeaders(false)

	rows := [][]string{}
	for _, result := range results.Rows {
		row := []string{}
		for i, cell := range result {
			typ := results.Columns[i].Type
			r := ""
			switch typ {
			case IntType:
				i := cell.AsInt()
				// if i != nil {
				// 	r = fmt.Sprintf("%d", *i)
				// }
				r = fmt.Sprintf("%d", i)
			case TextType:
				s := cell.AsText()
				// if s != nil {
				// 	r = *s
				// }
				r = s
			case BoolType:
				b := cell.AsBool()
				// if b != nil {
				// 	r = "t"
				// 	if !*b {
				// 		r = "f"
				// 	}
				// }
				if b {
					r = "true"
				} else {
					r = "false"
				}
			}

			row = append(row, r)
		}

		rows = append(rows, row)
	}

	table.SetBorder(false)
	table.AppendBulk(rows)
	table.Render()

	if len(rows) == 1 {
		fmt.Println("(1 result)")
	} else {
		fmt.Printf("(%d results)\n", len(rows))
	}

	return nil
}
