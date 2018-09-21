package nbfmt

import (
	"fmt"
	"log"
	"testing"
)

type Table struct {
	TableName string
}

const src = `
{{ for k, v in m }}
	{{ if v }}
		{{ switch k }}
			{{ case "hello" }}
				{{ k }}
			{{ default }}
				not hello
		{{ endswitch }}
	{{ elseif v == false }}
		this is false
	{{ endif }}
{{ endfor }}
`

func TestFmt(t *testing.T) {
	s, err := Fmt(src, map[string]interface{}{
		"m": map[string]bool{
			"hell":  true,
			"world": false,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s)
}
