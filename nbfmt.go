package nbfmt

// var seqRe = regexp.MustCompile(`\d+`)
// var fmtRe = regexp.MustCompile(`{{([^{].*?)?}}`)

// var queryRe = regexp.MustCompile(`(".*?"|[^>^ ]+)`)
// var intRe = regexp.MustCompile(`-?\d+`)
// var floatRe = regexp.MustCompile(`\d+\.\d+`)
// var boolRe = regexp.MustCompile(`(true|false)`)
// var complexRe = regexp.MustCompile(`\(\d+\.\d+\s?,\s?\d+\.\d+\)`)
// var stringRe = regexp.MustCompile(`"(.*?)"`)
// var fieldRe = regexp.MustCompile(`^[_a-zA-z][\w_]*$`)

// func convertString(val reflect.Value) string {
// 	return fmt.Sprintf("%s", val.Interface())
// }

// func convertInt(val reflect.Value) string {
// 	return fmt.Sprintf("%d", val.Interface())
// }

// func convertFloat(val reflect.Value) string {
// 	return fmt.Sprintf("%f", val.Interface())
// }

// func convertBool(val reflect.Value) string {
// 	return fmt.Sprintf("%t", val.Interface())
// }

// func convertStruct(val reflect.Value) string {
// 	var t fmt.Stringer
// 	if val.Type().Implements(reflect.TypeOf(t)) {
// 		return val.MethodByName("String").Call(nil)[0].Interface().(string)
// 	}
// 	jsonByte, _ := json.Marshal(val.Interface())
// 	return string(jsonByte)
// }

// func convertSlice(val reflect.Value) string {
// 	jsonByte, _ := json.Marshal(val.Interface())
// 	return string(jsonByte)
// }

// func convertMap(val reflect.Value) string {
// 	jsonByte, _ := json.Marshal(val.Interface())
// 	return string(jsonByte)
// }

// func convert(val reflect.Value) (string, error) {
// 	switch val.Kind() {
// 	case reflect.String:
// 		return convertString(val), nil
// 	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
// 		return convertInt(val), nil
// 	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
// 		return convertFloat(val), nil
// 	case reflect.Bool:
// 		return convertBool(val), nil
// 	case reflect.Struct:
// 		return convertStruct(val), nil
// 	case reflect.Slice, reflect.Array:
// 		return convertSlice(val), nil
// 	case reflect.Map:
// 		return convertMap(val), nil
// 	case reflect.Ptr:
// 		if val.CanAddr() {
// 			val = val.Elem()
// 			return convert(val)
// 		}
// 		return "", InvalidPtrError{val}
// 	default:
// 		return "", NotSupportedTypeError{val}
// 	}
// }

// func procQuery(query string) ([]interface{}, error) {
// 	l := queryRe.FindAllString(query, -1)
// 	if len(l) == 0 {
// 		return nil, nil
// 	}
// 	resultList := make([]interface{}, len(l))
// 	for i, q := range l {
// 		switch {
// 		case stringRe.MatchString(q):
// 			resultList[i] = stringRe.FindStringSubmatch(q)[1]
// 		case intRe.MatchString(q):
// 			i64, err := strconv.ParseInt(q, 10, 64)
// 			if err != nil {
// 				return nil, err
// 			}
// 			resultList[i] = int(i64)
// 		case floatRe.MatchString(q):
// 			f64, err := strconv.ParseFloat(q, 64)
// 			if err != nil {
// 				return nil, err
// 			}
// 			resultList[i] = f64
// 		case boolRe.MatchString(q):
// 			if q == "true" {
// 				resultList[i] = true
// 			} else {
// 				resultList[i] = false
// 			}
// 		case complexRe.MatchString(q):
// 			q = strings.Replace(strings.Trim(q, "()"), " ", "", -1)
// 			l := strings.Split(q, ",")
// 			f1, err := strconv.ParseFloat(l[0], 64)
// 			if err != nil {
// 				return nil, err
// 			}
// 			f2, err := strconv.ParseFloat(l[1], 64)
// 			if err != nil {
// 				return nil, err
// 			}
// 			resultList[i] = complex(f1, f2)
// 		case fieldRe.MatchString(q):
// 			resultList[i] = q
// 		default:
// 			return nil, InvalidQueryError{query, q}

// 		}
// 	}
// 	return resultList, nil
// }

// func getObj(q interface{}, val reflect.Value) (reflect.Value, error) {
// 	var isValid bool
// 	val, isValid = stripPtr(val)
// 	if !isValid {
// 		return val, InvalidValueError{q}
// 	}
// 	switch val.Kind() {
// 	case reflect.Array, reflect.Slice:
// 		if i, ok := q.(int); !ok {
// 			return val, InvalidSeqQueryError{q}
// 		} else {
// 			length, err := getLen(val)
// 			if err != nil {
// 				return val, err
// 			}
// 			i, err = fixIndex(i, length)
// 			if err != nil {
// 				return val, err
// 			}
// 			val, isValid = stripPtr(val.Index(i))
// 			if !isValid {
// 				return val, InvalidValueError{q}
// 			}
// 			return val, nil
// 		}
// 	case reflect.Map:
// 		t := reflect.TypeOf(q)
// 		if val.Type().Key().Name() != t.Name() {
// 			return val, MapKeyTypeError{requiredType: val.Type().Key(), providedType: t}
// 		}
// 		val, isValid = stripPtr(val.MapIndex(reflect.ValueOf(q)))
// 		if !isValid {
// 			return val, InvalidValueError{q}
// 		}
// 		return val, nil
// 	case reflect.Struct:
// 		switch v := q.(type) {
// 		case int:
// 			length, err := getLen(val)
// 			if err != nil {
// 				return val, err
// 			}
// 			v, err = fixIndex(v, length)
// 			if err != nil {
// 				return val, err
// 			}
// 			val, isValid = stripPtr(val.Field(v))
// 			if !isValid {
// 				return val, InvalidValueError{q}
// 			}
// 			return val, nil
// 		case string:
// 			val, isValid = stripPtr(val.FieldByName(v))
// 			if !isValid {
// 				return val, InvalidValueError{q}
// 			}
// 			return val, nil
// 		default:
// 			return val, InvalidStructFieldQueryError{q}
// 		}
// 	case reflect.Invalid:
// 		return val, InvalidValueError{q}
// 	default:
// 		return val, NotSupportedTypeError{val}
// 	}
// }

// func find(query string, value interface{}) (reflect.Value, error) {
// 	val := reflect.ValueOf(value)
// 	qList, err := procQuery(query)
// 	if err != nil {
// 		return val, err
// 	}
// 	if len(qList) == 0 {
// 		if value == nil {
// 			return reflect.ValueOf(""), nil
// 		}
// 		var isValid bool
// 		val, isValid = stripPtr(val)
// 		if !isValid {
// 			return val, InvalidValueError{"root value"}
// 		}
// 		return val, nil
// 	}
// 	for _, q := range qList {
// 		val, err = getObj(q, val)
// 		if err != nil {
// 			return val, err
// 		}
// 	}
// 	return val, nil
// }

// func stripPtr(val reflect.Value) (reflect.Value, bool) {
// 	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
// 		val = val.Elem()
// 	}
// 	if !val.IsValid() {
// 		return val, false
// 	}
// 	return val, true
// }

// func getLen(val reflect.Value) (int, error) {
// 	switch val.Kind() {
// 	case reflect.Slice, reflect.Array:
// 		return val.Len(), nil
// 	case reflect.Struct:
// 		return val.NumField(), nil
// 	default:
// 		return -1, NotSeqTypeError{val}
// 	}
// }

// func fixIndex(index, length int) (int, error) {
// 	if index < 0 {
// 		if length+index < 0 {
// 			return -1, IndexOutRangeError{index, length}
// 		}
// 		index = length + index
// 	}
// 	if index >= length {
// 		return -1, IndexOutRangeError{index, length}
// 	}
// 	return index, nil
// }

// //Fmt format template by value
// func Fmt(temp string, value interface{}) (string, error) {
// 	l := fmtRe.FindAllStringSubmatch(temp, -1)
// 	for _, t := range l {
// 		obj, err := find(t[1], value)
// 		if err != nil {
// 			return "", err
// 		}
// 		s, err := convert(obj)
// 		if err != nil {
// 			return "", err
// 		}
// 		temp = strings.Replace(temp, t[0], s, -1)
// 	}
// 	return temp, nil
// }

func Fmt(src string, env map[string]interface{}) (string, error) {
	temp, err := parseTemplate(src)
	if err != nil {
		return "", err
	}
	s, err := temp.run(env)
	if err != nil {
		return "", err
	}
	return s, nil
}
