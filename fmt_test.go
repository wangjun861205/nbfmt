package nbfmt

import (
	"fmt"
	"log"
	"testing"
)

type Table struct {
	TableName string
}

var src = `hello world
	{{ if X[123] }}
		{{ abc }}
	{{ endif }}
	{{ switch X[Z.w].Y }}
		{{ case 1, 2, 3 }}
			hello world
		{{ case "test" }}
			{{ test }}
		{{ default }}
			success
	{{ endswitch }}
test`

func TestFmt(t *testing.T) {
	l, err := parseStmt(src)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range l {
		for _, o := range s.objects {
			fmt.Println(o.typ)
		}
	}
}
