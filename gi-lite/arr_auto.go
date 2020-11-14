// 本文件是用 gen_array_code 工具自动生成的
package gi

import "unsafe"

type DoubleArray struct {
	P   unsafe.Pointer
	Len int
}

func MakeDoubleArray(length int) DoubleArray {
	size := int(unsafe.Sizeof(float64(0))) * length
	p := Malloc0(size)
	arr := DoubleArray{
		P:   p,
		Len: length,
	}
	return arr
}

func NewDoubleArray(values []float64) DoubleArray {
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

func (arr *DoubleArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr DoubleArray) AsSlice() []float64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]float64)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]float64)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type FloatArray struct {
	P   unsafe.Pointer
	Len int
}

func MakeFloatArray(length int) FloatArray {
	size := int(unsafe.Sizeof(float32(0))) * length
	p := Malloc0(size)
	arr := FloatArray{
		P:   p,
		Len: length,
	}
	return arr
}

func NewFloatArray(values []float32) FloatArray {
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

func (arr *FloatArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr FloatArray) AsSlice() []float32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]float32)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]float32)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type UniCharArray struct {
	P   unsafe.Pointer
	Len int
}

func MakeUniCharArray(length int) UniCharArray {
	size := int(unsafe.Sizeof(rune(0))) * length
	p := Malloc0(size)
	arr := UniCharArray{
		P:   p,
		Len: length,
	}
	return arr
}

func NewUniCharArray(values []rune) UniCharArray {
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

func (arr *UniCharArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr UniCharArray) AsSlice() []rune {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]rune)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]rune)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Int8Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeInt8Array(length int) Int8Array {
	size := int(unsafe.Sizeof(int8(0))) * length
	p := Malloc0(size)
	arr := Int8Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewInt8Array(values []int8) Int8Array {
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

func (arr *Int8Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int8Array) AsSlice() []int8 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]int8)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]int8)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Uint8Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeUint8Array(length int) Uint8Array {
	size := int(unsafe.Sizeof(uint8(0))) * length
	p := Malloc0(size)
	arr := Uint8Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewUint8Array(values []uint8) Uint8Array {
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

func (arr *Uint8Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint8Array) AsSlice() []uint8 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]uint8)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]uint8)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Int16Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeInt16Array(length int) Int16Array {
	size := int(unsafe.Sizeof(int16(0))) * length
	p := Malloc0(size)
	arr := Int16Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewInt16Array(values []int16) Int16Array {
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

func (arr *Int16Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int16Array) AsSlice() []int16 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]int16)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]int16)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Uint16Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeUint16Array(length int) Uint16Array {
	size := int(unsafe.Sizeof(uint16(0))) * length
	p := Malloc0(size)
	arr := Uint16Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewUint16Array(values []uint16) Uint16Array {
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

func (arr *Uint16Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint16Array) AsSlice() []uint16 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]uint16)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]uint16)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Int32Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeInt32Array(length int) Int32Array {
	size := int(unsafe.Sizeof(int32(0))) * length
	p := Malloc0(size)
	arr := Int32Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewInt32Array(values []int32) Int32Array {
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

func (arr *Int32Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int32Array) AsSlice() []int32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]int32)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]int32)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Uint32Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeUint32Array(length int) Uint32Array {
	size := int(unsafe.Sizeof(uint32(0))) * length
	p := Malloc0(size)
	arr := Uint32Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewUint32Array(values []uint32) Uint32Array {
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

func (arr *Uint32Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint32Array) AsSlice() []uint32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]uint32)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]uint32)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Int64Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeInt64Array(length int) Int64Array {
	size := int(unsafe.Sizeof(int64(0))) * length
	p := Malloc0(size)
	arr := Int64Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewInt64Array(values []int64) Int64Array {
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

func (arr *Int64Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Int64Array) AsSlice() []int64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]int64)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]int64)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type Uint64Array struct {
	P   unsafe.Pointer
	Len int
}

func MakeUint64Array(length int) Uint64Array {
	size := int(unsafe.Sizeof(uint64(0))) * length
	p := Malloc0(size)
	arr := Uint64Array{
		P:   p,
		Len: length,
	}
	return arr
}

func NewUint64Array(values []uint64) Uint64Array {
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

func (arr *Uint64Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr Uint64Array) AsSlice() []uint64 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]uint64)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]uint64)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

type GTypeArray struct {
	P   unsafe.Pointer
	Len int
}

func MakeGTypeArray(length int) GTypeArray {
	size := int(unsafe.Sizeof(GType(0))) * length
	p := Malloc0(size)
	arr := GTypeArray{
		P:   p,
		Len: length,
	}
	return arr
}

func NewGTypeArray(values []GType) GTypeArray {
	size := int(unsafe.Sizeof(GType(0))) * len(values)
	p := Malloc(size)
	arr := GTypeArray{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr *GTypeArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr GTypeArray) AsSlice() []GType {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]GType)(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr GTypeArray) Copy() []GType {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]GType, arr.Len)
	slice := (*(*[arrLenMax]GType)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}
