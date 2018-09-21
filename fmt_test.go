package nbfmt

import (
	"fmt"
	"log"
	"testing"
)

type Table struct {
	BoolList []bool
}

const src = `
{{ if !table.BoolList[2] }}
	success
{{ else }}
	not false
{{ endif }}
`

func TestFmt(t *testing.T) {
	s, err := Fmt(src, map[string]interface{}{
		"table": Table{[]bool{false, true, true}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s)
}
