package g

import (
	"errors"
	"reflect"
	"unsafe"

	"github.com/linuxdeepin/go-gir/gi"
)

/*
 * GValue
 */

var errNoMem = errors.New("cannot alloc memory")

func NewValue() (Value, error) {
	p := gi.Malloc0(SizeOfStructValue)
	if p == nil {
		return Value{}, errNoMem
	}
	return Value{P: p}, nil
}

func NewValueT(gType gi.GType) (Value, error) {
	v, err := NewValue()
	if err != nil {
		return Value{}, err
	}
	v.Init(gType)
	return v, nil
}

func NewValueWith(iVal interface{}) (Value, error) {
	v, err := NewValue()
	if err != nil {
		return Value{}, err
	}
	type0, err := getValueType(iVal)
	if err != nil {
		return Value{}, err
	}
	v.Init(type0)
	err = v.Set(iVal)
	if err != nil {
		return Value{}, err
	}
	return v, nil
}

func (v *Value) Free() {
	gi.Free(v.P)
	v.P = nil
}

//var gvalueGetters = struct {
//	m map[gi.GType]GValueGetter
//	sync.Mutex
//}{
//	m: make(map[gi.GType]GValueGetter),
//}
//
//type GValueGetter func(unsafe.Pointer) (interface{}, error)

//func registerGValueGetter(typ gi.GType, getter GValueGetter) {
//	gvalueGetters.Lock()
//	gvalueGetters.m[typ] = getter
//	gvalueGetters.Unlock()
//}


func (v Value) Get() (interface{}, error) {
	actualType := v.Type()
	//fmt.Println("actual Type:", actualType)
	//fmt.Println("fund Type:", fundamentalType)

	val, err := v.get(actualType)
	if err == nil {
		return val, nil
	} else if err == errTypeUnknown {
		fundamentalType := TypeFundamental(actualType)
		// fallback to fundamental type
		return v.get(fundamentalType)
	}
	return nil, err
}

func (v Value) Store(dest interface{}) error {
	src, err := v.Get()
	if err != nil {
		return err
	}

	// 有限制，如果 src 是 g.Object ，则要求 dest 必须也是 GObject，要实现 IObject 接口。
	if _, ok := src.(Object); ok {
		// is obj
		if _, isIObj := dest.(IObject); !isIObj {
			return errors.New("dest is not object")
		}
	}

	return gi.StoreInterfaces(src, dest)
}

var errTypeUnknown = errors.New("unknown type")

func (v Value) get(typ gi.GType) (ret interface{}, err error) {
	switch typ {
	case TYPE_INVALID:
		err = errors.New("invalid type")
	case TYPE_NONE:
	case TYPE_INTERFACE:
		err = errors.New("interface conversion not yet implemented")
	case TYPE_CHAR:
		ret = v.GetSchar()
	case TYPE_UCHAR:
		ret = v.GetUchar()
	case TYPE_BOOLEAN:
		ret = v.GetBoolean()
	case TYPE_INT:
		ret = v.GetInt()

	case TYPE_UINT:
		ret = v.GetUint()

	case TYPE_LONG:
		ret = v.GetLong()

	case TYPE_ULONG:
		ret = v.GetUlong()

	case TYPE_INT64:
		ret = v.GetInt64()

	case TYPE_UINT64:
		ret = v.GetUint64()

	case TYPE_ENUM:
		ret = v.GetEnum()

	case TYPE_FLAGS:
		ret = v.GetFlags()

	case TYPE_FLOAT:
		ret = v.GetFloat()

	case TYPE_DOUBLE:
		ret = v.GetDouble()

	case TYPE_STRING:
		ret = v.GetString()

	case TYPE_POINTER:
		ret = v.GetPointer()

	case TYPE_BOXED:
		ret = v.GetBoxed()

	case TYPE_PARAM:
		ret = v.GetParam()

	case TYPE_OBJECT:
		ret = v.GetObject()

	case TYPE_VARIANT:
		ret = v.GetVariant()

	default:
		err = errTypeUnknown
	}

	//gvalueGetters.Lock()
	//getter, ok := gvalueGetters.m[typ]
	//gvalueGetters.Unlock()
	//if !ok {
	//	return nil, errTypeUnknown
	//}
	//return getter(v.P)
	return
}

//func (v Value) GetWithType(reflectType reflect.Type) (interface{}, error) {
//	kind := reflectType.Kind()
//	switch kind {
//	case reflect.Bool:
//		val := v.GetBoolean()
//		return val, nil
//
//	case reflect.Int:
//		_, fundType, err := v.Type()
//		if err != nil {
//			return nil, err
//		}
//
//		switch fundType {
//		case TYPE_ENUM:
//			val := v.GetEnum()
//			return val, nil
//
//		default:
//			val := v.GetInt()
//			return val, nil
//		}
//
//	case reflect.Int8:
//		val := v.GetSchar()
//		return val, nil
//
//	case reflect.Int16:
//		val := v.GetInt()
//		return int16(val), nil
//
//	case reflect.Int32:
//		val := v.GetInt()
//		return int32(val), nil
//
//	case reflect.Int64:
//		val := v.GetInt64()
//		return val, nil
//
//	case reflect.Uint:
//		val := v.GetUint()
//		return val, nil
//
//	case reflect.Uint8:
//		val := v.GetUchar()
//		return val, nil
//
//	case reflect.Uint16:
//		val := v.GetUint()
//		return uint16(val), nil
//
//	case reflect.Uint32:
//		val := v.GetUint()
//		return uint32(val), nil
//
//	case reflect.Uint64:
//		val := v.GetUint64()
//		return val, nil
//
//	case reflect.Uintptr:
//		val := v.GetPointer()
//		return uintptr(val), nil
//
//	case reflect.Float32:
//		val := v.GetFloat()
//		return val, nil
//
//	case reflect.Float64:
//		val := v.GetDouble()
//		return val, nil
//
//	case reflect.UnsafePointer:
//		val := v.GetPointer()
//		return val, nil
//
//	case reflect.String:
//		val := v.GetString()
//		return val, nil
//
//	case reflect.Struct:
//		val := unsafe.Pointer(C.g_value_get_object(v.native()))
//
//		newValPtr := reflect.New(reflectType)
//		newVal := newValPtr.Elem()
//		ptrFieldVal := newVal.FieldByName("P")
//		ptrFieldVal.SetPointer(val)
//
//		return newVal.Interface(), nil
//
//	default:
//		// Complex64, Complex128
//		// Array
//		// Chan
//		// Func
//		// Interface
//		// Map
//		// Ptr
//		// Slice
//		return nil, errors.New("unsupported reflect type")
//	}
//}

var errTypeConvert = errors.New("type convert failed")

func getValueType(iVal interface{}) (gType gi.GType, err error) {
	switch iVal.(type) {
	case int8:
		gType = TYPE_CHAR
	case uint8:
		gType = TYPE_UCHAR
	case bool:
		gType = TYPE_BOOLEAN
	case int:
		gType = TYPE_INT
	case uint:
		gType = TYPE_UINT
	case int32:
		gType = TYPE_INT
	case uint32:
		gType = TYPE_UINT
	case gi.Long:
		gType = TYPE_LONG
	case gi.Ulong:
		gType = TYPE_ULONG
	case int64:
		gType = TYPE_INT64
	case uint64:
		gType = TYPE_UINT64
	case gi.Enum:
		gType = TYPE_ENUM
	case gi.Flags:
		gType = TYPE_FLAGS
	case float32:
		gType = TYPE_FLOAT
	case float64:
		gType = TYPE_DOUBLE
	case string:
		gType = TYPE_STRING
	case unsafe.Pointer:
		gType = TYPE_POINTER
	case ParamSpec:
		gType = TYPE_PARAM
	case Object:
		gType = TYPE_OBJECT
	case Variant:
		gType = TYPE_VARIANT
	default:
		err = errors.New("unsupported type")
	}
	return
}

func (v Value) Set(iVal interface{}) error {
	gType := v.Type()
	switch gType {
	case TYPE_INVALID:
		return errors.New("type is invalid")

	case TYPE_NONE:
		return nil

	case TYPE_INTERFACE:
		return errors.New("unsupported type interface")

	case TYPE_CHAR:
		val, ok := iVal.(int8)
		if !ok {
			return errTypeConvert
		}
		v.SetSchar(val)

	case TYPE_UCHAR:
		val, ok := iVal.(uint8)
		if !ok {
			return errTypeConvert
		}
		v.SetUchar(val)

	case TYPE_BOOLEAN:
		val, ok := iVal.(bool)
		if !ok {
			return errTypeConvert
		}
		v.SetBoolean(val)

	case TYPE_INT:
		rv := reflect.ValueOf(iVal)
		if rv.Type().ConvertibleTo(gi.TypeInt) {
			val := rv.Int()
			v.SetInt(int32(val))
		} else {
			return errTypeConvert
		}

	case TYPE_UINT:
		rv := reflect.ValueOf(iVal)
		if rv.Type().ConvertibleTo(gi.TypeUint) {
			val := rv.Uint()
			v.SetUint(uint32(val))
		} else {
			return errTypeConvert
		}

	case TYPE_LONG:
		rv := reflect.ValueOf(iVal)
		if rv.Type().ConvertibleTo(gi.TypeInt) {
			val := rv.Int()
			v.SetLong(val)
		} else {
			return errTypeConvert
		}

	case TYPE_ULONG:
		rv := reflect.ValueOf(iVal)
		if rv.Type().ConvertibleTo(gi.TypeUint) {
			val := rv.Uint()
			v.SetUlong(val)
		} else {
			return errTypeConvert
		}

	case TYPE_INT64:
		val, ok := iVal.(int64)
		if !ok {
			return errTypeConvert
		}
		v.SetInt64(val)

	case TYPE_UINT64:
		val, ok := iVal.(uint64)
		if !ok {
			return errTypeConvert
		}
		v.SetUint64(val)

	case TYPE_ENUM:
		val, ok := iVal.(int32)
		if !ok {
			return errTypeConvert
		}
		v.SetEnum(val)

	case TYPE_FLAGS:
		val, ok := iVal.(uint32)
		if !ok {
			return errTypeConvert
		}
		v.SetFlags(val)

	case TYPE_FLOAT:
		val, ok := iVal.(float32)
		if !ok {
			return errTypeConvert
		}
		v.SetFloat(val)

	case TYPE_DOUBLE:
		val, ok := iVal.(float64)
		if !ok {
			return errTypeConvert
		}
		v.SetDouble(val)

	case TYPE_STRING:
		val, ok := iVal.(string)
		if !ok {
			return errTypeConvert
		}
		v.SetString(val)

	case TYPE_POINTER:
		val, ok := iVal.(unsafe.Pointer)
		if ok {
			return errTypeConvert
		}
		v.SetPointer(val)

	case TYPE_BOXED:
		val, ok := iVal.(unsafe.Pointer)
		if !ok {
			return errTypeConvert
		}
		v.SetBoxed(val)

	case TYPE_PARAM:
		val, ok := iVal.(ParamSpec)
		if !ok {
			return errTypeConvert
		}
		v.SetParam(val)

	case TYPE_OBJECT:
		val, ok := iVal.(Object)
		if !ok {
			return errTypeConvert
		}
		v.SetObject(val)

	case TYPE_VARIANT:
		val, ok := iVal.(Variant)
		if !ok {
			return errTypeConvert
		}
		v.SetVariant(val)

	default:
		return errTypeConvert
	}

	return nil
}
