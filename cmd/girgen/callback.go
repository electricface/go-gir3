package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/electricface/go-gir3/gi"
)

func pCallback(s *SourceFile, fi *gi.CallableInfo) {
	name := fi.Name()
	log.Println("callback", name)

	var paramNameTypes []string
	var cParamTypeNames []string
	var fields []string
	var fieldSetLines []string

	var varReg VarReg
	numArgs := fi.NumArg()
	foundUserData := false
	for i := 0; i < numArgs; i++ {
		argInfo := fi.Arg(i)
		argTypeInfo := argInfo.Type()

		paramName := varReg.regParam(i, argInfo.Name())
		dir := argInfo.Direction()
		switch dir {
		case gi.DIRECTION_IN:
			parseResult := parseCbArgTypeDirIn(paramName, argTypeInfo)
			paramNameTypes = append(paramNameTypes, paramName+" "+parseResult.cgoType)
			cParamTypeNames = append(cParamTypeNames, parseResult.cType+" "+paramName)

			if argInfo.Name() == "user_data" && parseResult.cgoType == "C.gpointer" {
				// is user_data param
				foundUserData = true
				continue
			}

			fieldName := "F_" + paramName
			fields = append(fields, fieldName+" "+parseResult.goType)

			fieldSetLine := fmt.Sprintf("%v: %v,", fieldName, parseResult.expr)
			fieldSetLines = append(fieldSetLines, fieldSetLine)

		case gi.DIRECTION_OUT:
		case gi.DIRECTION_INOUT:
		}

		argTypeInfo.Unref()
		argInfo.Unref()
	}

	if !foundUserData {
		s.GoBody.Pn("// ignore callback %v", name)
		return
	}

	s.GoBody.Pn("type %vStruct struct {", name)
	for _, field := range fields {
		s.GoBody.Pn(field)
	}
	s.GoBody.Pn("}")

	s.GoBody.Pn("//export my%v", name)
	s.GoBody.Pn("func my%v(%v) {", name, strings.Join(paramNameTypes, ", "))

	s.CBody.Pn("extern void my%s(%v);", name, strings.Join(cParamTypeNames, ", "))
	s.CBody.Pn("static void* getPointer_my%v() {", name)
	s.CBody.Pn("return (void*)(my%v);", name)
	s.CBody.Pn("}")

	varFn := varReg.alloc("fn")
	s.GoBody.Pn("%v := gi.GetFunc(uint(uintptr(user_data)))", varFn)
	varArgs := varReg.alloc("args")
	s.GoBody.Pn("%v := %vStruct{", varArgs, name)

	for _, line := range fieldSetLines {
		s.GoBody.Pn(line)
	}

	s.GoBody.Pn("}") // end struct
	s.GoBody.Pn("%v(%v)", varFn, varArgs)
	s.GoBody.Pn("}") // end func
}

type parseCbArgTypeDirInResult struct {
	cgoType string
	cType   string
	goType  string
	expr    string
}

func parseCbArgTypeDirIn(paramName string, argTypeInfo *gi.TypeInfo) *parseCbArgTypeDirInResult {
	tag := argTypeInfo.Tag()
	isPtr := argTypeInfo.IsPointer()

	cgoType := "C.gpointer"
	cType := "gpointer"
	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO tag: %v, isPtr: %v*/", tag, isPtr)
	expr := fmt.Sprintf("unsafe.Pointer(%v)", paramName)

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:

	case gi.TYPE_TAG_BOOLEAN,
		gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型

	case gi.TYPE_TAG_VOID:
		if isPtr {
			cgoType = "C.gpointer"
			cType = "gpointer"
			goType = "unsafe.Pointer"
			expr = fmt.Sprintf("unsafe.Pointer(%v)", paramName)
		}
	}
	return &parseCbArgTypeDirInResult{
		cType:   cType,
		cgoType: cgoType,
		goType:  goType,
		expr:    expr,
	}
}
