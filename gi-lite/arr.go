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

const arrLenMax = 1 << 31

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

// strArr 的 array
type CStrvArray struct {
	P   unsafe.Pointer
	Len int
}

func (arr *CStrvArray) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr *CStrvArray) FreeAll() {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	slice := (*(*[arrLenMax]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
	for i := 0; i < arr.Len; i++ {
		strArr := CStrArray{P: slice[i]}
		strArr.SetLenZT()
		strArr.FreeAll()
	}
	Free(arr.P)
	arr.P = nil
}

func (arr *CStrvArray) SetLenZT() {
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

func (arr CStrvArray) Copy() (result [][]string) {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}

	slice := (*(*[arrLenMax]unsafe.Pointer)(arr.P))[:arr.Len:arr.Len]
	for _, value := range slice {
		strArr := CStrArray{P: value}
		strArr.SetLenZT()
		strSlice := strArr.Copy()
		result = append(result, strSlice)
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

func (arr *Uint8Array) SetLenZT() {
	slice := (*(*[arrLenMax]uint8)(arr.P))[:arrLenMax:arrLenMax]
	for i, value := range slice {
		if value == 0 {
			// 0 1 2
			// 1 2 0
			// 比如长度为3 的数组，最后一个是零值，实际是2个元素，在 value == 0 时，i 是 2, 所以 arr.Len = i
			arr.Len = i
			break
		}
	}
}
