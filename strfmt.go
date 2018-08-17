package nbfmt

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var ErrValueType = errors.New("strfmt: Not valid value type")
var ErrNotEnoughValue = errors.New("strfmt: Not enough values to format template")
var ErrNotSeqTemp = errors.New("strfmt: the templates in FmtBySeq() must be sequence template")
var ErrNotNameTemp = errors.New("strfmt: the templates in FmtByName() must be name template")

type identType int

const (
	name identType = iota
	seq
)

type varType int

const (
	_str varType = iota
	_int
	_float
	_bool
	_pointer
)

var fmtRe = regexp.MustCompile(`{{(.+?):([\.%\w]+?)}}`)
var seqRe = regexp.MustCompile(`\d+`)

func getIdentType(id string) identType {
	switch {
	case seqRe.MatchString(id):
		return seq
	default:
		return name
	}
}

func FmtBySeq(temp string, value interface{}) (string, error) {
	typ := reflect.TypeOf(value)
	val := reflect.ValueOf(value)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Struct {
		return "", ErrValueType
	}
	fmtList := fmtRe.FindAllStringSubmatch(temp, -1)
	var v string
	for _, f := range fmtList {
		if getIdentType(f[1]) != seq {
			return "", ErrNotSeqTemp
		}
		index, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil {
			return "", err
		}
		if index < 0 {
			index = int64(val.Len()) + index
		}
		if index < 0 {
			return "", ErrNotEnoughValue
		}
		if typ.Kind() == reflect.Slice {
			if index > int64(val.Len()) {
				return "", ErrNotEnoughValue
			}

			sliceVal := val.Index(int(index))
			if !sliceVal.CanInterface() {
				v = ""
			} else {
				for sliceVal.Kind() == reflect.Ptr {
					sliceVal = sliceVal.Elem()
				}
				if !sliceVal.IsValid() {
					v = ""
				} else {
					v = fmt.Sprintf(f[2], sliceVal.Interface())
				}
			}
		} else {
			if index > int64(val.NumField()) {
				return "", ErrNotEnoughValue
			}
			structVal := val.Field(int(index))
			if !structVal.CanInterface() {
				v = ""
			} else {
				for structVal.Kind() == reflect.Ptr {
					structVal = structVal.Elem()
				}
				if !structVal.IsValid() {
					v = ""
				} else {
					v = fmt.Sprintf(f[2], structVal.Interface())
				}
			}
		}
		temp = strings.Replace(temp, f[0], fmt.Sprintf(f[2], v), -1)
	}
	return temp, nil
}

func FmtByName(temp string, value interface{}) (string, error) {
	typ := reflect.TypeOf(value)
	val := reflect.ValueOf(value)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Map && typ.Kind() != reflect.Struct {
		return "", ErrValueType
	}
	fmtList := fmtRe.FindAllStringSubmatch(temp, -1)
	for _, f := range fmtList {
		if getIdentType(f[1]) != name {
			return "", ErrNotNameTemp
		}
		var v string
		if typ.Kind() == reflect.Map {
			mapVal := val.MapIndex(reflect.ValueOf(f[1]))
			if !mapVal.IsValid() {
				v = ""
			} else {
				for mapVal.Kind() == reflect.Ptr {
					mapVal = mapVal.Elem()
				}
				if !mapVal.IsValid() {
					v = ""
				} else {
					v = fmt.Sprintf(f[2], mapVal.Interface())
				}
			}
		} else {
			structVal := val.FieldByName(f[1])
			if !structVal.IsValid() {
				v = ""
			} else {
				for structVal.Kind() == reflect.Ptr {
					structVal = structVal.Elem()
				}
				if !structVal.IsValid() {
					v = ""
				} else {
					v = fmt.Sprintf(f[2], structVal.Interface())
				}
			}
		}
		temp = strings.Replace(temp, f[0], v, -1)
	}
	return temp, nil
}
