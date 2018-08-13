package zlog

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"time"
)

type fieldType int

const (
	unknownType fieldType = iota
	boolType
	floatType
	intType
	int64Type
	uintType
	uint64Type
	uintptrType
	stringType
	objectType
	stringerType
)

type Field struct {
	key       string
	fieldType fieldType
	ival      int64
	str       string
	obj       interface{}
}

func (f *Field) WriteValue(b []byte) []byte {
	switch f.fieldType {
	case boolType:
		return strconv.AppendBool(b, f.ival == 1)
	case stringType:
		return append(b, f.str...)
	case intType:
		return strconv.AppendInt(b, int64(f.ival), 10)
	case int64Type:
		return strconv.AppendInt(b, f.ival, 10)
	case floatType:
		return strconv.AppendFloat(b, math.Float64frombits(uint64(f.ival)), 'f', -1, 64)
	case uintType:
		return strconv.AppendUint(b, uint64(f.ival), 10)
	case uint64Type:
		return strconv.AppendUint(b, uint64(f.ival), 10)
	case objectType:
		return append(b, fmt.Sprintf("%+v", f.obj)...)
	case stringerType:
		return append(b, f.obj.(fmt.Stringer).String()...)
	case uintptrType:
		b = append(b, "0x"...)
		return strconv.AppendUint(b, uint64(f.ival), 16)
	default:

	}
	return nil
}

// Base64 constructs a field that encodes the given value as a padded base64
// string. The byte slice is converted to a base64 string eagerly.
func Base64(key string, val []byte) Field {
	return String(key, base64.StdEncoding.EncodeToString(val))
}

// Bool constructs a Field with the given key and value. Bools are marshaled
// lazily.
func Bool(key string, val bool) Field {
	var ival int64
	if val {
		ival = 1
	}

	return Field{key: key, fieldType: boolType, ival: ival}
}

// Float64 constructs a Field with the given key and value. The way the
// floating-point value is represented is encoder-dependent, so marshaling is
// necessarily lazy.
func Float64(key string, val float64) Field {
	return Field{key: key, fieldType: floatType, ival: int64(math.Float64bits(val))}
}

// Int constructs a Field with the given key and value. Marshaling ints is lazy.
func Int(key string, val int) Field {
	return Field{key: key, fieldType: intType, ival: int64(val)}
}

// Int64 constructs a Field with the given key and value. Like ints, int64s are
// marshaled lazily.
func Int64(key string, val int64) Field {
	return Field{key: key, fieldType: int64Type, ival: val}
}

// Uint constructs a Field with the given key and value.
func Uint(key string, val uint) Field {
	return Field{key: key, fieldType: uintType, ival: int64(val)}
}

// Uint64 constructs a Field with the given key and value.
func Uint64(key string, val uint64) Field {
	return Field{key: key, fieldType: uint64Type, ival: int64(val)}
}

// Uintptr constructs a Field with the given key and value.
func Uintptr(key string, val uintptr) Field {
	return Field{key: key, fieldType: uintptrType, ival: int64(val)}
}

// String constructs a Field with the given key and value.
func String(key string, val string) Field {
	return Field{key: key, fieldType: stringType, str: val}
}

// Stringer constructs a Field with the given key and the output of the value's
// String method. The Stringer's String method is called lazily.
func Stringer(key string, val fmt.Stringer) Field {
	return Field{key: key, fieldType: stringerType, obj: val}
}

// Duration constructs a Field with the given key and value. It represents
// durations as an integer number of nanoseconds.
func Duration(key string, val time.Duration) Field {
	return Int64(key, int64(val))
}

// Object constructs a field with the given key and an arbitrary object. It uses
// an encoding-appropriate, reflection-based function to lazily serialize nearly
// any object into the logging context, but it's relatively slow and
// allocation-heavy.
//
// If encoding fails (e.g., trying to serialize a map[int]string to JSON), Object
// includes the error message in the final log output.
func Object(key string, val interface{}) Field {
	return Field{key: key, fieldType: objectType, obj: val}
}
