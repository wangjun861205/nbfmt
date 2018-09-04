package nbfmt

import (
	"bufio"
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
			appendIdent(")", righParenthesisIdent)
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
				case eqIdent, neqIdent, ltIdent, lteIdent, gtIdent, gteIdent, andIdent, orIdent, leftParenthesisIdent, righParenthesisIdent:
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
				case ifIdent, elseifIdent, elseIdent, endifIdent, forIdent, inIdent, endforIdent, switchIdent, caseIdent, endswitchIdent:
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
			case caseIdent, endswitchIdent:
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
			case caseIdent:
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
			case righParenthesisIdent:
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
