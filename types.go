package nbfmt

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type identType int

const (
	unresolved identType = iota
	ifIdent
	elseifIdent
	elseIdent
	endifIdent
	forIdent
	inIdent
	endforIdent
	switchIdent
	caseIdent
	endswitchIdent
	objIdent
	leftParenthesisIdent
	righParenthesisIdent
	leftBracketIdent
	rightBracketIdent
	leftBraceIdent
	rightBraceIdent
	dotIdent
	numIdent
	strIdent
	boolIdent
	eqIdent
	neqIdent
	ltIdent
	lteIdent
	gtIdent
	gteIdent
	andIdent
	orIdent
)

// type exprType int

// const (
// 	tempExpr exprType = iota
// 	blockExpr
// )

type ident struct {
	name string
	typ  identType
}

type block interface {
	getSrc() string
	appendSrc(string)
	run(map[string]interface{}) (string, error)
	appendExpr(object)
	appendSubBlock(block)
	init() error
}

type template struct {
	blocks []block
}

func (t template) run(env map[string]interface{}) (string, error) {
	builder := strings.Builder{}
	for _, b := range t.blocks {
		s, err := b.run(env)
		if err != nil {
			return "", err
		}
		builder.WriteString(s)
	}
	return builder.String(), nil
}

type objType int

const (
	invalidObj objType = iota
	variable
	numconst
	strconst
	boolconst
	operator
	keyword
)

type object struct {
	typ    objType
	idents []ident
}

type objctx int

const (
	normal objctx = iota
	field
	index
	intindex
	fltindex
	strindex
)

func (o object) getVal(env map[string]interface{}) (interface{}, error) {
	var val reflect.Value
	if o.typ == variable {
		if o.idents[0].typ != objIdent {
			return nil, fmt.Errorf("invalid object: %s", o.idents[0].name)
		}
		v, ok := env[o.idents[0].name]
		if !ok {
			return nil, fmt.Errorf("object %s not exists in env variable", o.idents[0].name)
		}
		val = reflect.ValueOf(v)
		for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
			val = val.Elem()
			if !val.IsValid() {
				return nil, errors.New("invalid pointer or interface")
			}
		}
		var ctx objctx
		var indexStr string
		for _, id := range o.idents[1:] {
			switch ctx {
			case normal:
				switch id.typ {
				case dotIdent:
					ctx = field
				case leftBracketIdent:
					ctx = index
				default:
					return nil, fmt.Errorf("invalid select syntax: %s", id.name)
				}
			case field:
				switch id.typ {
				case objIdent:
					if val.Kind() != reflect.Struct {
						return nil, fmt.Errorf("object is not a struct")
					}
					f := val.FieldByName(id.name)
					for f.Kind() == reflect.Ptr || f.Kind() == reflect.Interface {
						f = f.Elem()
						if !f.IsValid() {
							return nil, fmt.Errorf("not valid pointer or interface")
						}
					}
					val = f
					ctx = normal
				default:
					return nil, fmt.Errorf("invalid select syntax: %s", id.name)

				}
			case index:
				switch id.typ {
				case numIdent:
					ctx = intindex
					indexStr += id.name
				case strIdent:
					ctx = strindex
					indexStr += id.name
				default:
					return nil, fmt.Errorf("invalid index syntax: %s", id.name)
				}
			case intindex:
				switch id.typ {
				case numIdent:
					indexStr += id.name
				case dotIdent:
					ctx = fltindex
					indexStr += id.name
				case rightBracketIdent:
					idx, err := strconv.ParseInt(indexStr, 10, 64)
					if err != nil {
						return nil, err
					}
					if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
						return nil, fmt.Errorf("object is not a slice or array")
					}
					v := val.Index(int(idx))
					for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
						v = v.Elem()
						if !v.IsValid() {
							return nil, fmt.Errorf("invalid pointer or interface")
						}
					}
					val = v
					ctx = normal
					indexStr = ""
				default:
					return nil, fmt.Errorf("invalid int index syntax: %s", id.name)
				}
			case fltindex:
				switch id.typ {
				case numIdent:
					indexStr += id.name
				case rightBracketIdent:
					idx, err := strconv.ParseFloat(indexStr, 64)
					if err != nil {
						return nil, err
					}
					if val.Kind() != reflect.Map {
						return nil, fmt.Errorf("object is not a map")
					}
					switch val.Type().Key().Kind() {
					case reflect.Float32:
						val = val.MapIndex(reflect.ValueOf(float32(idx)))
					case reflect.Float64:
						val = val.MapIndex(reflect.ValueOf(idx))
					default:
						return nil, errors.New("the type of map key is not float32 or float64")
					}
					if !val.IsValid() {
						return nil, errors.New("invalid map value")
					}
					for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
						val = val.Elem()
						if !val.IsValid() {
							return nil, errors.New("invalid pointer or interface")
						}
					}
					ctx = normal
					indexStr = ""
				default:
					return nil, errors.New("invalid float index syntax")
				}
			case strindex:
				switch id.typ {
				case strIdent:
					indexStr += id.name
				case rightBracketIdent:
					if val.Type().Key().Kind() != reflect.String {
						return nil, errors.New("the type of map key is not string")
					}
					val = val.MapIndex(reflect.ValueOf(indexStr))
					if !val.IsValid() {
						return nil, errors.New("invalid map value")
					}
					for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
						val = val.Elem()
						if !val.IsValid() {
							return nil, errors.New("invalid pointer or interface in map")
						}
					}
					ctx = normal
					indexStr = ""
				default:
					return nil, errors.New("invalid string index syntax")
				}
			}
		}
	} else {
		switch o.typ {
		case numconst:
			var isFloat bool
			builder := strings.Builder{}
			for _, id := range o.idents {
				builder.WriteString(id.name)
				if id.typ == dotIdent {
					isFloat = true
				}
			}
			if isFloat {
				f, err := strconv.ParseFloat(builder.String(), 64)
				if err != nil {
					return nil, err
				}
				return f, nil
			}
			return strconv.ParseInt(builder.String(), 10, 64)
		case strconst:
			builder := strings.Builder{}
			for _, id := range o.idents {
				builder.WriteString(id.name)
			}
			return builder.String(), nil
		case boolconst:
			return strconv.ParseBool(o.idents[0].name)
		}
	}
	return val.Interface(), nil
}

type codeType int

const (
	str codeType = iota
	blk
)

type code struct {
	src     string
	typ     codeType
	idents  []ident
	objects []object
}

type tempBlock struct {
	src string
}

func (b *tempBlock) getSrc() string {
	return b.src
}

func (b *tempBlock) appendSrc(s string) {
	b.src += s
}

func (b *tempBlock) run(env map[string]interface{}) (string, error) {
	return b.src, nil
}

func (b *tempBlock) appendExpr(o object)      {}
func (b *tempBlock) appendSubBlock(blk block) {}

func (b *tempBlock) init() error {
	return nil
}

type ifcaseBlock struct {
	src       string
	expr      []object
	subBlocks []block
	exprObj   *expression
}

func (b *ifcaseBlock) getSrc() string {
	return b.src
}

func (b *ifcaseBlock) appendSrc(s string) {
	b.src += s
}

func (b *ifcaseBlock) appendExpr(o object) {
	b.expr = append(b.expr, o)
}

func (b *ifcaseBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *ifcaseBlock) init() error {
	if b.expr[0].idents[0].typ != elseIdent {
		exprs := make([]object, len(b.expr)-1)
		copy(exprs, b.expr[1:])
		exprObj, err := parseExpr(&exprs, false)
		if err != nil {
			return err
		}
		b.exprObj = exprObj
	}
	for _, sb := range b.subBlocks {
		err := sb.init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *ifcaseBlock) run(env map[string]interface{}) (string, error) {
	if b.exprObj == nil {
		builder := strings.Builder{}
		for _, sb := range b.subBlocks {
			s, err := sb.run(env)
			if err != nil {
				return "", err
			}
			builder.WriteString(s)
		}
		return builder.String(), nil
	}
	isMatch, err := b.exprObj.eval(env)
	if err != nil {
		return "", err
	}
	if isMatch {
		builder := strings.Builder{}
		for _, sb := range b.subBlocks {
			s, err := sb.run(env)
			if err != nil {
				return "", err
			}
			builder.WriteString(s)
		}
		return builder.String(), nil
	} else {
		return "", nil
	}
}

type ifBlock struct {
	src       string
	subBlocks []block
}

func (b *ifBlock) getSrc() string {
	return b.src
}

func (b *ifBlock) appendSrc(s string) {
	b.src += s
}

func (b *ifBlock) appendExpr(o object) {}
func (b *ifBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *ifBlock) init() error {
	for _, sb := range b.subBlocks {
		err := sb.init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *ifBlock) run(env map[string]interface{}) (string, error) {
	for _, sb := range b.subBlocks {
		s, err := sb.run(env)
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
	}
	return "", nil
}

type forBlock struct {
	src          string
	expr         []object
	subBlocks    []block
	interVarName string
	iterObj      object
}

func (b *forBlock) getSrc() string {
	return b.src
}

func (b *forBlock) appendSrc(s string) {
	b.src += s
}

func (b *forBlock) appendExpr(o object) {
	b.expr = append(b.expr, o)
}

func (b *forBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *forBlock) init() error {
	if len(b.expr) != 4 {
		return fmt.Errorf("invalid for block expression: %v", b.expr)
	}
	if b.expr[1].typ != variable {
		return fmt.Errorf("invalid inter variable: %v", b.expr[1])
	}
	if len(b.expr[1].idents) != 1 {
		return fmt.Errorf("invalid inter variable idents length: %v", b.expr[1])
	}
	b.interVarName = b.expr[1].idents[0].name
	if b.expr[3].typ != variable {
		return fmt.Errorf("invalid iter target: %v", b.expr[3])
	}
	b.iterObj = b.expr[3]
	for _, sb := range b.subBlocks {
		err := sb.init()
		if err != nil {
			return err
		}

	}
	return nil
}

func (b *forBlock) run(env map[string]interface{}) (string, error) {
	o, err := b.iterObj.getVal(env)
	if err != nil {
		return "", err
	}
	kind := reflect.TypeOf(o).Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return "", fmt.Errorf("the object for iterating is not array or slice: %v", o)
	}
	val := reflect.ValueOf(o)
	builder := strings.Builder{}
	for i := 0; i < val.Len(); i++ {
		elemVal := val.Index(i)
		for elemVal.Kind() == reflect.Ptr || elemVal.Kind() == reflect.Interface {
			elemVal = elemVal.Elem()
			if !elemVal.IsValid() {
				return "", fmt.Errorf("invalid slice(array) element: index %d", i)
			}
		}
		if !elemVal.IsValid() {
			return "", fmt.Errorf("invalid slice(array) element: index %d", i)
		}
		env[b.interVarName] = elemVal.Interface()
		for _, sb := range b.subBlocks {
			s, err := sb.run(env)
			if err != nil {
				return "", err
			}
			builder.WriteString(s)
		}
	}
	return builder.String(), nil
}

type switchcaseBlock struct {
	src       string
	expr      []object
	subBlocks []block
	caseVal   object
}

func (b *switchcaseBlock) getSrc() string {
	return b.src
}

func (b *switchcaseBlock) appendSrc(s string) {
	b.src += s
}

func (b *switchcaseBlock) appendExpr(o object) {
	b.expr = append(b.expr, o)
}

func (b *switchcaseBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *switchcaseBlock) init() error {
	if len(b.expr) != 2 {
		return fmt.Errorf("invalid switch case: %v", b.expr)
	}
	b.caseVal = b.expr[1]
	for _, sb := range b.subBlocks {
		err := sb.init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *switchcaseBlock) run(env map[string]interface{}) (string, error) {
	builder := strings.Builder{}
	cv, err := b.caseVal.getVal(env)
	if err != nil {
		return "", err
	}
	switch b.caseVal.typ {
	case strconst:
		if sv, ok := env["_targetVal"].(string); ok {
			if cv.(string) == sv {
				for _, sb := range b.subBlocks {
					s, err := sb.run(env)
					if err != nil {
						return "", err
					}
					builder.WriteString(s)
				}
				return builder.String(), nil
			}
		} else {
			return "", fmt.Errorf("switch target value is not string: %v", env["targetVal"])
		}
	case numconst:
		var isFloat bool
		for _, id := range b.caseVal.idents {
			if id.typ == dotIdent {
				isFloat = true
			}
		}
		if isFloat {
			if fv, ok := env["_targetVal"].(float64); ok {
				if cv.(float64) == fv {
					for _, sb := range b.subBlocks {
						s, err := sb.run(env)
						if err != nil {
							return "", err
						}
						builder.WriteString(s)
					}
					return builder.String(), nil
				}
			} else {
				return "", fmt.Errorf("switch target value is not float64: %v", env["targetVal"])
			}
		} else {
			if iv, ok := env["_targetVal"].(int64); ok {
				if cv.(int64) == iv {
					for _, sb := range b.subBlocks {
						s, err := sb.run(env)
						if err != nil {
							return "", err
						}
						builder.WriteString(s)
					}
					return builder.String(), nil
				}
			} else {
				return "", fmt.Errorf("switch target value is not int64: %v", env["targetVal"])
			}
		}
	case boolconst:
		if bv, ok := env["_targetVal"].(bool); ok {
			if cv.(bool) == bv {
				for _, sb := range b.subBlocks {
					s, err := sb.run(env)
					if err != nil {
						return "", err
					}
					builder.WriteString(s)
				}
				return builder.String(), nil
			}
		} else {
			return "", fmt.Errorf("switch target value is not bool: %v", env["targetVal"])
		}
	default:
		return "", fmt.Errorf("case value is not comparable: %v", b.caseVal)
	}
	return "", nil
}

type switchBlock struct {
	src       string
	expr      []object
	subBlocks []block
	targetObj object
}

func (b *switchBlock) getSrc() string {
	return b.src
}

func (b *switchBlock) appendSrc(s string) {
	b.src += s
}

func (b *switchBlock) appendExpr(o object) {
	b.expr = append(b.expr, o)
}

func (b *switchBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *switchBlock) init() error {
	if len(b.expr) != 2 || b.expr[1].typ != variable {
		return fmt.Errorf("invalid switch: %v", b.expr)
	}
	b.targetObj = b.expr[1]
	for _, sb := range b.subBlocks {
		err := sb.init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *switchBlock) run(env map[string]interface{}) (string, error) {
	tv, err := b.targetObj.getVal(env)
	if err != nil {
		return "", err
	}
	env["_targetVal"] = tv
	for _, sb := range b.subBlocks {
		s, err := sb.run(env)
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
	}
	return "", nil
}

type valueBlock struct {
	src  string
	expr []object
}

func (b *valueBlock) getSrc() string {
	return b.src
}

func (b *valueBlock) appendSrc(s string) {
	b.src += s
}

func (b *valueBlock) appendExpr(o object) {
	b.expr = append(b.expr, o)
}

func (b *valueBlock) appendSubBlock(blk block) {}

func (b *valueBlock) init() error {
	if len(b.expr) != 1 {
		return fmt.Errorf("invalid value block: %v", b.expr)
	}
	return nil
}

func (b *valueBlock) run(env map[string]interface{}) (string, error) {
	v, err := b.expr[0].getVal(env)
	if err != nil {
		return "", err
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	default:
		return "", fmt.Errorf("invalid value type: %v", v)
	}
}

type exprType int

type exprGroup struct {
	objects   []object
	subGroups []exprGroup
}

func (g *exprGroup) split() {
	if len(g.subGroups) > 0 {
		for i := range g.subGroups {
			g.subGroups[i].split()
		}
	} else {
		buf := make([]object, 0, 8)
		for _, o := range g.objects {
			if o.idents[0].typ == andIdent || o.idents[0].typ == orIdent {
				if len(buf) > 0 {
					objs := make([]object, len(buf))
					copy(objs, buf)
					buf = buf[:0]
					g.subGroups = append(g.subGroups, exprGroup{objects: objs, subGroups: make([]exprGroup, 0, 8)})
				}
				g.subGroups = append(g.subGroups, exprGroup{objects: []object{o}, subGroups: make([]exprGroup, 0, 8)})
			} else {
				buf = append(buf, o)
			}
		}
		if len(buf) > 0 {
			objs := make([]object, len(buf))
			copy(objs, buf)
			buf = buf[:0]
			g.subGroups = append(g.subGroups, exprGroup{objects: objs, subGroups: make([]exprGroup, 0, 8)})
		}
		g.objects = g.objects[:0]
	}
}

type expression struct {
	leftVal  object
	op       object
	rightVal object
	relOp    object
	nextExpr *expression
	subExpr  *expression
}

func (e *expression) compare(env map[string]interface{}) (bool, error) {
	var flag int
	if e.leftVal.typ != invalidObj {
		flag |= 1 << 2
	}
	if e.op.typ != invalidObj {
		flag |= 1 << 1
	}
	if e.rightVal.typ != invalidObj {
		flag |= 1
	}
	switch flag {
	case 4:
		v, err := e.leftVal.getVal(env)
		if err != nil {
			return false, err
		}
		bv, ok := v.(bool)
		if !ok {
			return false, fmt.Errorf("invalid bool value in expression: %v", e.leftVal)
		}
		return bv, nil
	case 7:
		lv, err := e.leftVal.getVal(env)
		if err != nil {
			return false, err
		}
		rv, err := e.rightVal.getVal(env)
		if err != nil {
			return false, err
		}
		lk, rk := reflect.TypeOf(lv).Kind(), reflect.TypeOf(rv).Kind()
		if lk != rk {
			return false, fmt.Errorf("the types for comparing must be euqal: %v, %v", lv, rv)
		}
		switch lk {
		case reflect.String:
			return stringCompare(lv.(string), rv.(string), e.op)
		case reflect.Int64:
			return intCompare(lv.(int64), rv.(int64), e.op)
		case reflect.Float64:
			return floatCompare(lv.(float64), rv.(float64), e.op)
		default:
			return false, fmt.Errorf("the type is not supported for compring: %v", lk)
		}
	default:
		return false, fmt.Errorf("invalid expression: %v", e)
	}
}

func (e *expression) eval(env map[string]interface{}) (bool, error) {
	if e.subExpr != nil {
		isMatch, err := e.subExpr.eval(env)
		if err != nil {
			return false, err
		}
		if isMatch {
			if e.nextExpr == nil {
				return true, nil
			}
			if e.relOp.idents[0].typ == orIdent {
				return true, nil
			}
			return e.nextExpr.eval(env)
		} else {
			if e.nextExpr == nil {
				return false, nil
			}
			if e.relOp.idents[0].typ == orIdent {
				return e.nextExpr.eval(env)
			}
			return false, err
		}
	} else {
		// var isMatch bool
		// if e.op.typ == invalidObj {
		// 	lv, err := e.leftVal.getVal(env)
		// 	if err != nil {
		// 		return false, err
		// 	}
		// 	if bv, ok := lv.(bool); ok {
		// 		isMatch = bv
		// 	} else {
		// 		return false, fmt.Errorf("invalid bool value: %v", e.leftVal)
		// 	}
		// } else {
		// 	lv, err := e.leftVal.getVal(env)
		// 	if err != nil {
		// 		return false, err
		// 	}
		// 	rv, err := e.rightVal.getVal(env)
		// 	if err != nil {
		// 		return false, err
		// 	}
		// 	switch e.op.idents[0].typ {
		// 	case eqIdent:
		// 		lKind, rKind := reflect.TypeOf(lv).Kind(), reflect.TypeOf(rv).Kind()
		// 		if lKind != rKind {
		// 			return false, fmt.Errorf("the types of left value and right value is not equal: %v, %v", e.leftVal, e.rightVal)
		// 		}
		// 		isMatch = lv == rv
		// 	case neqIdent:
		// 		lKind, rKind := reflect.TypeOf(lv).Kind(), reflect.TypeOf(rv).Kind()
		// 		if lKind != rKind {
		// 			return false, fmt.Errorf("the types of left value and right value is not equal: %v, %v", e.leftVal, e.rightVal)
		// 		}
		// 		isMatch = lv != rv
		// 	case ltIdent, lteIdent, gtIdent, gteIdent:
		// 		lKind, rKind := reflect.TypeOf(lv).Kind(), reflect.TypeOf(rv).Kind()
		// 		if lKind != rKind {
		// 			return false, fmt.Errorf("the types of left value and right value is not equal: %v, %v", e.leftVal, e.rightVal)
		// 		}
		// 		switch lKind {
		// 		case reflect.String:
		// 			l, r := lv.(string), rv.(string)
		// 			switch e.op.idents[0].typ {
		// 			case ltIdent:
		// 				isMatch = l < r
		// 			case lteIdent:
		// 				isMatch = l <= r
		// 			case gtIdent:
		// 				isMatch = l > r
		// 			case gteIdent:
		// 				isMatch = l >= r
		// 			default:
		// 				return false, fmt.Errorf("invalid compare operator: %v", e.op)
		// 			}
		// 		case reflect.Int64:
		// 			l, r := lv.(int64), rv.(int64)
		// 			switch e.op.idents[0].typ {
		// 			case ltIdent:
		// 				isMatch = l < r
		// 			case lteIdent:
		// 				isMatch = l <= r
		// 			case gtIdent:
		// 				isMatch = l > r
		// 			case gteIdent:
		// 				isMatch = l >= r
		// 			default:
		// 				return false, fmt.Errorf("invalid compare operator: %v", e.op)
		// 			}
		// 		case reflect.Float64:
		// 			l, r := lv.(float64), rv.(float64)
		// 			switch e.op.idents[0].typ {
		// 			case ltIdent:
		// 				isMatch = l < r
		// 			case lteIdent:
		// 				isMatch = l <= r
		// 			case gtIdent:
		// 				isMatch = l > r
		// 			case gteIdent:
		// 				isMatch = l >= r
		// 			default:
		// 				return false, fmt.Errorf("invalid compare operator: %v", e.op)
		// 			}
		// 		default:
		// 			return false, fmt.Errorf("invalid types for less compare: %v, %v", e.leftVal, e.rightVal)
		// 		}
		// 	}
		// }
		isMatch, err := e.compare(env)
		if err != nil {
			return false, err
		}
		if isMatch {
			if e.nextExpr != nil {
				if e.op.idents[0].typ == andIdent {
					return e.nextExpr.eval(env)
				}
				return true, nil
			}
			return true, nil
		} else {
			if e.nextExpr != nil {
				if e.op.idents[0].typ == andIdent {
					return false, nil
				}
				return e.nextExpr.eval(env)
			}
			return false, nil
		}
	}
}
