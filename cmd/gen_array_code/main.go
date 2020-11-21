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

package main

import (
	"fmt"
	"log"
	"os"
	"text/template"
)

const templateTxt = `
type {{ .TypeName }}Array struct {
	P unsafe.Pointer
	Len int
}

func Make{{ .TypeName }}Array(length int) {{ .TypeName }}Array {
	size := int(unsafe.Sizeof({{ .GoElemType }}(0))) * length
	p := Malloc0(size)
	arr := {{ .TypeName }}Array{
		P: p,
		Len: length,
	}
	return arr
}

func New{{ .TypeName }}Array(values []{{ .GoElemType }}) {{ .TypeName }}Array {
	size := int(unsafe.Sizeof({{ .GoElemType }}(0))) * len(values)
	p := Malloc(size)
	arr := {{ .TypeName }}Array{
		P:   p,
		Len: len(values),
	}
	slice := arr.AsSlice()
	copy(slice, values)
	return arr
}

func (arr *{{ .TypeName }}Array) Free() {
	Free(arr.P)
	arr.P = nil
}

func (arr {{ .TypeName }}Array) AsSlice() []{{ .GoElemType }} {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	slice := (*(*[arrLenMax]{{ .GoElemType }})(arr.P))[:arr.Len:arr.Len]
	return slice
}

func (arr {{ .TypeName }}Array) Copy() []{{ .GoElemType }} {
	if arr.Len < 0 {
		panic("arr.len < 0")
	}
	if arr.Len == 0 {
		return nil
	}
	result := make([]{{ .GoElemType }}, arr.Len)
	slice := (*(*[arrLenMax]{{ .GoElemType }})(arr.P))[:arr.Len:arr.Len]
	copy(result, slice)
	return result
}

func (arr *{{ .TypeName }}Array) SetLenZT() {
	slice := (*(*[arrLenMax]{{ .GoElemType }})(arr.P))[:arrLenMax:arrLenMax]
	for i, value := range slice {
		if value == 0 {
			arr.Len = i
			break
		}
	}
}
`

type params struct {
	TypeName   string
	GoElemType string
}

func main() {
	// print header
	fmt.Println(`// 本文件是用 gen_array_code 工具自动生成的
package gi

import "unsafe"
`)

	t1 := template.New("t1")
	_, err := t1.Parse(templateTxt)
	if err != nil {
		log.Fatal(err)
	}
	arrParams := []params{
		{TypeName: "Double", GoElemType: "float64"},
		{TypeName: "Float", GoElemType: "float32"},
		{TypeName: "UniChar", GoElemType: "rune"},
		{TypeName: "Int8", GoElemType: "int8"},
		{TypeName: "Uint8", GoElemType: "uint8"},
		{TypeName: "Int16", GoElemType: "int16"},
		{TypeName: "Uint16", GoElemType: "uint16"},
		{TypeName: "Int32", GoElemType: "int32"},
		{TypeName: "Uint32", GoElemType: "uint32"},
		{TypeName: "Int64", GoElemType: "int64"},
		{TypeName: "Uint64", GoElemType: "uint64"},
		{TypeName: "GType", GoElemType: "GType"},
	}
	for _, param := range arrParams {
		err = t1.Execute(os.Stdout, param)
		if err != nil {
			log.Fatal(err)
		}
	}
}
