package nbfmt

import (
	"fmt"
	"reflect"
)

//MapKeyTypeError will be return when query type is not equal to map key type
type MapKeyTypeError struct {
	requiredType reflect.Type
	providedType reflect.Type
}

func (e MapKeyTypeError) Error() string {
	return fmt.Sprintf("nbfmt: map key type error (require %s, supplied %s)", e.requiredType.Name(), e.providedType.Name())
}

//InvalidValueError will be return when object value is not valid
type InvalidValueError struct {
	val reflect.Value
}

func (e InvalidValueError) Error() string {
	return fmt.Sprintf("nbfmt: invalid value error (%s is not valid value)", e.val.Type())
}

//NotSupportedTypeError will be return when object type is not supported
type NotSupportedTypeError struct {
	val reflect.Value
}

func (e NotSupportedTypeError) Error() string {
	return fmt.Sprintf("nbfmt: not supported type %s", e.val.Type())
}

//InvalidQueryError will be return when query is not valid
type InvalidQueryError struct {
	query string
	q     string
}

func (e InvalidQueryError) Error() string {
	return fmt.Sprintf("nbfmt: invalid sub query %s in query %s, ", e.q, e.query)
}

//InvalidSeqQueryError will be return when the type of query for seqence object is not int
type InvalidSeqQueryError struct {
	q interface{}
}

func (e InvalidSeqQueryError) Error() string {
	return fmt.Sprintf("nbfmt: invalid sequence query %v", e.q)
}

//NotSeqTypeError will be return when object type is not sequence type (array, slice)
type NotSeqTypeError struct {
	val reflect.Value
}

func (e NotSeqTypeError) Error() string {
	return fmt.Sprintf("nbfmt: %s is not a sequence type", e.val.Type())
}

//IndexOutRangeError will be returned when query index is out of range of sequence object or struct
type IndexOutRangeError struct {
	index  int
	length int
}

func (e IndexOutRangeError) Error() string {
	return fmt.Sprintf("nbfmt: index %d is out of range (length: %d)", e.index, e.length)
}

//InvalidPtrError will be return when object pointer cannot be referecned
type InvalidPtrError struct {
	val reflect.Value
}

func (e InvalidPtrError) Error() string {
	return fmt.Sprintf("nbfmt: *%s is not valid ptr", e.val.Type())
}

//InvalidStructFieldQueryError will be return when the type of struct field query is not int or string
type InvalidStructFieldQueryError struct {
	q interface{}
}

func (e InvalidStructFieldQueryError) Error() string {
	return fmt.Sprintf("nbfmt: %v is not valid struct field query(only string and int is supported)", e.q)
}
