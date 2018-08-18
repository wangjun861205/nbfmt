package nbfmt

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var ErrValueType = errors.New("nbfmt: Not valid value type")
var ErrNotEnoughValue = errors.New("nbfmt: Not enough values to format template")
var ErrNotSeqTemp = errors.New("nbfmt: the templates in FmtBySeq() must be sequence template")
var ErrNotNameTemp = errors.New("nbfmt: the templates in FmtByName() must be name template")
var ErrNotValidPtr = errors.New("nbfmt: the pointer of value is not valid")
var ErrNotSupportedType = errors.New("nbfmt: the type of value is not supported")
var ErrNotSliceOrArray = errors.New("nbfmt: the type of value is not slice or array")
var ErrNotValidValue = errors.New("nbfmt: not valid value")
var ErrNotSeqVal = errors.New("nbfmt: the type of value is not a sequence type")
var ErrNotMapVal = errors.New("nbfmt: the type of value is not a map type")
var ErrNotStructVal = errors.New("nbfmt: the type of value is not a struct type")

var seqRe = regexp.MustCompile(`\d+`)
var fmtRe = regexp.MustCompile(`{{(.*?)}}`)
var queryRe = regexp.MustCompile(`(".*?"|[^\.]+)`)
var mapIndexRe = regexp.MustCompile(`"(.*?)"`)

func convertString(val reflect.Value) string {
	return fmt.Sprintf("%s", val.Interface())
}

func convertInt(val reflect.Value) string {
	return fmt.Sprintf("%d", val.Interface())
}

func convertFloat(val reflect.Value) string {
	return fmt.Sprintf("%f", val.Interface())
}

func convertBool(val reflect.Value) string {
	return fmt.Sprintf("%t", val.Interface())
}

func convertStruct(val reflect.Value) string {
	var t fmt.Stringer
	if val.Type().Implements(reflect.TypeOf(t)) {
		return val.MethodByName("String").Call(nil)[0].Interface().(string)
	}
	jsonByte, _ := json.Marshal(val.Interface())
	return string(jsonByte)
}

func convertSlice(val reflect.Value) string {
	jsonByte, _ := json.Marshal(val.Interface())
	return string(jsonByte)
}

func convertMap(val reflect.Value) string {
	jsonByte, _ := json.Marshal(val.Interface())
	return string(jsonByte)
}

func convert(val reflect.Value) (string, error) {
	switch val.Kind() {
	case reflect.String:
		return convertString(val), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return convertInt(val), nil
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return convertFloat(val), nil
	case reflect.Bool:
		return convertBool(val), nil
	case reflect.Struct:
		return convertStruct(val), nil
	case reflect.Slice, reflect.Array:
		return convertSlice(val), nil
	case reflect.Map:
		return convertMap(val), nil
	case reflect.Ptr:
		if val.CanAddr() {
			val = val.Elem()
			return convert(val)
		}
		return "", ErrNotValidPtr
	default:
		return "", ErrNotSupportedType
	}
}

func getTarget(query string, value interface{}) (reflect.Value, error) {
	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return val, ErrNotValidValue
	}
	var err error
	val, err = stripPtr(val)
	if err != nil {
		return val, err
	}
	query = strings.Trim(query, ".")
	if query == "" {
		return val, nil
	}
	l := queryRe.FindAllStringSubmatch(query, -1)
	for _, q := range l {
		if seqRe.MatchString(q[1]) {
			index, err := strconv.ParseInt(q[1], 10, 64)
			if err != nil {
				return val, err
			}
			length, err := getLen(val)
			if err != nil {
				return val, err
			}
			index, err = fixIndex(index, length)
			if err != nil {
				return val, err
			}
			val, err = getTargetByNumIndex(index, val)
			if err != nil {
				return val, err
			}
		} else if mapIndexRe.MatchString(q[1]) {
			index := mapIndexRe.FindStringSubmatch(q[1])
			if val.Kind() != reflect.Map {
				return val, ErrNotMapVal
			}
			var err error
			val, err = getTargetByStrIndex(index[1], val)
			if err != nil {
				return val, err
			}
		} else {
			index := q[1]
			if val.Kind() != reflect.Struct {
				return val, ErrNotStructVal
			}
			var err error
			val, err = getTargetByStrIndex(index, val)
			if err != nil {
				return val, err
			}
		}
	}
	if !val.IsValid() {
		return val, ErrNotValidValue
	}
	return val, nil
}

func stripPtr(val reflect.Value) (reflect.Value, error) {
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if !val.IsValid() {
			return val, ErrNotValidPtr
		}
		val = val.Elem()
	}
	return val, nil
}

func getLen(val reflect.Value) (int64, error) {
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		return int64(val.Len()), nil
	case reflect.Struct:
		return int64(val.NumField()), nil
	default:
		return -1, ErrNotSeqVal
	}
}

func fixIndex(index, length int64) (int64, error) {
	if index < 0 {
		index = length + index
		if index < 0 {
			return -1, ErrNotEnoughValue
		}
	}
	if index >= length {
		return -1, ErrNotEnoughValue
	}
	return index, nil
}

func getTargetByNumIndex(index int64, val reflect.Value) (reflect.Value, error) {
	if !val.IsValid() {
		return val, ErrNotValidValue
	}
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		return stripPtr(val.Index(int(index)))
	case reflect.Struct:
		return stripPtr(val.Field(int(index)))
	default:
		return val, ErrNotSeqVal
	}
}

func getTargetByStrIndex(index string, val reflect.Value) (reflect.Value, error) {
	if !val.IsValid() {
		return val, ErrNotValidValue
	}
	switch val.Kind() {
	case reflect.Map:
		return stripPtr(val.MapIndex(reflect.ValueOf(index)))
	case reflect.Struct:
		return stripPtr(val.FieldByName(index))
	default:
		return val, ErrNotMapVal
	}
}

func Fmt(temp string, value interface{}) (string, error) {
	l := fmtRe.FindAllStringSubmatch(temp, -1)
	for _, t := range l {
		target, err := getTarget(t[1], value)
		if err != nil {
			return "", err
		}
		s, err := convert(target)
		if err != nil {
			return "", err
		}
		temp = strings.Replace(temp, t[0], s, -1)
	}
	return temp, nil
}
