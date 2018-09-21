package nbfmt

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type identType int

const (
	varIdent              identType = iota // xxx
	strIdent                               // "xxx"
	byteIdent                              // 'x'
	intIdent                               // 123
	floatIdent                             // 123.45
	boolIdent                              // true and false
	ifIdent                                // if
	elseifIdent                            // elseif
	elseIdent                              // else
	endifIdent                             // endif
	forIdent                               // for
	inIdent                                // in
	endforIdent                            // endfor
	switchIdent                            // switch
	caseIdent                              // case
	defaultIdent                           // default
	endswitchIdent                         // endswitch
	commaIdent                             // ,
	asteriskIdent                          // *
	dotIdent                               // .
	plugIdent                              // +
	subIdent                               // -
	divIdent                               // /
	lessThanIdent                          // <
	lessThanEqualIdent                     // <=
	greatThanIdent                         // >
	greatThanEqualIdent                    // >=
	equalIdent                             // ==
	notEqualIdent                          // !=
	exclamationIdent                       // !
	andIdent                               // &&
	orIdent                                // ||
	leftParenthesisIdent                   // (
	rightParenthesisIdent                  // )
	leftBracketIdent                       // [
	rightBracketIdent                      // ]
	nilIdent                               // nil
)

type ident struct {
	src string
	typ identType
}

func (id *ident) String() string {
	return id.src
}

func (id *ident) eval(env map[string]interface{}) (interface{}, error) {
	switch id.typ {
	case strIdent:
		return strings.Trim(id.src, "\""), nil
	case byteIdent:
		return id.src[1], nil
	case intIdent:
		return strconv.ParseInt(id.src, 10, 64)
	case floatIdent:
		return strconv.ParseFloat(id.src, 64)
	case boolIdent:
		return strconv.ParseBool(id.src)
	case varIdent:
		val, ok := env[id.src]
		if !ok {
			return nil, fmt.Errorf("nbfmt.ident.eval() error: %s is not exist in env map", id.src)
		}
		switch v := val.(type) {
		case int:
			return int64(v), nil
		case float32:
			return float64(v), nil
		}
		return val, nil
	case nilIdent:
		return nil, nil
	default:
		return nil, fmt.Errorf("nbfmt.ident.eval() error: %s cannot be eval", id.src)

	}
}

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
	src    string
	typ    stmtType
	idents []*ident
}

func (s *stmt) String() string {
	return s.src
}

type block interface {
	getSrc() string
	appendSrc(string)
	appendSubBlock(block)
	// blow is new edition
	eval(map[string]interface{}) (string, error)
}

type template struct {
	blocks []block
}

func (t template) eval(env map[string]interface{}) (string, error) {
	builder := strings.Builder{}
	for _, b := range t.blocks {
		s, err := b.eval(env)
		if err != nil {
			return "", err
		}
		builder.WriteString(s)
	}
	return builder.String(), nil
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

func (b *tempBlock) appendSubBlock(blk block) {}

func (b *tempBlock) eval(env map[string]interface{}) (string, error) {
	return b.src, nil
}

type ifcaseBlock struct {
	src       string
	subBlocks []block
	exprObj   *expression
	//blow is new edition
	stmt *stmt
	exp  *expression
}

func (b *ifcaseBlock) getSrc() string {
	return b.src
}

func (b *ifcaseBlock) appendSrc(s string) {
	b.src += s
}

func (b *ifcaseBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *ifcaseBlock) eval(env map[string]interface{}) (string, error) {
	expVal, err := b.exp.copy().eval(env)
	if err != nil {
		return "", err
	}
	if isMatch, ok := expVal.(bool); !ok {
		return "", fmt.Errorf("nbfmt.ifcaseBlock.eval() error: the type of expression in if case block must be bool (%v)\n", b.exp)
	} else {
		builder := strings.Builder{}
		if isMatch {
			for _, sb := range b.subBlocks {
				s, err := sb.eval(env)
				if err != nil {
					return "", err
				}
				builder.WriteString(s)
			}
			return builder.String(), nil
		}
		return "", nil
	}
}

type defaultBlock struct {
	src       string
	stmt      *stmt
	subBlocks []block
}

func (b *defaultBlock) getSrc() string {
	return b.src
}

func (b *defaultBlock) appendSrc(s string) {
	b.src += s
}

func (b *defaultBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *defaultBlock) eval(env map[string]interface{}) (string, error) {
	builder := strings.Builder{}
	for _, sb := range b.subBlocks {
		s, err := sb.eval(env)
		if err != nil {
			return "", err
		}
		builder.WriteString(s)
	}
	return builder.String(), nil
}

type ifBlock struct {
	src       string
	subBlocks []block
	//blow is new edition
	caseBlocks   []*ifcaseBlock
	defaultBlock *defaultBlock
	stmts        []*stmt
}

func (b *ifBlock) getSrc() string {
	return b.src
}

func (b *ifBlock) appendSrc(s string) {
	b.src += s
}

func (b *ifBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *ifBlock) run(env map[string]interface{}) (string, error) {
	for _, sb := range b.subBlocks {
		s, err := sb.eval(env)
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
	}
	return "", nil
}

func (b *ifBlock) eval(env map[string]interface{}) (string, error) {
	for _, sb := range b.caseBlocks {
		s, err := sb.eval(env)
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
	}
	if b.defaultBlock != nil {
		s, err := b.defaultBlock.eval(env)
		if err != nil {
			return "", err
		}
		return s, nil
	}
	return "", nil
}

type forBlock struct {
	src          string
	subBlocks    []block
	indexVarName string
	valueVarName string
	//blow is new edition
	stmt    *stmt
	objExpr *expression
}

func (b *forBlock) getSrc() string {
	return b.src
}

func (b *forBlock) appendSrc(s string) {
	b.src += s
}

func (b *forBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *forBlock) eval(env map[string]interface{}) (string, error) {
	localEnv := make(map[string]interface{})
	for k, v := range env {
		localEnv[k] = v
	}
	iterObj, err := b.objExpr.copy().eval(env)
	if err != nil {
		return "", err
	}
	builder := strings.Builder{}
	iterObjVal := reflect.ValueOf(iterObj)
	switch iterObjVal.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < iterObjVal.Len(); i++ {
			localEnv[b.indexVarName] = int64(i)
			localEnv[b.valueVarName] = iterObjVal.Index(i).Interface()
			for _, sb := range b.subBlocks {
				s, err := sb.eval(localEnv)
				if err != nil {
					return "", err
				}
				builder.WriteString(s)
			}
		}
		return builder.String(), nil
	case reflect.Map:
		keys := iterObjVal.MapKeys()
		for _, key := range keys {
			localEnv[b.indexVarName] = key.Interface()
			localEnv[b.valueVarName] = iterObjVal.MapIndex(key).Interface()
			for _, sb := range b.subBlocks {
				s, err := sb.eval(localEnv)
				if err != nil {
					return "", err
				}
				builder.WriteString(s)
			}
		}
		return builder.String(), nil
	default:
		return "", fmt.Errorf("nbfmt.forBlock.eval() the object for iterating is not a array(slice) or a map (%s)\n", b.objExpr.String())
	}
}

type switchcaseBlock struct {
	src       string
	subBlocks []block
	//blow is new edition
	exps []*expression
	stmt *stmt
}

func (b *switchcaseBlock) getSrc() string {
	return b.src
}

func (b *switchcaseBlock) appendSrc(s string) {
	b.src += s
}

func (b *switchcaseBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *switchcaseBlock) init() error {
	return nil
}

func (b *switchcaseBlock) eval(env map[string]interface{}) (string, error) {
	tarVal := env["_targetVal"]
	for _, e := range b.exps {
		expVal, err := e.copy().eval(env)
		if err != nil {
			return "", err
		}
		if tarVal == expVal {
			builder := strings.Builder{}
			for _, sb := range b.subBlocks {
				s, err := sb.eval(env)
				if err != nil {
					return "", err
				}
				builder.WriteString(s)
			}
			return builder.String(), nil
		}
	}
	return "", nil
}

type switchBlock struct {
	src       string
	subBlocks []block
	//blow is new edition
	caseBlocks   []*switchcaseBlock
	defaultBlock *defaultBlock
	exp          *expression
	stmt         *stmt
}

func (b *switchBlock) getSrc() string {
	return b.src
}

func (b *switchBlock) appendSrc(s string) {
	b.src += s
}

func (b *switchBlock) appendSubBlock(blk block) {
	b.subBlocks = append(b.subBlocks, blk)
}

func (b *switchBlock) eval(env map[string]interface{}) (string, error) {
	localEnv := make(map[string]interface{})
	for k, v := range env {
		localEnv[k] = v
	}
	tarVal, err := b.exp.copy().eval(env)
	if err != nil {
		return "", err
	}
	localEnv["_targetVal"] = tarVal
	for _, cb := range b.caseBlocks {
		s, err := cb.eval(localEnv)
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
	}
	if b.defaultBlock != nil {
		return b.defaultBlock.eval(localEnv)
	}
	return "", nil
}

type valueBlock struct {
	src string
	//blow is new edition
	exp  *expression
	stmt *stmt
}

func (b *valueBlock) getSrc() string {
	return b.src
}

func (b *valueBlock) appendSrc(s string) {
	b.src += s
}

func (b *valueBlock) appendSubBlock(blk block) {}

func (b *valueBlock) eval(env map[string]interface{}) (string, error) {
	expVal, err := b.exp.copy().eval(env)
	if err != nil {
		return "", err
	}
	switch val := expVal.(type) {
	case string:
		return val, nil
	case byte:
		return fmt.Sprintf("%c", val), nil
	case int, int8, int16, int32, int64, uint, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val), nil
	case float32, float64:
		return fmt.Sprintf("%f", val), nil
	case bool:
		return fmt.Sprintf("%t", val), nil
	case nil:
		return "nil", nil
	default:
		return "", fmt.Errorf("nbfmt.valueBlock.eval() error: unsupported value block type (%s)", b.exp.String())
	}
}

type stmtStack struct {
	stmtList *[]*stmt
}

func newStmtStack(l *[]*stmt) *stmtStack {
	return &stmtStack{l}
}

func (ss *stmtStack) pop() *stmt {
	s, remain := (*ss.stmtList)[0], (*ss.stmtList)[1:]
	ss.stmtList = &remain
	return s
}

func (ss *stmtStack) push(s *stmt) {
	l := make([]*stmt, len(*(ss.stmtList))+1)
	l[0] = s
	copy(l[1:], (*ss.stmtList))
	*ss.stmtList = l
}

func (ss *stmtStack) len() int {
	return len(*(ss.stmtList))
}

func (ss *stmtStack) checkType() stmtType {
	return (*ss.stmtList)[0].typ
}

type operator struct {
	src      string
	priority int
}

func (o *operator) String() string {
	return o.src
}

var indexOperator = operator{"[]", 7}
var dotOperator = operator{".", 7}
var derefOperator = operator{"*", 6}
var notOperator = operator{"!", 6}
var mulOperator = operator{"*", 5}
var divOperator = operator{"/", 5}
var plugOperator = operator{"+", 4}
var subOperator = operator{"-", 4}
var equalOperator = operator{"==", 3}
var notEqualOperator = operator{"!=", 3}
var lessThanOperator = operator{"<", 3}
var lessThanEqualOperator = operator{"<=", 3}
var greatThanOperator = operator{">", 3}
var greatThanEqualOperator = operator{">=", 3}
var andOperator = operator{"&&", 2}
var orOperator = operator{"||", 1}

type expression struct {
	subExpr  *expression
	parExpr  *expression
	nextExpr *expression
	prevExpr *expression
	ident    *ident
	operator *operator
	value    interface{}
}

func (e *expression) String() string {
	var identStr, operatorStr string
	if e.ident != nil {
		identStr = e.ident.String()
	} else {
		identStr = "nil"
	}
	if e.operator != nil {
		operatorStr = e.operator.String()
	} else {
		operatorStr = "nil"
	}
	return fmt.Sprintf(`
ident: %s
operator: %s`, identStr, operatorStr)
}

func (e *expression) pop() *expression {
	priExpr := e
	curExpr := e
	nxtExpr := e.nextExpr
	for nxtExpr != nil {
		if nxtExpr.operator != nil {
			if nxtExpr.operator.priority > curExpr.operator.priority {
				priExpr = nxtExpr
				curExpr = nxtExpr
				nxtExpr = nxtExpr.nextExpr
			} else {
				curExpr = nxtExpr
				nxtExpr = nxtExpr.nextExpr
			}
		} else {
			break
		}
	}
	if priExpr.subExpr != nil && priExpr.subExpr.value == nil {
		return priExpr.subExpr.pop()
	}
	if priExpr.operator == &indexOperator && priExpr.nextExpr.value == nil {
		return priExpr.nextExpr.pop()
	}
	return priExpr
}

func assertToInt(lv, rv interface{}) (int64, int64, bool) {
	ilv, ok := lv.(int64)
	if !ok {
		return 0, 0, false
	}
	irv, ok := rv.(int64)
	if !ok {
		return 0, 0, false
	}
	return ilv, irv, true
}

func assertToFloat(lv, rv interface{}) (float64, float64, bool) {
	flv, ok := lv.(float64)
	if !ok {
		return 0, 0, false
	}
	frv, ok := rv.(float64)
	if !ok {
		return 0, 0, false
	}
	return flv, frv, true
}

func assertToStr(lv, rv interface{}) (string, string, bool) {
	slv, ok := lv.(string)
	if !ok {
		return "", "", false
	}
	srv, ok := rv.(string)
	if !ok {
		return "", "", false
	}
	return slv, srv, true
}

func assertToBool(lv, rv interface{}) (bool, bool, bool) {
	blv, ok := lv.(bool)
	if !ok {
		return false, false, false
	}
	brv, ok := rv.(bool)
	if !ok {
		return false, false, false
	}
	return blv, brv, true
}

func assertToByte(lv, rv interface{}) (byte, byte, bool) {
	blv, ok := lv.(byte)
	if !ok {
		return 0, 0, false
	}
	brv, ok := rv.(byte)
	if !ok {
		return 0, 0, false
	}
	return blv, brv, true
}

func add(lv, rv interface{}) (interface{}, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			slv, srv, ok := assertToStr(lv, rv)
			if !ok {
				return nil, fmt.Errorf("nbfmt.add() cannot add (%T and %T)\n", lv, rv)
			}
			return slv + srv, nil
		}
		return flv + frv, nil
	}
	return ilv + irv, nil
}

func sub(lv, rv interface{}) (interface{}, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			return nil, fmt.Errorf("nbfmt.sub() cannot subtract (%T and %T)\n", lv, rv)
		}
		return flv - frv, nil
	}
	return ilv - irv, nil

}

func mul(lv, rv interface{}) (interface{}, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			return nil, fmt.Errorf("nbfmt.mul() cannot multiply (%T and %T)\n", lv, rv)
		}
		return flv * frv, nil
	}
	return ilv * irv, nil
}

func div(lv, rv interface{}) (interface{}, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			return nil, fmt.Errorf("nbfmt.div() cannot div (%T and %T)\n", lv, rv)
		}
		return flv / frv, nil
	}
	return ilv / irv, nil
}

func equal(lv, rv interface{}) (bool, error) {
	return lv == rv, nil
}

func notEqual(lv, rv interface{}) (bool, error) {
	return lv != rv, nil
}

func lessThan(lv, rv interface{}) (bool, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			return false, fmt.Errorf("nbfmt.lessThan() cannot compare (%T and %T)\n", lv, rv)
		}
		return flv < frv, nil
	}
	return ilv < irv, nil

}

func lessThanEqual(lv, rv interface{}) (bool, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			return false, fmt.Errorf("nbfmt.lessEqualThan() cannot compare (%T and %T)\n", lv, rv)
		}
		return flv <= frv, nil
	}
	return ilv <= irv, nil
}

func greatThan(lv, rv interface{}) (bool, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			return false, fmt.Errorf("nbfmt.greatThan() cannot compare (%T and %T)\n", lv, rv)
		}
		return flv > frv, nil
	}
	return ilv > irv, nil
}

func greatThanEqual(lv, rv interface{}) (bool, error) {
	ilv, irv, ok := assertToInt(lv, rv)
	if !ok {
		flv, frv, ok := assertToFloat(lv, rv)
		if !ok {
			return false, fmt.Errorf("nbfmt.greatEqualThan() cannot compare (%T and %T)\n", lv, rv)
		}
		return flv >= frv, nil
	}
	return ilv >= irv, nil
}

func and(lv, rv interface{}) (bool, error) {
	blv, brv, ok := assertToBool(lv, rv)
	if !ok {
		return false, fmt.Errorf("nbfmt.and() cannot calculate (%T and %T)\n", lv, rv)
	}
	return blv && brv, nil
}

func or(lv, rv interface{}) (bool, error) {
	blv, brv, ok := assertToBool(lv, rv)
	if !ok {
		return false, fmt.Errorf("nbfmt.or() cannot calculate (%T and %T)\n", lv, rv)
	}
	return blv || brv, nil
}

func deref(i interface{}) (interface{}, error) {
	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.Ptr {
		return nil, fmt.Errorf("nbfmt.deref() error: invalid dereference operate for %T (%v)", i, i)
	}
	return val.Elem().Interface(), nil
}

func field(s interface{}, f *ident) (interface{}, error) {
	if f.typ != varIdent {
		return nil, fmt.Errorf("nbfmt.field() error: field ident is not varIdent (%s)", f.src)
	}
	val := reflect.ValueOf(s)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("nbfmt.field() error: %v is not struct", s)
	}
	field := val.FieldByName(f.src)
	if !field.IsValid() {
		return nil, fmt.Errorf("nbfmt.field() error: %s field is not valid", f.src)
	}
	return field.Interface(), nil
}

func index(obj, idx interface{}) (interface{}, error) {
	val := reflect.ValueOf(obj)
	switch val.Kind() {
	case reflect.Map:
		v := val.MapIndex(reflect.ValueOf(idx))
		if !v.IsValid() {
			return nil, fmt.Errorf("nbfmt.index() error: invalid map element (index: %v)", idx)
		}
		result := v.Interface()
		switch r := result.(type) {
		case int:
			return int64(r), nil
		case float32:
			return float64(r), nil
		}
		return result, nil
	case reflect.Array, reflect.Slice:
		if int64Idx, ok := idx.(int64); ok {
			v := val.Index(int(int64Idx))
			if !v.IsValid() {
				return nil, fmt.Errorf("nbfmt.index() error: invalid array element (index: %v)", idx)
			}
			result := v.Interface()
			switch r := result.(type) {
			case int:
				return int64(r), nil
			case float32:
				return float64(r), nil
			}
			return result, nil
		}
		return nil, fmt.Errorf("nbfmt.index() error: invalid index for array or slice (index: %v, type: %T)", idx, idx)
	default:
		return nil, fmt.Errorf("nbfmt.index() error: cannot index %T (%v)", obj, obj)
	}
}

func not(i interface{}) (bool, error) {
	boolVal, ok := i.(bool)
	if !ok {
		return false, fmt.Errorf("nbfmt.not() error: invalid not operate for %T type (%v)", i, i)
	}
	return !boolVal, nil
}

func (e *expression) copy() *expression {
	ne := &expression{}
	if e.ident != nil {
		nid := *e.ident
		ne.ident = &nid
	}
	// id := *e.ident
	op := e.operator
	// ne.ident = &id
	ne.operator = op
	if e.subExpr != nil {
		ne.subExpr = e.subExpr.copy()
		ne.subExpr.parExpr = ne
	}
	if e.nextExpr != nil {
		ne.nextExpr = e.nextExpr.copy()
		ne.nextExpr.prevExpr = ne
	}
	return ne
}

func (e *expression) eval(env map[string]interface{}) (interface{}, error) {
	priExpr := e.pop()
	nextExpr := priExpr.nextExpr
	leftVal := func() (interface{}, error) {
		if priExpr.value == nil && priExpr.ident != nil {
			v, err := priExpr.ident.eval(env)
			if err != nil {
				return nil, err
			}
			priExpr.value = v
		}
		return priExpr.value, nil
	}
	rightVal := func() (interface{}, error) {
		if nextExpr.value == nil {
			v, err := nextExpr.ident.eval(env)
			if err != nil {
				return nil, err
			}
			nextExpr.value = v
		}
		return nextExpr.value, nil
	}
	virExpr := &expression{}
	switch priExpr.operator {
	case &derefOperator:
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := deref(rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &notOperator:
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := not(rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &plugOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := add(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &subOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := sub(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &mulOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := mul(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &divOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := mul(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &dotOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		result, err := field(lv, nextExpr.ident)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &indexOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := index(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &equalOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := equal(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &notEqualOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := notEqual(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &lessThanOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := lessThan(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &lessThanEqualOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := lessThanEqual(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &greatThanOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := greatThan(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &greatThanEqualOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := greatThanEqual(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &andOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := and(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case &orOperator:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		rv, err := rightVal()
		if err != nil {
			return nil, err
		}
		result, err := or(lv, rv)
		if err != nil {
			return nil, err
		}
		virExpr.value = result
	case nil:
		lv, err := leftVal()
		if err != nil {
			return nil, err
		}
		virExpr.value = lv
	}
	if nextExpr != nil {
		if nextExpr.nextExpr != nil {
			virExpr.nextExpr = nextExpr.nextExpr
			virExpr.operator = nextExpr.operator
			nextExpr.nextExpr.prevExpr = virExpr
			if priExpr.prevExpr != nil {
				virExpr.prevExpr = priExpr.prevExpr
				priExpr.prevExpr.nextExpr = virExpr
			} else {
				if priExpr.parExpr != nil {
					virExpr.parExpr = priExpr.parExpr
					priExpr.parExpr.subExpr = virExpr
				} else {
					e = virExpr
				}
			}
		} else {
			if priExpr.prevExpr != nil {
				virExpr.prevExpr = priExpr.prevExpr
				priExpr.prevExpr.nextExpr = virExpr
			} else {
				if priExpr.parExpr != nil {
					priExpr.parExpr.value = virExpr.value
				} else {
					return virExpr.value, nil
				}
			}
		}
	} else {
		if priExpr.prevExpr != nil {
			virExpr.prevExpr = priExpr.prevExpr
			priExpr.prevExpr.nextExpr = virExpr
		} else {
			if priExpr.parExpr != nil {
				priExpr.parExpr.value = virExpr.value
			} else {
				return virExpr.value, nil
			}
		}
	}
	return e.eval(env)

}
