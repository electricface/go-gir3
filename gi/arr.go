package gi

import "unsafe"

type BoolArray struct {
	P unsafe.Pointer
	Len int
}

func NewBoolArray(values ...bool) BoolArray {
	size := int(unsafe.Sizeof(int32(0))) * len(values)
	p := Malloc(size)
	arr := BoolArray{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	for i, value := range values {
		if value {
			slice[i] = 1
		} else {
			slice[i] = 0
		}
	}
	return arr
}

func (arr BoolArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr BoolArray) AsSlice() []int32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]int32)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr BoolArray) Copy() []bool {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]bool, arr.Len)
	slice := (*(*[1 << 32]int32)(arr.P))[:arr.Len:arr.Len]
	for i, value := range slice {
		if value != 0 {
			result[i] = true
		}
	}
	return result
}

type CStrArray struct {
	P unsafe.Pointer
	Len int
}

func (arr CStrArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr CStrArray) AsSlice() []unsafe.Pointer {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr CStrArray) Copy() []string {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	var result []string
	slice := (*(*[1 << 32]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
	for _, value := range slice {
		if value == nil {
			break
		}
		result = append(result, GoString(value))
	}
	return result
}


// 以下代码是用 gen_array_code 工具自动生成的

type DoubleArray struct {
	P unsafe.Pointer
	Len int
}

func NewDoubleArray(values ...float64) DoubleArray {
	size := int(unsafe.Sizeof(float64(0))) * len(values)
	p := Malloc(size)
	arr := DoubleArray{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr DoubleArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr DoubleArray) AsSlice() []float64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]float64)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr DoubleArray) Copy() []float64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]float64, arr.Len)
	slice := (*(*[1 << 32]float64)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type FloatArray struct {
	P unsafe.Pointer
	Len int
}

func NewFloatArray(values ...float32) FloatArray {
	size := int(unsafe.Sizeof(float32(0))) * len(values)
	p := Malloc(size)
	arr := FloatArray{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr FloatArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr FloatArray) AsSlice() []float32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]float32)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr FloatArray) Copy() []float32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]float32, arr.Len)
	slice := (*(*[1 << 32]float32)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type UniCharArray struct {
	P unsafe.Pointer
	Len int
}

func NewUniCharArray(values ...rune) UniCharArray {
	size := int(unsafe.Sizeof(rune(0))) * len(values)
	p := Malloc(size)
	arr := UniCharArray{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr UniCharArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr UniCharArray) AsSlice() []rune {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]rune)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr UniCharArray) Copy() []rune {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]rune, arr.Len)
	slice := (*(*[1 << 32]rune)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Int8Array struct {
	P unsafe.Pointer
	Len int
}

func NewInt8Array(values ...int8) Int8Array {
	size := int(unsafe.Sizeof(int8(0))) * len(values)
	p := Malloc(size)
	arr := Int8Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Int8Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int8Array) AsSlice() []int8 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]int8)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Int8Array) Copy() []int8 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]int8, arr.Len)
	slice := (*(*[1 << 32]int8)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Uint8Array struct {
	P unsafe.Pointer
	Len int
}

func NewUint8Array(values ...uint8) Uint8Array {
	size := int(unsafe.Sizeof(uint8(0))) * len(values)
	p := Malloc(size)
	arr := Uint8Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Uint8Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint8Array) AsSlice() []uint8 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]uint8)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Uint8Array) Copy() []uint8 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]uint8, arr.Len)
	slice := (*(*[1 << 32]uint8)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Int16Array struct {
	P unsafe.Pointer
	Len int
}

func NewInt16Array(values ...int16) Int16Array {
	size := int(unsafe.Sizeof(int16(0))) * len(values)
	p := Malloc(size)
	arr := Int16Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Int16Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int16Array) AsSlice() []int16 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]int16)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Int16Array) Copy() []int16 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]int16, arr.Len)
	slice := (*(*[1 << 32]int16)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Uint16Array struct {
	P unsafe.Pointer
	Len int
}

func NewUint16Array(values ...uint16) Uint16Array {
	size := int(unsafe.Sizeof(uint16(0))) * len(values)
	p := Malloc(size)
	arr := Uint16Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Uint16Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint16Array) AsSlice() []uint16 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]uint16)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Uint16Array) Copy() []uint16 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]uint16, arr.Len)
	slice := (*(*[1 << 32]uint16)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Int32Array struct {
	P unsafe.Pointer
	Len int
}

func NewInt32Array(values ...int32) Int32Array {
	size := int(unsafe.Sizeof(int32(0))) * len(values)
	p := Malloc(size)
	arr := Int32Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Int32Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int32Array) AsSlice() []int32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]int32)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Int32Array) Copy() []int32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]int32, arr.Len)
	slice := (*(*[1 << 32]int32)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Uint32Array struct {
	P unsafe.Pointer
	Len int
}

func NewUint32Array(values ...uint32) Uint32Array {
	size := int(unsafe.Sizeof(uint32(0))) * len(values)
	p := Malloc(size)
	arr := Uint32Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Uint32Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint32Array) AsSlice() []uint32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]uint32)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Uint32Array) Copy() []uint32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]uint32, arr.Len)
	slice := (*(*[1 << 32]uint32)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Int64Array struct {
	P unsafe.Pointer
	Len int
}

func NewInt64Array(values ...int64) Int64Array {
	size := int(unsafe.Sizeof(int64(0))) * len(values)
	p := Malloc(size)
	arr := Int64Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Int64Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int64Array) AsSlice() []int64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]int64)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Int64Array) Copy() []int64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]int64, arr.Len)
	slice := (*(*[1 << 32]int64)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}


type Uint64Array struct {
	P unsafe.Pointer
	Len int
}

func NewUint64Array(values ...uint64) Uint64Array {
	size := int(unsafe.Sizeof(uint64(0))) * len(values)
	p := Malloc(size)
	arr := Uint64Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr Uint64Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint64Array) AsSlice() []uint64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[1 << 31]uint64)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr Uint64Array) Copy() []uint64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]uint64, arr.Len)
	slice := (*(*[1 << 32]uint64)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

