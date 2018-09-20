package nbfmt

import (
	"fmt"
	"log"
	"testing"
)

type Table struct {
	TableName string
}

var src = `{{ switch hello[1] }}
{{ case 9 }}
9
{{ case 8 }}
8
{{ case 7 }}
7
{{ default }}
-1
{{ endswitch }}`

func TestFmt(t *testing.T) {
	l, err := parseStmt(src)
	if err != nil {
		log.Fatal(err)
	}
	temp, err := genTemplate(l)
	if err != nil {
		log.Fatal(err)
	}
	s, err := temp.eval(map[string]interface{}{"hello": []int{9, 8, 7, 6}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s)
	// fmt.Println(expr)
	// fmt.Println(expr.nextExpr)
	// fmt.Println(expr.nextExpr.nextExpr)
	// fmt.Println(expr.pop())
}
