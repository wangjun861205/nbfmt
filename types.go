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
	defaultIdent
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
	commaIdent
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
	punct
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
	if !val.IsValid() {
		return nil, fmt.Errorf("nbfmt getVal() error: value is not valid (object %v)", o)
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
	indexVarName string
	valueVarName string
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
	ctx := "start"
	for _, o := range b.expr[1:] {
		switch o.typ {
		case variable:
			switch ctx {
			case "start":
				ctx = "index"
				if len(o.idents) > 1 {
					return fmt.Errorf("invalid for statement(invalid index): %v", o)
				}
				b.indexVarName = o.idents[0].name
			case "comma":
				ctx = "value"
				if len(o.idents) > 1 {
					return fmt.Errorf("invalid for statement(invalid value): %v", o)
				}
				b.valueVarName = o.idents[0].name
			case "in":
				ctx = "target"
				b.iterObj = o
				ctx = "finish"
			default:
				return fmt.Errorf("invalid for statement: %v", b.expr)
			}
		case punct:
			if o.idents[0].typ != commaIdent {
				return fmt.Errorf("invalid for statement: %v", b.expr)
			}
			switch ctx {
			case "index":
				ctx = "comma"
			default:
				return fmt.Errorf("invalid for statement: %v", b.expr)
			}
		case keyword:
			if o.idents[0].typ != inIdent {
				return fmt.Errorf("invalid for statement: %v", b.expr)
			}
			switch ctx {
			case "value":
				ctx = "in"
			default:
				return fmt.Errorf("invalid for statement: %v", b.expr)
			}
		default:
			return fmt.Errorf("invalid for statement: %v", b.expr)

		}
	}
	if ctx != "finish" {
		return fmt.Errorf("invalid for statement: %v", b.expr)
	}
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
	val := reflect.ValueOf(o)
	builder := strings.Builder{}
	switch kind {
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			return "", nil
		}
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
			env[b.indexVarName] = i
			env[b.valueVarName] = elemVal.Interface()
			for _, sb := range b.subBlocks {
				s, err := sb.run(env)
				if err != nil {
					return "", err
				}
				builder.WriteString(s)
			}
		}
	case reflect.Map:
		keyList := val.MapKeys()
		if len(keyList) == 0 {
			return "", nil
		}
		for _, key := range keyList {
			elemVal := val.MapIndex(key)
			for elemVal.Kind() == reflect.Ptr || elemVal.Kind() == reflect.Interface {
				elemVal = elemVal.Elem()
				if !elemVal.IsValid() {
					return "", fmt.Errorf("invalid map element: key %v", key)
				}
			}
			if !elemVal.IsValid() {
				return "", fmt.Errorf("invalid map element: key %v", key)
			}
			env[b.indexVarName] = key.Interface()
			env[b.valueVarName] = elemVal.Interface()
			for _, sb := range b.subBlocks {
				s, err := sb.run(env)
				if err != nil {
					return "", err
				}
				builder.WriteString(s)
			}
		}
	default:
		return "", fmt.Errorf("only array, slice and map are supported by for loop: %v", val.Interface())
	}
	return builder.String(), nil
}

type switchcaseBlock struct {
	src       string
	expr      []object
	subBlocks []block
	caseVals  []*object
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
	switch b.expr[0].idents[0].typ {
	case caseIdent:
		if len(b.expr) < 2 {
			return fmt.Errorf("invalid switch case: %v", b.expr)
		}
		for i, o := range b.expr[1:] {
			if o.idents[0].typ != commaIdent {
				b.caseVals = append(b.caseVals, &(b.expr[i+1]))
			}
		}

	case defaultIdent:
		if len(b.expr) != 1 {
			return fmt.Errorf("invalid switch default case: %v", b.expr)
		}
	default:
		return fmt.Errorf("invalid switch case block: %v", b.expr)
	}
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
	if len(b.caseVals) == 0 {
		for _, sb := range b.subBlocks {
			s, err := sb.run(env)
			if err != nil {
				return "", err
			}
			builder.WriteString(s)
		}
		return builder.String(), nil
	}
	for _, caseVal := range b.caseVals {
		cv, err := caseVal.getVal(env)
		if err != nil {
			return "", err
		}
		switch caseVal.typ {
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
				return "", fmt.Errorf("switch target value is not string: %v", env["_targetVal"])
			}
		case numconst:
			var isFloat bool
			for _, id := range caseVal.idents {
				if id.typ == dotIdent {
					isFloat = true
				}
			}
			if isFloat {
				var switchValue, caseValue float64
				switch env["_targetVal"].(type) {
				case float64:
					switchValue = env["_targetVal"].(float64)
				case float32:
					switchValue = float64(env["_targetVal"].(float32))
				default:
					return "", fmt.Errorf("invalid switch value: %v", env["_targetVal"])
				}
				switch cv.(type) {
				case float64:
					caseValue = cv.(float64)
				case float32:
					caseValue = float64(cv.(float32))
				default:
					return "", fmt.Errorf("invalid switch case value: %v", cv)
				}
				if switchValue == caseValue {
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
				var switchValue, caseValue int64
				switch v := env["_targetVal"].(type) {
				case int:
					switchValue = int64(v)
				case int8:
					switchValue = int64(v)
				case int16:
					switchValue = int64(v)
				case int32:
					switchValue = int64(v)
				case int64:
					switchValue = v
				default:
					return "", fmt.Errorf("invalid switch value: %v", env["_targetVal"])
				}
				switch v := cv.(type) {
				case int:
					caseValue = int64(v)
				case int8:
					caseValue = int64(v)
				case int16:
					caseValue = int64(v)
				case int32:
					caseValue = int64(v)
				case int64:
					caseValue = v
				default:
					return "", fmt.Errorf("invalid switch case value: %v", cv)
				}
				if switchValue == caseValue {
					for _, sb := range b.subBlocks {
						s, err := sb.run(env)
						if err != nil {
							return "", err
						}
						builder.WriteString(s)
					}
					return builder.String(), nil
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
				return "", fmt.Errorf("switch target value is not bool: %v", env["_targetVal"])
			}
		default:
			return "", fmt.Errorf("case value is not comparable: %v", caseVal)
		}
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
		return val, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val), nil
	case float32, float64:
		return fmt.Sprintf("%f", val), nil
	case bool:
		return fmt.Sprintf("%t", val), nil
	default:
		return "", fmt.Errorf("invalid value type: %v", v)
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
		case reflect.Bool:
			return boolCompare(lv.(bool), rv.(bool), e.op)
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
		isMatch, err := e.compare(env)
		if err != nil {
			return false, err
		}
		if isMatch {
			if e.nextExpr != nil {
				if e.relOp.idents[0].typ == andIdent {
					return e.nextExpr.eval(env)
				}
				return true, nil
			}
			return true, nil
		} else {
			if e.nextExpr != nil {
				if e.relOp.idents[0].typ == andIdent {
					return false, nil
				}
				return e.nextExpr.eval(env)
			}
			return false, nil
		}
	}
}
