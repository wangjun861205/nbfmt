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

func split(src string) ([]code, error) {
	l := make([]code, 0, 128)
	reader := bufio.NewReader(strings.NewReader(src))
	buf := strings.Builder{}
	var isOpen bool
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			if isOpen {
				return l, errors.New("not closed template")
			}
			if buf.Len() > 0 {
				l = append(l, code{src: buf.String(), typ: str})
			}
			return l, nil
		}
		if b == '{' {
			nb, err := reader.Peek(2)
			if err != nil {
				if err != io.EOF {
					return nil, err
				}
				if isOpen {
					return l, errors.New("not closed template")
				}
				buf.WriteString("{")
				buf.WriteString(string(nb))
				l = append(l, code{src: buf.String(), typ: str})
				return l, nil
			}
			if nb[0] == '{' {
				if nb[1] == '{' {
					buf.WriteByte(b)
					continue
				}
				if buf.Len() > 0 {
					l = append(l, code{src: buf.String(), typ: str})
				}
				buf.Reset()
				reader.Discard(1)
				buf.WriteString("{{")
				isOpen = true
				continue
			}
			buf.WriteByte('{')
			continue
		}
		if b == '}' {
			if !isOpen {
				buf.WriteByte(b)
				continue
			}
			nb, err := reader.Peek(1)
			if err != nil {
				return l, err
			}
			if nb[0] == '}' {
				reader.Discard(1)
				buf.WriteString("}}")
				isOpen = false
				l = append(l, code{src: buf.String(), typ: blk, idents: make([]ident, 0, 8)})
				buf.Reset()
				continue
			}

		}
		buf.WriteByte(b)
	}
	return l, nil
}

var numRe = regexp.MustCompile(`\d+`)

func splitBlock(c *code) error {
	reader := strings.NewReader(strings.Trim(c.src, "{} "))
	var isOpen bool
	builder := strings.Builder{}
	flushBuilder := func() {
		if builder.Len() > 0 {
			c.idents = append(c.idents, ident{name: builder.String()})
			builder.Reset()
		}
	}
	appendIdent := func(name string, typ identType) {
		c.idents = append(c.idents, ident{name: name, typ: typ})
	}
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				flushBuilder()
				break
			}
			return err
		}
		switch b {
		case '\'', '"', '`':
			isOpen = !isOpen
			if !isOpen {
				c.idents = append(c.idents, ident{name: builder.String(), typ: strIdent})
				builder.Reset()
			} else {
				flushBuilder()
			}
		case '(':
			flushBuilder()
			appendIdent("(", leftParenthesisIdent)
		case ')':
			flushBuilder()
			appendIdent(")", rightParenthesisIdent)
		case '[':
			flushBuilder()
			appendIdent("[", leftBracketIdent)
		case ']':
			flushBuilder()
			appendIdent("]", rightBracketIdent)
		case '{':
			flushBuilder()
			appendIdent("{", leftBraceIdent)
		case '}':
			flushBuilder()
			appendIdent("}", rightBraceIdent)
		case '.':
			if isOpen {
				builder.WriteByte(b)
			} else {
				flushBuilder()
				appendIdent(".", dotIdent)
			}
		case ',':
			flushBuilder()
			appendIdent(",", commaIdent)
		case ' ', '\t':
			if isOpen {
				builder.WriteByte(b)
			} else {
				flushBuilder()
			}
		default:
			builder.WriteByte(b)
		}
	}
	for i, id := range c.idents {
		if id.typ == unresolved {
			switch {
			case id.name == "if":
				c.idents[i].typ = ifIdent
			case id.name == "elseif":
				c.idents[i].typ = elseifIdent
			case id.name == "else":
				c.idents[i].typ = elseIdent
			case id.name == "endif":
				c.idents[i].typ = endifIdent
			case id.name == "for":
				c.idents[i].typ = forIdent
			case id.name == "in":
				c.idents[i].typ = inIdent
			case id.name == "endfor":
				c.idents[i].typ = endforIdent
			case id.name == "switch":
				c.idents[i].typ = switchIdent
			case id.name == "case":
				c.idents[i].typ = caseIdent
			case id.name == "default":
				c.idents[i].typ = defaultIdent
			case id.name == "endswitch":
				c.idents[i].typ = endswitchIdent
			case id.name == "==":
				c.idents[i].typ = eqIdent
			case id.name == "!=":
				c.idents[i].typ = neqIdent
			case id.name == "<":
				c.idents[i].typ = ltIdent
			case id.name == "<=":
				c.idents[i].typ = lteIdent
			case id.name == ">":
				c.idents[i].typ = gtIdent
			case id.name == ">=":
				c.idents[i].typ = gteIdent
			case id.name == "&&":
				c.idents[i].typ = andIdent
			case id.name == "||":
				c.idents[i].typ = orIdent
			case numRe.MatchString(id.name):
				c.idents[i].typ = numIdent
			case id.name == "true" || id.name == "false":
				c.idents[i].typ = boolIdent
			case id.name == ",":
				c.idents[i].typ = commaIdent
			default:
				c.idents[i].typ = objIdent
			}
		}
	}
	return nil
}

func parseObj(l *[]code) error {
	for i, c := range *l {
		if c.typ == blk {
			(*l)[i].objects = make([]object, 0, 8)
			idents := make([]ident, 0, 8)
			var typ objType
			for _, id := range c.idents {
				switch id.typ {
				case objIdent:
					switch typ {
					case variable:
						idents = append(idents, id)
					case invalidObj:
						typ = variable
						idents = append(idents, id)
					default:
						ids := make([]ident, len(idents))
						copy(ids, idents)
						(*l)[i].objects = append((*l)[i].objects, object{idents: ids, typ: typ})
						typ = variable
						idents = []ident{id}
					}
				case numIdent:
					switch typ {
					case numconst:
						idents = append(idents, id)
					case invalidObj:
						typ = numconst
						idents = []ident{id}
					case variable:
						if idents[len(idents)-1].typ != leftBracketIdent {
							return errors.New("invalid number syntax")
						}
						idents = append(idents, id)
					default:
						ids := make([]ident, len(idents))
						copy(ids, idents)
						(*l)[i].objects = append((*l)[i].objects, object{idents: ids, typ: typ})
						typ = numconst
						idents = []ident{id}
					}
				case dotIdent:
					switch typ {
					case numconst, variable:
						idents = append(idents, id)
					default:
						return errors.New("invalid dot syntax")
					}
				case boolIdent:
					switch typ {
					case invalidObj:
						(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: boolconst})
					default:
						return errors.New("invalid bool identifier")

					}
				case commaIdent:
					ids := make([]ident, len(idents))
					copy(ids, idents)
					(*l)[i].objects = append((*l)[i].objects, object{idents: ids, typ: typ})
					typ = invalidObj
					idents = idents[:0]
					(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: punct})

				case leftBracketIdent:
					switch typ {
					case variable:
						idents = append(idents, id)
					default:
						return errors.New("invalid left bracket syntax")
					}
				case rightBracketIdent:
					switch typ {
					case variable:
						idents = append(idents, id)
					default:
						return errors.New("invalid right bracket syntax")
					}
				case strIdent:
					switch typ {
					case variable:
						if idents[len(idents)-1].typ != leftBracketIdent {
							return fmt.Errorf("invalid string syntax: %v", id)
						}
						idents = append(idents, id)
					case invalidObj:
						(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: strconst})
					default:
						ids := make([]ident, len(idents))
						copy(ids, idents)
						(*l)[i].objects = append((*l)[i].objects, object{idents: ids, typ: typ})
						idents = idents[:0]
						typ = invalidObj
						(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: strconst})
					}
				case eqIdent, neqIdent, ltIdent, lteIdent, gtIdent, gteIdent, andIdent, orIdent, leftParenthesisIdent, rightParenthesisIdent:
					switch typ {
					case invalidObj:
						(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: operator})
					default:
						ids := make([]ident, len(idents))
						copy(ids, idents)
						(*l)[i].objects = append((*l)[i].objects, object{idents: ids, typ: typ})
						idents = idents[:0]
						typ = invalidObj
						(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: operator})
					}
				case ifIdent, elseifIdent, elseIdent, endifIdent, forIdent, inIdent, endforIdent, switchIdent, caseIdent, defaultIdent, endswitchIdent:
					switch typ {
					case invalidObj:
						(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: keyword})
					default:
						ids := make([]ident, len(idents))
						copy(ids, idents)
						(*l)[i].objects = append((*l)[i].objects, object{idents: ids, typ: typ})
						idents = idents[:0]
						typ = invalidObj
						(*l)[i].objects = append((*l)[i].objects, object{idents: []ident{id}, typ: keyword})
					}
				default:
					return errors.New("unknown ident type")
				}
			}
			if typ != invalidObj {
				ids := make([]ident, len(idents))
				copy(ids, idents)
				(*l)[i].objects = append((*l)[i].objects, object{idents: ids, typ: typ})
				idents = idents[:0]
				typ = invalidObj
			}
		}
	}
	return nil
}

func parseBlock(l *[]code) (block, error) {
	var c code
	c, *l = (*l)[0], (*l)[1:]
	switch c.typ {
	case str:
		return parseTempBlock(c), nil
	case blk:
		switch c.idents[0].typ {
		case ifIdent:
			return parseIfBlock(c, l)
		case forIdent:
			return parseForBlock(c, l)
		case switchIdent:
			return parseSwitchBlock(c, l)
		case objIdent:
			return parseValueBlock(c), nil
		default:
			return nil, fmt.Errorf("invalid block: %s", c.src)
		}
	default:
		return nil, errors.New("unknown code type")
	}
}

func parseIfBlock(c code, l *[]code) (*ifBlock, error) {
	b := &ifBlock{src: c.src}
	cb := &ifcaseBlock{src: c.src}
	for _, o := range c.objects {
		cb.appendExpr(o)
	}
	b.appendSubBlock(cb)
	for len(*l) > 0 {
		c, *l = (*l)[0], (*l)[1:]
		switch c.typ {
		case str:
			subBlock := parseTempBlock(c)
			cb.appendSubBlock(subBlock)
			cb.appendSrc(subBlock.getSrc())
			b.appendSrc(subBlock.getSrc())
		case blk:
			switch c.idents[0].typ {
			case elseifIdent, elseIdent:
				ncb, err := parseIfCaseBlock(c, l)
				if err != nil {
					return nil, err
				}
				b.appendSubBlock(ncb)
				b.appendSrc(ncb.getSrc())
			case endifIdent:
				b.appendSrc(c.src)
				return b, nil
			default:
				nl := make([]code, len(*l)+1)
				nl[0] = c
				copy(nl[1:], *l)
				*l = nl
				subBlock, err := parseBlock(l)
				if err != nil {
					return nil, err
				}
				cb.appendSubBlock(subBlock)
				cb.appendSrc(subBlock.getSrc())
				b.appendSrc(subBlock.getSrc())
			}
		}
	}
	return nil, errors.New("invalid if block")
}

func parseIfCaseBlock(c code, l *[]code) (*ifcaseBlock, error) {
	b := &ifcaseBlock{src: c.src}
	for _, o := range c.objects {
		b.appendExpr(o)
	}
	for len(*l) > 0 {
		c, *l = (*l)[0], (*l)[1:]
		switch c.typ {
		case str:
			b.appendSubBlock(&tempBlock{src: c.src})
			b.appendSrc(c.src)
		default:
			switch c.idents[0].typ {
			case elseifIdent, elseIdent, endifIdent:
				nl := make([]code, len(*l)+1)
				nl[0] = c
				copy(nl[1:], *l)
				*l = nl
				return b, nil
			default:
				nl := make([]code, len(*l)+1)
				nl[0] = c
				copy(nl[1:], *l)
				*l = nl
				subBlock, err := parseBlock(l)
				if err != nil {
					return nil, err
				}
				b.appendSubBlock(subBlock)
				b.appendSrc(subBlock.getSrc())
			}
		}
	}
	return nil, errors.New("invalid if case block")
}

func parseTempBlock(c code) *tempBlock {
	return &tempBlock{src: c.src}
}

func parseForBlock(c code, l *[]code) (*forBlock, error) {
	b := &forBlock{src: c.src}
	for _, o := range c.objects {
		b.appendExpr(o)
	}
	for len(*l) > 0 {
		c, *l = (*l)[0], (*l)[1:]
		switch c.typ {
		case str:
			b.appendSubBlock(&tempBlock{src: c.src})
			b.appendSrc(c.src)
		default:
			switch c.idents[0].typ {
			case endforIdent:
				b.appendSrc(c.src)
				return b, nil
			default:
				nl := make([]code, len(*l)+1)
				nl[0] = c
				copy(nl[1:], *l)
				*l = nl
				subBlock, err := parseBlock(l)
				if err != nil {
					return nil, err
				}
				b.appendSubBlock(subBlock)
				b.appendSrc(subBlock.getSrc())
			}
		}
	}
	return nil, errors.New("invalid for block")
}

func parseSwitchCaseBlock(c code, l *[]code) (*switchcaseBlock, error) {
	b := &switchcaseBlock{src: c.src}
	for _, o := range c.objects {
		b.appendExpr(o)
	}
	for len(*l) > 0 {
		c, *l = (*l)[0], (*l)[1:]
		switch c.typ {
		case str:
			b.appendSubBlock(&tempBlock{src: c.src})
			b.appendSrc(c.src)
		default:
			switch c.idents[0].typ {
			case caseIdent, defaultIdent, endswitchIdent:
				nl := make([]code, len(*l)+1)
				nl[0] = c
				copy(nl[1:], *l)
				*l = nl
				return b, nil
			default:
				nl := make([]code, len(*l)+1)
				nl[0] = c
				copy(nl[1:], *l)
				*l = nl
				subBlock, err := parseBlock(l)
				if err != nil {
					return nil, err
				}
				b.appendSubBlock(subBlock)
				b.appendSrc(subBlock.getSrc())
			}
		}
	}
	return nil, errors.New("invalid switch case block")
}

func parseSwitchBlock(c code, l *[]code) (*switchBlock, error) {
	b := &switchBlock{src: c.src}
	for _, o := range c.objects {
		b.appendExpr(o)
	}
	for len(*l) > 0 {
		c, *l = (*l)[0], (*l)[1:]
		switch c.typ {
		case str:
			continue
		default:
			switch c.idents[0].typ {
			case caseIdent, defaultIdent:
				cb, err := parseSwitchCaseBlock(c, l)
				if err != nil {
					return nil, err
				}
				b.appendSubBlock(cb)
				b.appendSrc(cb.getSrc())
			case endswitchIdent:
				b.appendSrc(c.src)
				return b, nil
			default:
				nl := make([]code, len(*l)+1)
				nl[0] = c
				copy(nl[1:], *l)
				*l = nl
				subBlock, err := parseBlock(l)
				if err != nil {
					return nil, err
				}
				b.appendSubBlock(subBlock)
				b.appendSrc(subBlock.getSrc())
			}
		}
	}
	return nil, errors.New("invalid switch block")
}

func parseValueBlock(c code) *valueBlock {
	b := &valueBlock{src: c.src}
	for _, o := range c.objects {
		b.appendExpr(o)
	}
	return b
}

func parseExpr(l *[]object, sub bool) (*expression, error) {
	if len(*l) == 0 {
		return nil, nil
	}
	expr := &expression{}
	var o object
	var err error
	ctx := "left"
	for len(*l) > 0 {
		o, *l = (*l)[0], (*l)[1:]
		switch o.typ {
		case operator:
			switch o.idents[0].typ {
			case andIdent, orIdent:
				ctx = "left"
				expr.relOp = o
				expr.nextExpr, err = parseExpr(l, false)
				if err != nil {
					return nil, err
				}
			case leftParenthesisIdent:
				expr.subExpr, err = parseExpr(l, true)
				if err != nil {
					return nil, err
				}
			case rightParenthesisIdent:
				if !sub {
					nl := make([]object, len(*l)+1)
					nl[0] = o
					copy(nl[1:], *l)
					*l = nl
					return expr, nil
				} else {
					return expr, nil
				}
			default:
				if ctx == "op" {
					ctx = "right"
					expr.op = o
				} else {
					return nil, fmt.Errorf("invalid operator: %v", o)
				}
			}
		default:
			switch ctx {
			case "left":
				expr.leftVal = o
				ctx = "op"
			case "right":
				expr.rightVal = o
				ctx = "final"
			default:
				return nil, fmt.Errorf("invalid expression value: %v", o)

			}
		}
	}
	return expr, nil
}

func parseTemplate(src string) (template, error) {
	temp := template{}
	cl, err := split(src)
	if err != nil {
		return temp, err
	}
	for i := range cl {
		if cl[i].typ == blk {
			err = splitBlock(&cl[i])
			if err != nil {
				return temp, err
			}
		}
	}
	err = parseObj(&cl)
	if err != nil {
		return temp, err
	}
	bl := make([]block, 0, 128)
	for len(cl) > 0 {
		b, err := parseBlock(&cl)
		if err != nil {
			return temp, err
		}
		bl = append(bl, b)
	}
	for _, b := range bl {
		err = b.init()
		if err != nil {
			return temp, err
		}
	}
	temp.blocks = bl
	return temp, nil
}

// ==================================================below is new edition==============================================================

type stmtType int

const (
	templatestmt stmtType = iota
	ifstmt
	elseifstmt
	elsestmt
	endifstmt
	forstmt
	endforstmt
	switchstmt
	casestmt
	defaultstmt
	endswitchstmt
	valuestmt
)

type stmt struct {
	src     string
	typ     stmtType
	idents  []ident
	objects []*object
}

func (s *stmt) String() string {
	return s.src
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
					err := parseStmtIdents(l)
					if err != nil {
						return nil, err
					}
					parseStmtType(l)
					err = parseStmtObjects(l)
					if err != nil {
						return nil, err
					}
					parseObjectsType(l)
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

var numIdentRe = regexp.MustCompile(`^-?\d+$`)
var varIdentRe = regexp.MustCompile(`^[a-zA-z_][\w_]*$`)
var chrIdentRe = regexp.MustCompile(`^'.*'$`)
var strIdentRe = regexp.MustCompile("^[\"|`].*[\"|`]$")
var boolIdentRe = regexp.MustCompile("^(true|false)$")

func parseIdent(s string) (ident, error) {
	switch s {
	case "if":
		return ident{name: s, typ: ifIdent}, nil
	case "elseif":
		return ident{name: s, typ: elseifIdent}, nil
	case "else":
		return ident{name: s, typ: elseIdent}, nil
	case "endif":
		return ident{name: s, typ: endifIdent}, nil
	case "for":
		return ident{name: s, typ: forIdent}, nil
	case "in":
		return ident{name: s, typ: inIdent}, nil
	case "endfor":
		return ident{name: s, typ: endforIdent}, nil
	case "switch":
		return ident{name: s, typ: switchIdent}, nil
	case "case":
		return ident{name: s, typ: caseIdent}, nil
	case "default":
		return ident{name: s, typ: defaultIdent}, nil
	case "endswitch":
		return ident{name: s, typ: endswitchIdent}, nil
	case ".":
		return ident{name: s, typ: dotIdent}, nil
	case ",":
		return ident{name: s, typ: commaIdent}, nil
	case "(":
		return ident{name: s, typ: leftParenthesisIdent}, nil
	case ")":
		return ident{name: s, typ: rightParenthesisIdent}, nil
	case "[":
		return ident{name: s, typ: leftBracketIdent}, nil
	case "]":
		return ident{name: s, typ: rightBracketIdent}, nil
	case "{":
		return ident{name: s, typ: leftBraceIdent}, nil
	case "}":
		return ident{name: s, typ: rightBracketIdent}, nil
	case "!":
		return ident{name: s, typ: exclamationIdent}, nil
	case "<":
		return ident{name: s, typ: ltIdent}, nil
	case ">":
		return ident{name: s, typ: gtIdent}, nil
	case "<=":
		return ident{name: s, typ: lteIdent}, nil
	case ">=":
		return ident{name: s, typ: gteIdent}, nil
	case "==":
		return ident{name: s, typ: eqIdent}, nil
	case "!=":
		return ident{name: s, typ: neqIdent}, nil
	case "&&":
		return ident{name: s, typ: andIdent}, nil
	case "||":
		return ident{name: s, typ: orIdent}, nil
	default:
		switch {
		case numIdentRe.MatchString(s):
			return ident{name: s, typ: numIdent}, nil
		case varIdentRe.MatchString(s):
			return ident{name: s, typ: varIdent}, nil
		case chrIdentRe.MatchString(s):
			return ident{name: s, typ: chrIdent}, nil
		case strIdentRe.MatchString(s):
			return ident{name: s, typ: strIdent}, nil
		case boolIdentRe.MatchString(s):
			return ident{name: s, typ: strIdent}, nil
		default:
			return ident{}, fmt.Errorf("nbfmt.parseIdent() parse error: invalid ident (%#v)", s)
		}
	}
}

func parseStmtIdents(l []*stmt) error {
OUTER:
	for _, s := range l {
		if s.src[:2] != "{{" {
			continue
		}
		reader := strings.NewReader(strings.Trim(s.src, "{} "))
		buf := strings.Builder{}
		ctx := "stmt"
		handleBuf := func() error {
			id, err := parseIdent(buf.String())
			if err != nil {
				return err
			}
			s.idents = append(s.idents, id)
			buf.Reset()
			return nil
		}
		for {
			char, err := reader.ReadByte()
			if err != nil {
				if err == io.EOF {
					switch ctx {
					case "stmt":
						continue OUTER
					default:
						err := handleBuf()
						if err != nil {
							return err
						}
						continue OUTER
					}
				} else {
					return err
				}
			}
			switch char {
			case '\'':
				switch ctx {
				case "stmt":
					ctx = "singleQuote"
					buf.WriteByte(char)
				case "singleQuote":
					ctx = "stmt"
					buf.WriteByte(char)
					err := handleBuf()
					if err != nil {
						return err
					}
				case "doubleQuote", "backQuote":
					buf.WriteByte(char)
				default:
					ctx = "singleQuote"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '"':
				switch ctx {
				case "stmt":
					ctx = "doubleQuote"
					buf.WriteByte(char)
				case "doubleQuote":
					ctx = "stmt"
					buf.WriteByte(char)
					err := handleBuf()
					if err != nil {
						return err
					}
				case "backQuote", "singleQuote":
					buf.WriteByte(char)
				default:
					ctx = "doubleQuote"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '`':
				switch ctx {
				case "stmt":
					ctx = "backQuote"
					buf.WriteByte(char)
				case "backQuote":
					ctx = "stmt"
					buf.WriteByte(char)
					err := handleBuf()
					if err != nil {
						return err
					}
				case "doubleQuote", "singleQuote":
					buf.WriteByte(char)
				default:
					ctx = "backQuote"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case ' ', '\t', '\n', '\r':
				switch ctx {
				case "stmt":
					continue
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				default:
					ctx = "stmt"
					err := handleBuf()
					if err != nil {
						return err
					}
				}
			case '<':
				switch ctx {
				case "stmt":
					ctx = "lt"
					buf.WriteByte(char)
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				default:
					ctx = "lt"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '>':
				switch ctx {
				case "stmt":
					ctx = "gt"
					buf.WriteByte(char)
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				default:
					ctx = "lt"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '!':
				switch ctx {
				case "stmt":
					ctx = "exclamation"
					buf.WriteByte(char)
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				default:
					ctx = "exclamation"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '=':
				switch ctx {
				case "stmt":
					ctx = "equal"
					buf.WriteByte(char)
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				case "exclamation", "equal", "lt", "gt":
					ctx = "stmt"
					buf.WriteByte(char)
					err := handleBuf()
					if err != nil {
						return err
					}
				default:
					ctx = "equal"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '&':
				switch ctx {
				case "stmt":
					ctx = "and"
					buf.WriteByte(char)
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				case "and":
					ctx = "stmt"
					buf.WriteByte(char)
					err := handleBuf()
					if err != nil {
						return err
					}
				default:
					ctx = "and"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '|':
				switch ctx {
				case "stmt":
					ctx = "or"
					buf.WriteByte(char)
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				case "or":
					ctx = "stmt"
					buf.WriteByte(char)
					err := handleBuf()
					if err != nil {
						return err
					}
				default:
					ctx = "or"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			case '(', ')', '[', ']', '{', '}', ',', '.':
				switch ctx {
				case "stmt":
					buf.WriteByte(char)
					err := handleBuf()
					if err != nil {
						return err
					}
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				default:
					ctx = "stmt"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
					err = handleBuf()
					if err != nil {
						return err
					}

				}
			default:
				switch ctx {
				case "stmt":
					ctx = "ident"
					buf.WriteByte(char)
				case "singleQuote", "doubleQuote", "backQuote":
					buf.WriteByte(char)
				case "ident":
					buf.WriteByte(char)
				default:
					ctx = "ident"
					err := handleBuf()
					if err != nil {
						return err
					}
					buf.WriteByte(char)
				}
			}
		}
	}
	return nil
}

func parseStmtObjects(l []*stmt) error {
OUTER:
	for _, s := range l {
		if s.typ != templatestmt {
			ctxList := []string{"clean"}
			idBuf := make([]ident, 0, 8)
			srcBuf := strings.Builder{}
			currentCtx := func() string {
				return ctxList[len(ctxList)-1]
			}
			popCtx := func() {
				ctxList = ctxList[:len(ctxList)-1]
			}
			pushCtx := func(ctx string) {
				ctxList = append(ctxList, ctx)
			}
			resetCtx := func() {
				ctxList = []string{"clean"}
			}
			resetIdBuf := func() {
				idBuf = idBuf[:0]
			}
			appendId := func(id ident) {
				idBuf = append(idBuf, id)
				srcBuf.WriteString(id.name)
			}
			copyIdBuf := func() []ident {
				l := make([]ident, len(idBuf))
				copy(l, idBuf)
				return l
			}
			refresh := func() {
				s.objects = append(s.objects, &object{src: srcBuf.String(), idents: copyIdBuf()})
				resetCtx()
				resetIdBuf()
				srcBuf.Reset()
			}
			for _, id := range s.idents {
				switch id.typ {
				case varIdent:
					switch currentCtx() {
					case "clean", "index":
						pushCtx("var")
						appendId(id)
					case "dot":
						popCtx()
						if currentCtx() != "var" {
							return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid variable ident (%s) in statement (%s)\n", id.name, s.src)
						}
						appendId(id)
					case "operator", "keyword", "comma":
						refresh()
						pushCtx("var")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid variable ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case strIdent:
					switch currentCtx() {
					case "clean", "index":
						pushCtx("str")
						appendId(id)
					case "operator", "keyword", "comma":
						refresh()
						pushCtx("str")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid string ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case numIdent:
					switch currentCtx() {
					case "clean", "index":
						pushCtx("int")
						appendId(id)
					case "dot":
						popCtx()
						if currentCtx() != "int" {
							return fmt.Errorf("nbfmt.parseStmtObjects() parse error: invalid string ident(%s) in statement (%s)\n", id.name, s.src)
						}
						popCtx()
						pushCtx("float")
						appendId(id)
					case "float":
						appendId(id)
					case "operator", "keyword", "comma":
						refresh()
						pushCtx("int")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid number ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case chrIdent:
					switch currentCtx() {
					case "clean", "index":
						pushCtx("chr")
						appendId(id)
					case "operator", "keyword", "comma":
						refresh()
						pushCtx("int")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid char ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case leftBracketIdent:
					switch currentCtx() {
					case "var":
						pushCtx("index")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid left bracket ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case rightBracketIdent:
					switch currentCtx() {
					case "var", "int", "float", "str", "chr", "bool":
						popCtx()
						if currentCtx() != "index" {
							return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid right bracket ident (%s) in statement (%s)\n", id.name, s.src)
						}
						popCtx()
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid right bracket ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case dotIdent:
					switch currentCtx() {
					case "var", "int":
						pushCtx("dot")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid dot ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case ifIdent, elseifIdent, elseIdent, endifIdent, forIdent, inIdent, endforIdent, switchIdent, caseIdent, defaultIdent, endswitchIdent:
					switch currentCtx() {
					case "clean":
						pushCtx("keyword")
						appendId(id)
					case "var", "int", "float", "str", "chr", "bool":
						popCtx()
						if currentCtx() != "clean" {
							return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid keyword ident (%s) in statement (%s)\n", id.name, s.src)
						}
						refresh()
						pushCtx("keyword")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid keyword ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case ltIdent, lteIdent, gtIdent, gteIdent, eqIdent, neqIdent, andIdent, orIdent:
					switch currentCtx() {
					case "var", "int", "float", "str", "chr", "bool":
						refresh()
						pushCtx("operator")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid operator ident (%s) in statement (%s)\n", id.name, s.src)

					}
				case exclamationIdent:
					switch currentCtx() {
					case "clean":
						pushCtx("operator")
						appendId(id)
					case "operator", "comma":
						refresh()
						pushCtx("opeartor")
						appendId(id)
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid exclamation ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case leftParenthesisIdent:
					switch currentCtx() {
					case "clean":
						appendId(id)
						refresh()
					case "keyword", "operator", "comma":
						refresh()
						appendId(id)
						refresh()
					default:
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid left parenthesis ident (%s) in statement (%s)\n", id.name, s.src)
					}
				case rightParenthesisIdent:
					switch currentCtx() {
					case "clean", "dot", "index", "keyword", "operator":
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid right parenthesis ident (%s) in statement (%s)\n", id.name, s.src)
					default:
						refresh()
						appendId(id)
						refresh()
					}
				case commaIdent:
					switch currentCtx() {
					case "clean", "dot", "index", "keyword", "operator", "comma":
						return fmt.Errorf("nbfmt.parseStmtObjects parse error: invalid comma ident (%s) in statement (%s)\n", id.name, s.src)
					default:
						refresh()
						pushCtx("comma")
						appendId(id)
					}
				default:
					return fmt.Errorf("nbfmt.parseStmtObjects() parse error: unsupported ident (%s) in statement (%s)\n", id.name, s.src)
				}
			}
			switch currentCtx() {
			case "clean":
				continue OUTER
			case "var", "int", "float", "str", "chr", "bool", "keyword":
				popCtx()
				if currentCtx() != "clean" {
					return fmt.Errorf("nbfmt.parseStmtObjects() parse error: incomplete ident (%s) in statement (%s)\n", srcBuf.String(), s.src)
				}
				refresh()
			default:
				return fmt.Errorf("nbfmt.parseStmtObjects() parse error: incomplete ident (%s) in statement (%s)\n", srcBuf.String(), s.src)
			}
		}
	}
	return nil
}

func parseObjectsType(l []*stmt) {
	for _, s := range l {
		switch s.typ {
		case templatestmt:
			continue
		default:
			for _, o := range s.objects {
				switch o.idents[0].typ {
				case ifIdent, elseifIdent, elseIdent, endifIdent, forIdent, inIdent, endforIdent, switchIdent, caseIdent, defaultIdent, endswitchIdent:
					o.typ = kwdobj
				case ltIdent, lteIdent, gtIdent, gteIdent, eqIdent, neqIdent, exclamationIdent, andIdent, orIdent:
					o.typ = oprobj
				case strIdent:
					o.typ = strconstobj
				case chrIdent:
					o.typ = chrconstobj
				case numIdent:
					var isFloat bool
					for _, id := range o.idents {
						if id.typ == dotIdent {
							isFloat = true
							break
						}
					}
					if isFloat {
						o.typ = fltconstobj
					} else {
						o.typ = intconstobj
					}
				case boolIdent:
					o.typ = bolconstobj
				case varIdent:
					o.typ = varobj
				case commaIdent:
					o.typ = pctobj
				case leftParenthesisIdent, rightParenthesisIdent:
					o.typ = prtobj
				}
			}
		}
	}
}
