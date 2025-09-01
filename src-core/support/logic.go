package support

import (
	"reflect"
)

// copy from  condition.TernaryOperator
func If[T, U any](isTrue T, ifValue U, elseValue U) U {
	if Bool(isTrue) {
		return ifValue
	} else {
		return elseValue
	}
}

func Or[T any](ifValue T, elseValue T) T {
	if Bool(ifValue) {
		return ifValue
	} else {
		return elseValue
	}
}

func Bool[T any](value T) bool {
	switch m := any(value).(type) {
	case interface{ Bool() bool }:
		return m.Bool()
	case interface{ IsZero() bool }:
		return !m.IsZero()
	}
	return reflectValue(&value)
}

func reflectValue(vp any) bool {
	rv := reflect.ValueOf(vp).Elem()
	switch rv.Kind() {
	case reflect.Map, reflect.Slice:
		return rv.Len() != 0
	default:
		is := rv.IsZero()
		return !is
	}
}
