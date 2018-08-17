package strfmt

import (
	"fmt"
	"testing"
)

type T struct {
	Name  *string
	Value *int
}

func TestFmt(t *testing.T) {
	temp := `test value 2 {{Name:%s}} value 1 {{Value:%d}} value 3 {{Test:%s}}`
	name := "hello"
	ts := T{&name, nil}
	s, err := FmtByName(temp, ts)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(s)
}
