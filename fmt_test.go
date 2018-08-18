package nbfmt

import (
	"fmt"
	"testing"
)

type T struct {
	Name  string
	Value int
	U     U
}

type U struct {
	Name  string
	Value float64
}

func TestFmt(t *testing.T) {
	temp := `test value 2 {{1}} value 1 {{U>0}}`
	ts := T{"hello", 100, U{"test", 1.23}}
	s, err := Fmt(temp, ts)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(s)
}
