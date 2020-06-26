/*
 * Copyright (C) 2019 ~ 2020 Uniontech Software Technology Co.,Ltd
 *
 * Author:
 *
 * Maintainer:
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package gi

import "unsafe"

const arrLenMax = 1 << 32

type BoolArray struct {
	P   unsafe.Pointer
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

func (arr *BoolArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr BoolArray) AsSlice() []int32 {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
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
	P   unsafe.Pointer
	Len int
}

// free container
func (arr *CStrArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr *CStrArray) FreeAll() {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[arrLenMax]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
	for i := 0; i < arr.Len; i++ {
		Free(slice[i])
	}
	Free(arr.P)
	arr.P = nil
}

const NilStr = "\x00\x00\x00\x00"

func NewCStrArrayWithStrings(values ...string) CStrArray {
	size := int(unsafe.Sizeof(uintptr(0))) * len(values)
	p := Malloc(size)
	arr := CStrArray{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	for i, value := range values {
		slice[i] = CString(value)
	}

	return arr
}

func NewCStrArrayZTWithStrings(values ...string) CStrArray {
	size := int(unsafe.Sizeof(uintptr(0))) * (len(values) + 1)
	p := Malloc(size)
	arr := CStrArray{
		P:   p,
		Len: len(values) + 1,
	}
	slice := arr.AsSlice()
	for i, value := range values {
		slice[i] = CString(value)
	}
	slice[len(values)] = nil
	arr.Len--
	return arr
}

func (arr *CStrArray) SetLenZT() {
	//noinspection GoInvalidIndexOrSliceExpression
	slice := (*(*[arrLenMax]unsafe.Pointer)(arr.P))[:arrLenMax:arrLenMax]
	for i, value := range slice {
		if value == nil {
			// 0 1 2
			// p p nil
			// 比如长度为3 的数组，最后一个是零值，实际是2个元素，在 value == nil 时，i 是 2, 所以 arr.Len = i
			arr.Len = i
			break
		}
	}
}

func (arr CStrArray) AsSlice() []unsafe.Pointer {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
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
	slice := (*(*[arrLenMax]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
	for _, value := range slice {
		result = append(result, GoString(value))
	}
	return result
}

type PointerArray struct {
	P   unsafe.Pointer
	Len int
}

func (arr *PointerArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr *PointerArray) SetLenZT() {
	(*CStrArray)(arr).SetLenZT()
}

func (arr PointerArray) AsSlice() []unsafe.Pointer {
	return CStrArray(arr).AsSlice()
}

func (arr PointerArray) Copy() []unsafe.Pointer {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]unsafe.Pointer, arr.Len)
	slice := (*(*[arrLenMax]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

func NewPointerArray(values ...unsafe.Pointer) PointerArray {
	size := int(unsafe.Sizeof(uintptr(0))) * len(values)
	p := Malloc(size)
	arr := PointerArray{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

// 以下代码是用 gen_array_code 工具自动生成的

type DoubleArray struct {
	P   unsafe.Pointer
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

func NewGTypeArray(values ...GType) GTypeArray {
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
