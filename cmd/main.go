package main

import (
	"github.com/database-from-zero-to-one/yunsql"
)

func main() {
	mb := yunsql.NewMemoryBackend()
	yunsql.RunRepl(mb)
}
