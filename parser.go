package nbfmt

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var intRe = regexp.MustCompile(`^-?\d+$`)
var floatRe = regexp.MustCompile(`^-?\d+\.\d+$`)
var varIdentRe = regexp.MustCompile(`^[a-zA-z_][\w_]*$`)
var chrIdentRe = regexp.MustCompile(`^'.'$`)
var strIdentRe = regexp.MustCompile("^[\"|`].*[\"|`]$")
var boolIdentRe = regexp.MustCompile(`^(true|false)$`)

func parseIdent(s string) (*ident, error) {
	switch s {
	case "if":
		return &ident{src: s, typ: ifIdent}, nil
	case "elseif":
		return &ident{src: s, typ: elseifIdent}, nil
	case "else":
		return &ident{src: s, typ: elseIdent}, nil
	case "endif":
		return &ident{src: s, typ: endifIdent}, nil
	case "for":
		return &ident{src: s, typ: forIdent}, nil
	case "in":
		return &ident{src: s, typ: inIdent}, nil
	case "endfor":
		return &ident{src: s, typ: endforIdent}, nil
	case "switch":
		return &ident{src: s, typ: switchIdent}, nil
	case "case":
		return &ident{src: s, typ: caseIdent}, nil
	case "default":
		return &ident{src: s, typ: defaultIdent}, nil
	case "endswitch":
		return &ident{src: s, typ: endswitchIdent}, nil
	case ".":
		return &ident{src: s, typ: dotIdent}, nil
	case ",":
		return &ident{src: s, typ: commaIdent}, nil
	case "(":
		return &ident{src: s, typ: leftParenthesisIdent}, nil
	case ")":
		return &ident{src: s, typ: rightParenthesisIdent}, nil
	case "[":
		return &ident{src: s, typ: leftBracketIdent}, nil
	case "]":
		return &ident{src: s, typ: rightBracketIdent}, nil
	case "+":
		return &ident{src: s, typ: plugIdent}, nil
	case "-":
		return &ident{src: s, typ: subIdent}, nil
	case "*":
		return &ident{src: s, typ: asteriskIdent}, nil
	case "/":
		return &ident{src: s, typ: divIdent}, nil
	case "!":
		return &ident{src: s, typ: exclamationIdent}, nil
	case "<":
		return &ident{src: s, typ: lessThanIdent}, nil
	case ">":
		return &ident{src: s, typ: greatThanIdent}, nil
	case "<=":
		return &ident{src: s, typ: lessThanEqualIdent}, nil
	case ">=":
		return &ident{src: s, typ: greatThanEqualIdent}, nil
	case "==":
		return &ident{src: s, typ: equalIdent}, nil
	case "!=":
		return &ident{src: s, typ: notEqualIdent}, nil
	case "&&":
		return &ident{src: s, typ: andIdent}, nil
	case "||":
		return &ident{src: s, typ: orIdent}, nil
	case "nil":
		return &ident{src: s, typ: nilIdent}, nil
	default:
		switch {
		case boolIdentRe.MatchString(s):
			return &ident{src: s, typ: boolIdent}, nil
		case intRe.MatchString(s):
			return &ident{src: s, typ: intIdent}, nil
		case floatRe.MatchString(s):
			return &ident{src: s, typ: floatIdent}, nil
		case varIdentRe.MatchString(s):
			return &ident{src: s, typ: varIdent}, nil
		case chrIdentRe.MatchString(s):
			return &ident{src: s, typ: byteIdent}, nil
		case strIdentRe.MatchString(s):
			return &ident{src: s, typ: strIdent}, nil
		default:
			return nil, fmt.Errorf("nbfmt.parseIdent() parse error: invalid ident (%s)", s)
		}
	}
}

func parseIdents(l []*stmt) error {
OUTER:
	for _, s := range l {
		if len(s.src) < 2 || s.src[:2] != "{{" {
			continue
		}
		reader := strings.NewReader(strings.Trim(s.src, "{} "))
		builder := strings.Builder{}
		ctx := "empty"
		reflush := func() error {
			ctx = "empty"
			id, err := parseIdent(builder.String())
			if err != nil {
				return err
			}
			s.idents = append(s.idents, id)
			builder.Reset()
			return nil
		}
		checkPrev := func() string {
			if len(s.idents) == 0 {
				return "empty"
			}
			switch s.idents[len(s.idents)-1].typ {
			case ifIdent, elseifIdent, elseIdent, endifIdent, forIdent, inIdent, endforIdent, switchIdent, caseIdent, defaultIdent, endswitchIdent:
				return "keyword"
			case intIdent:
				return "int"
			case floatIdent:
				return "float"
			case boolIdent:
				return "bool"
			case strIdent:
				return "str"
			case varIdent:
				return "var"
			case leftBracketIdent, rightBracketIdent, leftParenthesisIdent, rightParenthesisIdent, commaIdent:
				return "punctuation"
			case dotIdent, lessThanIdent, lessThanEqualIdent, greatThanIdent, greatThanEqualIdent, notEqualIdent, equalIdent, andIdent, orIdent,
				asteriskIdent, plugIdent, subIdent, divIdent:
				return "operator"
			case nilIdent:
				return "nil"
			default:
				panic(s.idents[len(s.idents)-1].src)
			}
		}
		for {
			b, err := reader.ReadByte()
			if err != nil {
				if err == io.EOF {
					switch ctx {
					case "empty":
						continue OUTER
					case "str":
						return fmt.Errorf("nbfmt.parseIdents() error: incompleted string in statement (%s)\n", s)
					default:
						err := reflush()
						if err != nil {
							return err
						}
						continue OUTER
					}
				} else {
					return err
				}
			}
			switch b {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "empty", "operator", "punctuation", "keyword":
						ctx = "int"
						builder.WriteByte(b)
					default:
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					}
				case "str", "byte":
					builder.WriteByte(b)
				case "keyword":
					ctx = "var"
					builder.WriteByte(b)
				case "var", "int", "float":
					builder.WriteByte(b)
				case "operator", "punctuation":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "int"
					builder.WriteByte(b)
				default:
					builder.WriteByte(b)
					return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
				}
			case '-':
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "empty", "operator", "punctuation", "keyword":
						ctx = "int"
						builder.WriteByte(b)
					case "int", "float", "var":
						ctx = "operator"
						builder.WriteByte(b)
					default:
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					}
				case "str", "byte":
					builder.WriteByte(b)
				case "operator", "punctuation":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "int"
					builder.WriteByte(b)
				case "int", "float", "var":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "operator"
					builder.WriteByte(b)
				default:
					builder.WriteByte(b)
					return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
				}
			case '"':
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "empty", "operator", "punctuation", "keyword":
						ctx = "str"
						builder.WriteByte(b)
					default:
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					}
				case "str":
					builder.WriteByte(b)
					err := reflush()
					if err != nil {
						return err
					}
				case "byte":
					builder.WriteByte(b)
				default:
					builder.WriteByte(b)
					return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
				}
			case '\'':
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "empty", "operator", "punctuation":
						ctx = "byte"
						builder.WriteByte(b)
					default:
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					}
				case "str":
					builder.WriteByte(b)
				case "byte":
					builder.WriteByte(b)
					err := reflush()
					if err != nil {
						return err
					}
				case "operator", "punctuation":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "byte"
					builder.WriteByte(b)
				default:
					builder.WriteByte(b)
					return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
				}
			case '.':
				switch ctx {
				case "str", "byte":
					builder.WriteByte(b)
				case "var":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "operator"
					builder.WriteByte(b)
				case "int":
					ctx = "float"
					builder.WriteByte(b)
				default:
					builder.WriteByte(b)
					return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
				}
			case '>', '<', '=', '!', '+', '*', '/', '&', '|':
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "empty", "var", "str", "int", "float", "bool", "punctuation", "keyword":
						ctx = "operator"
						builder.WriteByte(b)
					default:
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					}
				case "var", "int", "float", "bool":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "operator"
					builder.WriteByte(b)
				case "operator", "str", "byte":
					builder.WriteByte(b)
				case "punctuation":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "operator"
					builder.WriteByte(b)
				default:
					builder.WriteByte(b)
					return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
				}
			case '(', ')', '[', ']':
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "empty", "operator", "punctuation", "keyword":
						ctx = "punctuation"
						builder.WriteByte(b)
					default:
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					}
				case "str", "byte":
					builder.WriteByte(b)
				default:
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "punctuation"
					builder.WriteByte(b)
				}
			case ',':
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "var", "str", "chr", "int", "float", "bool":
						ctx = "punctuation"
						builder.WriteByte(b)
					default:
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					}
				case "str", "byte":
					builder.WriteByte(b)
				default:
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "punctuation"
					builder.WriteByte(b)
				}
			case ' ':
				switch ctx {
				case "empty":
					continue
				case "str", "byte":
					builder.WriteByte(b)
				default:
					err := reflush()
					if err != nil {
						return err
					}
				}
			default:
				switch ctx {
				case "empty":
					switch checkPrev() {
					case "int", "float", "bool":
						builder.WriteByte(b)
						return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
					default:
						ctx = "var"
						builder.WriteByte(b)
					}
				case "str", "byte", "var":
					builder.WriteByte(b)
				case "operator", "punctuation":
					err := reflush()
					if err != nil {
						return err
					}
					ctx = "var"
					builder.WriteByte(b)
				default:
					builder.WriteByte(b)
					return fmt.Errorf("nbfmt.parseIdents() error: invalid ident (%s) in statement (%s)\n", builder.String(), s)
				}
			}
		}
	}
	return nil
}

func parseStmt(src string) ([]*stmt, error) {
	ctx := "temp"
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	l := make([]*stmt, 0, 128)
	reader := bufio.NewReader(strings.NewReader(src))
	for {
		char, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				switch ctx {
				case "temp", "lbrace":
					if buf.Len() > 0 {
						s := &stmt{src: buf.String()}
						l = append(l, s)
					}
					err := parseIdents(l)
					if err != nil {
						return nil, err
					}
					parseStmtType(l)
					return l, nil
				default:
					return nil, fmt.Errorf("nbfmt.parseStmt() parse error: statement is not complate (%s)\n", buf.String())
				}
			} else {
				return nil, err
			}
		}
		switch char {
		case '{':
			switch ctx {
			case "temp":
				ctx = "lbrace"
				buf.WriteByte(char)
			case "lbrace":
				ctx = "dlbrace"
				buf.WriteByte(char)
			case "dlbrace":
				buf.WriteByte(char)
			default:
				buf.WriteByte(char)
				return nil, fmt.Errorf("nbfmt.parseStmt() parse error: invalid statement syntax (%s)\n", buf.String())
			}
		case '}':
			switch ctx {
			case "temp":
				buf.WriteByte(char)
			case "lbrace":
				ctx = "temp"
				buf.WriteByte(char)
			case "dlbrace":
				buf.WriteByte(char)
				return nil, fmt.Errorf("nbfmt.parseStmt() parse error: invalid statement syntax (%s)\n", buf.String())
			case "rbrace":
				ctx = "temp"
				buf.WriteByte(char)
				s := &stmt{src: buf.String()}
				l = append(l, s)
				buf.Reset()
			case "stmt":
				ctx = "rbrace"
				buf.WriteByte(char)
			default:
				buf.WriteByte(char)
				return nil, fmt.Errorf("nbfmt.parseStmt() parse error: unsupported context (%s) in statement (%s)\n", ctx, buf.String())
			}
		default:
			switch ctx {
			case "temp":
				buf.WriteByte(char)
			case "lbrace":
				ctx = "temp"
				buf.WriteByte(char)
			case "dlbrace":
				ctx = "stmt"
				buf.Truncate(buf.Len() - 2)
				if buf.Len() > 0 {
					s := &stmt{src: buf.String()}
					l = append(l, s)
					buf.Reset()
				}
				buf.WriteString("{{")
				buf.WriteByte(char)
			case "stmt":
				buf.WriteByte(char)
			default:
				buf.WriteByte(char)
				return nil, fmt.Errorf("nbfmt.parseStmt() parse error: invalid statement syntax (%s)\n", buf.String())
			}
		}
	}
}

func parseStmtType(l []*stmt) {
	for _, s := range l {
		switch len(s.idents) {
		case 0:
			s.typ = templatestmt
		default:
			switch s.idents[0].typ {
			case ifIdent:
				s.typ = ifstmt
			case elseifIdent:
				s.typ = elseifstmt
			case elseIdent:
				s.typ = elsestmt
			case endifIdent:
				s.typ = endifstmt
			case forIdent:
				s.typ = forstmt
			case endforIdent:
				s.typ = endforstmt
			case switchIdent:
				s.typ = switchstmt
			case caseIdent:
				s.typ = casestmt
			case defaultIdent:
				s.typ = defaultstmt
			case endswitchIdent:
				s.typ = endswitchstmt
			default:
				s.typ = valuestmt
			}
		}
	}
}

func parseExpression(identList *[]*ident, isGroup bool, isIndex bool) (*expression, error) {
	if len(*identList) == 0 {
		return nil, nil
	}
	pop := func() *ident {
		if len(*identList) == 0 {
			return nil
		}
		id, remain := (*identList)[0], (*identList)[1:]
		*identList = remain
		return id
	}

	push := func(id *ident) {
		nl := make([]*ident, len(*identList)+1)
		nl[0] = id
		copy(nl[1:], *identList)
		*identList = nl
	}

	peek := func() *ident {
		if len(*identList) == 0 {
			return nil
		}
		return (*identList)[0]
	}

	str := func() string {
		builder := strings.Builder{}
		for _, id := range *identList {
			builder.WriteString(id.src + " ")
		}
		return builder.String()
	}
	e := &expression{}
	id := pop()
	var deref bool
	switch id.typ {
	case asteriskIdent:
		nextIdent := peek()
		if nextIdent == nil || nextIdent.typ != varIdent {
			return nil, fmt.Errorf("nbfmt.parseExpression() error: invalid asterisk ident (%s) in expression (%s)\n", id.src, str())
		}
		e.operator = &derefOperator
		deref = true
	case varIdent, strIdent, byteIdent, intIdent, floatIdent, boolIdent, nilIdent:
		e.ident = id
	case leftParenthesisIdent:
		subExpr, err := parseExpression(identList, true, false)
		if err != nil {
			return nil, err
		}
		e.subExpr = subExpr
		subExpr.parExpr = e
	case leftBracketIdent:
		subExpr, err := parseExpression(identList, false, true)
		if err != nil {
			return nil, err
		}
		e.subExpr = subExpr
		subExpr.parExpr = e
	default:
		return nil, fmt.Errorf("nbfmt.parseExpression() error: invalid ident (%s) in expression (%s)\n", id.src, str())
	}
	if !deref {
		op := pop()
		if op == nil {
			return e, nil
		}
		switch op.typ {
		case leftBracketIdent:
			e.operator = &indexOperator
			push(op)
		case rightParenthesisIdent:
			if isGroup {
				return e, nil
			}
			return nil, fmt.Errorf("nbfmt.parseExpression() error: invalid right parenthesis ident (%s) in expression (%s)\n", op.src, str())
		case rightBracketIdent:
			if isIndex {
				return e, nil
			}
			return nil, fmt.Errorf("nbfmt.parseExpression() error: invalid right bracket ident (%s) in expression (%s)\n", op.src, str())
		case dotIdent:
			e.operator = &dotOperator
		case lessThanIdent:
			e.operator = &lessThanOperator
		case lessThanEqualIdent:
			e.operator = &lessThanEqualOperator
		case greatThanIdent:
			e.operator = &greatThanOperator
		case greatThanEqualIdent:
			e.operator = &greatThanEqualOperator
		case equalIdent:
			e.operator = &equalOperator
		case notEqualIdent:
			e.operator = &notEqualOperator
		case plugIdent:
			e.operator = &plugOperator
		case subIdent:
			e.operator = &subOperator
		case asteriskIdent:
			e.operator = &mulOperator
		case divIdent:
			e.operator = &divOperator
		case andIdent:
			e.operator = &andOperator
		case orIdent:
			e.operator = &orOperator
		default:
			return nil, fmt.Errorf("nbfmt.parseExpression() error: invalid operator ident (%s) in expression (%s)\n", op.src, str())
		}
	}
	nextExpr, err := parseExpression(identList, isGroup, isIndex)
	if err != nil {
		return nil, err
	}
	if nextExpr == nil {
		return e, nil
	}
	e.nextExpr = nextExpr
	nextExpr.prevExpr = e
	return e, nil
}

func genIfCaseBlock(ss *stmtStack) (*ifcaseBlock, error) {
	icb := &ifcaseBlock{}
	s := ss.pop()
	ctx := "start"
	switch s.typ {
	case ifstmt, elseifstmt:
		if len(s.idents) < 2 {
			return nil, fmt.Errorf("nbfmt.genIfCaseBlock() parse error: invalid if case statement (%s)\n", s)
		}
		exprIdentList := s.idents[1:]
		expr, err := parseExpression(&exprIdentList, false, false)
		if err != nil {
			return nil, err
		}
		icb.exp = expr
		icb.stmt = s
		icb.appendSrc(s.src)
	default:
		return nil, fmt.Errorf("nbfmt.genIfCaseBlock() parse error: invalid if case statement (%s)\n", s)
	}
OUTER:
	for ss.len() > 0 {
		st := ss.checkType()
		switch st {
		case elseifstmt, elsestmt, endifstmt:
			ctx = "finish"
			break OUTER
		case ifstmt, forstmt, switchstmt, templatestmt, valuestmt:
			subBlock, err := genBlock(ss)
			if err != nil {
				return nil, err
			}
			icb.appendSubBlock(subBlock)
			icb.appendSrc(subBlock.getSrc())
		default:
			return nil, fmt.Errorf("nbfmt.genIfCaseBlock() parser error: invalid subblock statement (%s)", s)
		}
	}
	if ctx != "finish" {
		return nil, errors.New("nbfmt.genIfCaseBlock() parser error: not finished block")
	}
	if len(icb.subBlocks) == 0 {
		return nil, errors.New("nbfmt.genIfCaseBlock() parser error: empty if case block")
	}
	return icb, nil
}

func genDefaultBlock(ss *stmtStack) (*defaultBlock, error) {
	db := &defaultBlock{}
	defaultType := ss.checkType()
	s := ss.pop()
	db.appendSrc(s.src)
	ctx := "start"
OUTER:
	for ss.len() > 0 {
		switch ss.checkType() {
		case ifstmt, forstmt, switchstmt, templatestmt, valuestmt:
			subBlock, err := genBlock(ss)
			if err != nil {
				return nil, err
			}
			db.subBlocks = append(db.subBlocks, subBlock)
			db.appendSrc(subBlock.getSrc())
		case endifstmt:
			if defaultType == elsestmt {
				ctx = "finish"
				break OUTER
			} else {
				return nil, errors.New("nbfmt.genDefaultBlock() parse error: invalid endif statement")
			}
		case endswitchstmt:
			if defaultType == defaultstmt {
				ctx = "finish"
				break OUTER
			} else {
				return nil, errors.New("nbfmt.genDefaultBlock() parse error: invalid endswitch statement")
			}
		default:
			return nil, errors.New("nbfmt.genDefaultBlock() parse error: invalid statement")
		}
	}
	if ctx != "finish" {
		return nil, errors.New("nbfmt.genDefaultBlock() parse error: not finished default block")
	}
	if len(db.subBlocks) == 0 {
		return nil, errors.New("nbfmt.genDefaultBlock() parse error: empty default block")
	}
	return db, nil

}

func genIfBlock(ss *stmtStack) (*ifBlock, error) {
	ib := &ifBlock{}
	ctx := "start"
OUTER:
	for ss.len() > 0 {
		st := ss.checkType()
		switch st {
		case ifstmt, elseifstmt:
			switch ctx {
			case "start":
				caseBlock, err := genIfCaseBlock(ss)
				if err != nil {
					return nil, err
				}
				ib.caseBlocks = append(ib.caseBlocks, caseBlock)
				ib.appendSrc(caseBlock.getSrc())
			default:
				return nil, errors.New("nbfmt.genIfBlock() error: invalid if (else if) statement position")
			}
		case elsestmt:
			switch ctx {
			case "start":
				ctx = "else"
				defBlock, err := genDefaultBlock(ss)
				if err != nil {
					return nil, err
				}
				ib.defaultBlock = defBlock
				ib.appendSrc(defBlock.getSrc())
			default:
				return nil, errors.New("nbfmt.genIfBlock() error: invalid else statement position")
			}
		case endifstmt:
			s := ss.pop()
			ib.appendSrc(s.src)
			ctx = "finish"
			break OUTER
		}
	}
	if ctx != "finish" {
		return nil, errors.New("nbfmt.genIfBlock() error: not finished if block")
	}
	if len(ib.caseBlocks) == 0 {
		return nil, errors.New("nbfmt.genIfBlock() error: empty if block")
	}
	return ib, nil
}

func genSwitchCaseBlock(ss *stmtStack) (*switchcaseBlock, error) {
	scb := &switchcaseBlock{}
	s := ss.pop()
	if len(s.idents) < 2 {
		return nil, fmt.Errorf("nbfmt.genSwitchCaseBlock() parse error: invalid switch case statement (%s)\n", s)
	}
	exprIdentList := make([]*ident, 0, 8)
	exprList := make([]*expression, 0, 8)
	for _, id := range s.idents[1:] {
		switch id.typ {
		case commaIdent:
			if len(exprIdentList) == 0 {
				return nil, fmt.Errorf("nbfmt.genSwitchCaseBlock() parse error: invalid switch case statement (%s)\n", s)
			}
			expr, err := parseExpression(&exprIdentList, false, false)
			if err != nil {
				return nil, err
			}
			exprList = append(exprList, expr)
			exprIdentList = exprIdentList[:0]
		default:
			exprIdentList = append(exprIdentList, id)
		}
	}
	if len(exprIdentList) > 0 {
		expr, err := parseExpression(&exprIdentList, false, false)
		if err != nil {
			return nil, err
		}
		exprList = append(exprList, expr)
	}
	scb.exps = exprList
	scb.stmt = s
	scb.appendSrc(s.src)
	ctx := "start"
OUTER:
	for ss.len() > 0 {
		switch ss.checkType() {
		case ifstmt, forstmt, switchstmt, templatestmt, valuestmt:
			subBlock, err := genBlock(ss)
			if err != nil {
				return nil, err
			}
			scb.appendSubBlock(subBlock)
			scb.appendSrc(subBlock.getSrc())
		case casestmt, defaultstmt, endswitchstmt:
			ctx = "finish"
			break OUTER
		default:
			return nil, fmt.Errorf("nbfmt.genSwitchCaseBlock() parse error: invalid statement (%s)\n", ss.pop().src)
		}
	}
	if ctx != "finish" {
		return nil, errors.New("nbfmt.genSwitchCaseBlock() parse error: not finished switchcase block\n")
	}
	if len(scb.subBlocks) == 0 {
		return nil, errors.New("nbfmt.genSwitchCaseBlock() parse error: empty switchcase block")
	}
	return scb, nil
}

func genSwitchBlock(ss *stmtStack) (*switchBlock, error) {
	sb := &switchBlock{}
	s := ss.pop()
	if len(s.idents) < 2 {
		return nil, fmt.Errorf("nbfmt.genSwitchBlock() parse error: invalid switch statement (%s)\n", s)
	}
	exprIdentList := s.idents[1:]
	expr, err := parseExpression(&exprIdentList, false, false)
	if err != nil {
		return nil, err
	}
	sb.exp = expr
	sb.stmt = s
	sb.appendSrc(s.src)
	ctx := "start"
OUTER:
	for ss.len() > 0 {
		switch ss.checkType() {
		case casestmt:
			switch ctx {
			case "start":
				caseBlock, err := genSwitchCaseBlock(ss)
				if err != nil {
					return nil, err
				}
				sb.appendSrc(caseBlock.getSrc())
				sb.caseBlocks = append(sb.caseBlocks, caseBlock)
			default:
				return nil, fmt.Errorf("nbfmt.genSwitchBlock() parse error: wrong case statement position (%s)\n", ss.pop().src)
			}
		case defaultstmt:
			switch ctx {
			case "start":
				ctx = "default"
				defBlock, err := genDefaultBlock(ss)
				if err != nil {
					return nil, err
				}
				sb.appendSrc(defBlock.getSrc())
				sb.defaultBlock = defBlock
			default:
				return nil, fmt.Errorf("nbfmt.genSwitchBlock() parse error: wrong default statement position (%s)\n", ss.pop().src)
			}
		case endswitchstmt:
			ctx = "finish"
			s := ss.pop()
			sb.appendSrc(s.src)
			break OUTER
		case templatestmt:
			ss.pop()
			continue
		default:
			return nil, fmt.Errorf("nbfmt.genSwitchBlock() parse error: invalid statement (%s) in switch block\n", ss.pop().src)
		}
	}
	if ctx != "finish" {
		return nil, errors.New("nbfmt.genSwitchBlock() parse error: not finished switch block\n")
	}
	if len(sb.caseBlocks) == 0 {
		return nil, errors.New("nbfmt.genSwitchBlock() parse error: empty switch block\n")
	}
	return sb, nil
}

func genForBlock(ss *stmtStack) (*forBlock, error) {
	fb := &forBlock{}
	s := ss.pop()
	if len(s.idents) < 6 {
		return nil, fmt.Errorf("nbfmt.genForBlock() parse error: invalid for statement (%s)\n", s.src)
	}
	indexIdent := s.idents[1]
	variableIdent := s.idents[3]
	objExprIdent := s.idents[5:]
	if indexIdent.typ != varIdent || variableIdent.typ != varIdent {
		return nil, fmt.Errorf("nbfmt.genForBlock() parse error: invalid for statement (%s)\n", s.src)
	}
	fb.appendSrc(s.src)
	fb.stmt = s
	fb.indexVarName = indexIdent.src
	fb.valueVarName = variableIdent.src
	objExpr, err := parseExpression(&objExprIdent, false, false)
	if err != nil {
		return nil, err
	}
	fb.objExpr = objExpr
	ctx := "start"
OUTER:
	for ss.len() > 0 {
		switch ss.checkType() {
		case ifstmt, forstmt, switchstmt, templatestmt, valuestmt:
			subBlock, err := genBlock(ss)
			if err != nil {
				return nil, err
			}
			fb.appendSubBlock(subBlock)
			fb.appendSrc(subBlock.getSrc())
		case endforstmt:
			s := ss.pop()
			fb.appendSrc(s.src)
			ctx = "finish"
			break OUTER
		default:
			return nil, fmt.Errorf("nbfmt.genForBlock() parse error: invalid statement (%s)\n", ss.pop().src)
		}
	}
	if ctx != "finish" {
		return nil, errors.New("nbfmt.genFroBlock() parse error: not finished for block\n")
	}
	if len(fb.subBlocks) == 0 {
		return nil, errors.New("nbfmt.genForBlock() parse error: empty for block\n")
	}
	return fb, nil
}

func genTemplateBlock(ss *stmtStack) (*tempBlock, error) {
	s := ss.pop()
	tb := &tempBlock{s.src}
	return tb, nil
}

func genValueBlock(ss *stmtStack) (*valueBlock, error) {
	vb := &valueBlock{}
	s := ss.pop()
	expr, err := parseExpression(&s.idents, false, false)
	if err != nil {
		return nil, err
	}
	vb.exp = expr
	vb.stmt = s
	vb.appendSrc(s.src)
	return vb, nil
}

func genBlock(ss *stmtStack) (block, error) {
	switch ss.checkType() {
	case ifstmt:
		return genIfBlock(ss)
	case forstmt:
		return genForBlock(ss)
	case switchstmt:
		return genSwitchBlock(ss)
	case templatestmt:
		return genTemplateBlock(ss)
	case valuestmt:
		return genValueBlock(ss)
	default:
		return nil, fmt.Errorf("nbfmt.genBlock() error: invalid statement (%s)\n", ss.pop().src)
	}
}

func genTemplate(sl []*stmt) (template, error) {
	ss := newStmtStack(&sl)
	t := template{}
	for ss.len() > 0 {
		b, err := genBlock(ss)
		if err != nil {
			return t, err
		}
		t.blocks = append(t.blocks, b)
	}
	return t, nil
}
