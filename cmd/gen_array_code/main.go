package main

import (
	"log"
	"os"
	"text/template"
)

const templateTxt = `
type {{ .TypeName }}Array struct {
	P unsafe.Pointer
	Len int
}

func New{{ .TypeName }}Array(values ...{{ .GoElemType }}) {{ .TypeName }}Array {
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

`

type params struct {
	TypeName   string
	GoElemType string
}

func main() {
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
	}
	for _, param := range arrParams {
		err = t1.Execute(os.Stdout, param)
		if err != nil {
			log.Fatal(err)
		}
	}
}
