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
	{{ switch X }}
		{{ case 1 }}
			hello world
		{{ case 2 }}
			test
		{{ default }}
			success
	{{ endswitch }}
test`

func TestFmt(t *testing.T) {
	// l, err := split(src)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for i := range l {
	// 	if l[i].typ == blk {
	// 		err = splitBlock(&l[i])
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 	}
	// }
	// // for _, b := range l {
	// // 	fmt.Println(b)
	// // }
	// err = parseObj(&l)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// // for _, c := range l {
	// // 	fmt.Println(c.objects)
	// // }
	// blockList := make([]block, 0, 128)
	// for len(l) > 0 {
	// 	b, err := parseBlock(&l)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	blockList = append(blockList, b)
	// }
	// ib := blockList[1].(*ifBlock)
	// icb := ib.subBlocks[0].(*ifcaseBlock)
	// ol := icb.expr[1:]
	// group, err := parseExprGroup(&ol)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// group.split()
	// expr, err := parseExpr(group)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(*expr.subExprs[0].subExprs[0].rightVal)
	temp, err := parseTemplate(src)
	if err != nil {
		log.Fatal(err)
	}
	s, err := temp.run(map[string]interface{}{"X": 5})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s)
}
