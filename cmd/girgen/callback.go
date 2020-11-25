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
	"strings"

	"github.com/electricface/go-gir3/gi"
)

func pCallback(s *SourceFile, fi *gi.CallableInfo) {
	// DestroyNotify
	pCallbackFuncDefine(s.GoBody, fi)

	// CallDestroyNotify
	// TODO
}

func pCallbackFuncDefine(b *SourceBody, fi *gi.CallableInfo) {
	name := fi.Name()

	var paramNameTypes []string
	var varReg VarReg
	numArgs := fi.NumArg()
	for i := 0; i < numArgs; i++ {
		argInfo := fi.Arg(i)
		argTypeInfo := argInfo.Type()

		paramName := varReg.registerParam(i, argInfo.Name())
		dir := argInfo.Direction()
		switch dir {
		case gi.DIRECTION_IN:
			result := parseCbArgTypeDirIn(paramName, argTypeInfo)
			paramNameTypes = append(paramNameTypes, paramName+" "+result.goType)
		case gi.DIRECTION_OUT:
			// TODO
		case gi.DIRECTION_INOUT:
			// TODO
		}
	}

	args := strings.Join(paramNameTypes, ", ")
	result := ""
	b.Pn("type %v func(%v) %v", name, args, result)
}

type parseCbArgTypeDirInResult struct {
	goType string
}

func parseCbArgTypeDirIn(paramName string, argTypeInfo *gi.TypeInfo) *parseCbArgTypeDirInResult {
	tag := argTypeInfo.Tag()
	isPtr := argTypeInfo.IsPointer()

	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO_CB tag: %v, isPtr: %v*/", tag, isPtr)

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		if isPtr {
			goType = "string"
		}

	case gi.TYPE_TAG_BOOLEAN,
		gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型
		if !isPtr {
			goType = getTypeWithTag(tag)
		} else {
			goType = "*" + getTypeWithTag(tag)
		}

	case gi.TYPE_TAG_UNICHAR:
		if !isPtr {
			goType = "rune"
		}

	case gi.TYPE_TAG_VOID:
		if isPtr {
			goType = "unsafe.Pointer"
		}

		// TODO
	}

	return &parseCbArgTypeDirInResult{
		goType: goType,
	}
}

func pCallback1(s *SourceFile, fi *gi.CallableInfo) {
	name := fi.Name()

	var paramNameTypes []string
	var cParamTypeNames []string
	var fields []string
	var fieldSetLines []string
	var handleArgs []string

	var varReg VarReg
	numArgs := fi.NumArg()
	foundUserData := false
	for i := 0; i < numArgs; i++ {
		argInfo := fi.Arg(i)
		argTypeInfo := argInfo.Type()

		paramName := varReg.registerParam(i, argInfo.Name())
		fieldName := snake2Camel(argInfo.Name())
		dir := argInfo.Direction()
		switch dir {
		case gi.DIRECTION_IN:
			parseResult := parseCbArgTypeDirIn1(paramName, argTypeInfo)
			paramNameTypes = append(paramNameTypes, paramName+" "+parseResult.cgoType)
			cParamTypeNames = append(cParamTypeNames, parseResult.cType+" "+paramName)

			if argInfo.Name() == "user_data" && parseResult.cgoType == "C.gpointer" {
				// is user_data param
				foundUserData = true
				continue
			}

			fields = append(fields, fieldName+" "+parseResult.goType)

			fieldSetLine := fmt.Sprintf("%v: %v,", fieldName, parseResult.expr)
			fieldSetLines = append(fieldSetLines, fieldSetLine)
			handleArgs = append(handleArgs, parseResult.expr)

		case gi.DIRECTION_OUT:
			parseResult := parseCbArgTypeDirOut1(paramName, argTypeInfo)
			paramNameTypes = append(paramNameTypes, paramName+" "+parseResult.cgoType)
			cParamTypeNames = append(cParamTypeNames, parseResult.cType+" "+paramName)

			fields = append(fields, fieldName+" "+parseResult.goType)

			fieldSetLine := fmt.Sprintf("%v: %v,", fieldName, parseResult.expr)
			fieldSetLines = append(fieldSetLines, fieldSetLine)
			handleArgs = append(handleArgs, parseResult.expr)

		case gi.DIRECTION_INOUT:
			parseResult := parseCbArgTypeDirInOut1(paramName, argTypeInfo)
			paramNameTypes = append(paramNameTypes, paramName+" "+parseResult.cgoType)
			cParamTypeNames = append(cParamTypeNames, parseResult.cType+" "+paramName)

			fields = append(fields, fieldName+" "+parseResult.goType)

			fieldSetLine := fmt.Sprintf("%v: %v,", fieldName, parseResult.expr)
			fieldSetLines = append(fieldSetLines, fieldSetLine)
			handleArgs = append(handleArgs, parseResult.expr)
		}

		argTypeInfo.Unref()
		argInfo.Unref()
	}

	retType := fi.ReturnType()
	defer retType.Unref()
	varResult := varReg.alloc("result")
	retResult := parseCbRet1(retType, varResult)

	wrapperFuncName := "gi" + _optNamespace + name

	s.CBody.Pn("extern %v %v(%v);", retResult.cType, wrapperFuncName, strings.Join(cParamTypeNames, ", "))
	s.CBody.Pn("static void* get%vWrapper() {", _optNamespace+name)
	s.CBody.Pn("    return (void*)(%v);", wrapperFuncName)
	s.CBody.Pn("}")

	argsStructName := name + "Arg"
	if len(fields) > 1 {
		argsStructName += "s"
	}
	s.GoBody.Pn("type %v struct {", argsStructName)
	for _, field := range fields {
		s.GoBody.Pn(field)
	}
	s.GoBody.Pn("}")

	s.GoBody.Pn("func Get%vWrapper() unsafe.Pointer {", name)
	s.GoBody.Pn("return unsafe.Pointer(C.get%vWrapper())", _optNamespace+name)
	s.GoBody.Pn("}")

	s.GoBody.Pn("//export %v", wrapperFuncName)
	retPart := ""
	varCResult := varReg.alloc("c_result")
	if retResult.goType != "" {
		retPart = fmt.Sprintf("(%v %v)", varCResult, retResult.cgoType)
	}
	s.GoBody.Pn("func %v(%v) %v {", wrapperFuncName, strings.Join(paramNameTypes, ", "), retPart)

	if foundUserData {
		varClosure := varReg.alloc("closure")
		s.GoBody.Pn("%v := gi.GetFunc(uint(uintptr(user_data)))", varClosure)

		s.GoBody.Pn("if %v.Fn != nil {", varClosure) // begin if 0

		varArgs := varReg.alloc("args")
		s.GoBody.Pn("%v := &%v{", varArgs, argsStructName)
		for _, line := range fieldSetLines {
			s.GoBody.Pn(line)
		}

		s.GoBody.Pn("}") // end struct
		varFn := varReg.alloc("fn")

		fnType := fmt.Sprintf("func(*%v) %v", argsStructName, retResult.goType)
		s.GoBody.Pn("%v := %v.Fn.(%v)", varFn, varClosure, fnType)

		// call fn
		if retResult.goType != "" {
			s.GoBody.Pn("%v := %v(%v)", varResult, varFn, varArgs)
			s.GoBody.Pn("%v = %v", varCResult, retResult.expr)
		} else {
			s.GoBody.Pn("%v(%v)", varFn, varArgs)
		}

		// 回调函数调用之后
		s.GoBody.Pn("if %v.Scope == gi.ScopeAsync {", varClosure) // begin if 1
		s.GoBody.Pn("    gi.UnregisterFunc(uint(uintptr(user_data)))")
		s.GoBody.Pn("}") // end if 1

		s.GoBody.Pn("}") // end if 0

	} else {
		// 没有 user_data 参数
		if strSliceContains(_cfg.ManualCallbacks, name) {
			s.GoBody.Pn("handleDestroyNotify(%v)", strings.Join(handleArgs, ", "))
		} else {
			// TODO
			s.GoBody.Pn("// TODO: not found user_data")
		}
	}

	if retResult.goType != "" {
		s.GoBody.Pn("return")
	}

	s.GoBody.Pn("}") // end func
}

type parseCbRetResult1 struct {
	cgoType string
	cType   string
	goType  string
	expr    string
}

func parseCbRet1(retType *gi.TypeInfo, varResult string) *parseCbRetResult1 {
	tag := retType.Tag()
	isPtr := retType.IsPointer()

	cgoType := "C.gpointer"
	cType := "gpointer"
	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO_CB tag: %v, isPtr: %v*/", tag, isPtr)
	expr := fmt.Sprintf("%v(%v)", cgoType, varResult)

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		if isPtr {
			cgoType = "*C.gchar"
			cType = "gchar*"
			goType = "string"
			expr = fmt.Sprintf("(%v)(C.CString(%v))", cgoType, varResult)
		}

	case gi.TYPE_TAG_BOOLEAN,
		gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型
		if !isPtr {
			cType = getCTypeWithTag(tag)
			cgoType = "C." + cType
			goType = getTypeWithTag(tag)
			expr = fmt.Sprintf("%v(%v)", cgoType, varResult)
			if tag == gi.TYPE_TAG_BOOLEAN {
				expr = fmt.Sprintf("%v(gi.Bool2Int(%v))", cgoType, varResult)
			}
		} else {
			cType = getCTypeWithTag(tag) + "*"
			cgoType = "*C." + getCTypeWithTag(tag)
			goType = "*" + getTypeWithTag(tag)
			expr = fmt.Sprintf("(%v)(unsafe.Pointer(%v))", cgoType, varResult)
		}

	case gi.TYPE_TAG_UNICHAR:
		if !isPtr {
			cType = "gunichar"
			cgoType = "C.gunichar"
			goType = "rune"
			expr = fmt.Sprintf("%v(%v)", cgoType, varResult)
		}

	case gi.TYPE_TAG_VOID:
		if isPtr {
			cgoType = "C.gpointer"
			cType = "gpointer"
			goType = "unsafe.Pointer"
			expr = fmt.Sprintf("%v(unsafe.Pointer(%v))", cgoType, varResult)
			// TODO 可能可以不需要 unsafe.Pointer()
		} else {
			cgoType = ""
			cType = "void"
			goType = ""
		}

	case gi.TYPE_TAG_INTERFACE:
		//ii := argTypeInfo.Interface()
		//ifcType := ii.Type()
		//if ifcType == gi.INFO_TYPE_ENUM || ifcType == gi.INFO_TYPE_FLAGS {
		//	if !isPtr {
		//		identPrefix := getCIdentifierPrefix(ii)
		//		name := ii.Name()
		//		cType = identPrefix + name
		//		cgoType = "C." + cType
		//
		//		name = getTypeName(ii) // 加上可能的包前缀
		//		if ifcType == gi.INFO_TYPE_ENUM {
		//			goType = getEnumTypeName(name)
		//		} else {
		//			// flags
		//			goType = getFlagsTypeName(name)
		//		}
		//		expr = fmt.Sprintf("%v(%v)", goType, paramName)
		//	}
		//} else if ifcType == gi.INFO_TYPE_STRUCT || ifcType == gi.INFO_TYPE_UNION ||
		//	ifcType == gi.INFO_TYPE_OBJECT || ifcType == gi.INFO_TYPE_INTERFACE {
		//	if isPtr {
		//		identPrefix := getCIdentifierPrefix(ii)
		//		name := ii.Name()
		//		if identPrefix == "cairo" {
		//			if name == "Context" {
		//				name = "_t"
		//			} else {
		//				name = "_" + strings.ToLower(name) + "_t"
		//			}
		//		}
		//		cType = identPrefix + name + "*"
		//		cgoType = "*C." + identPrefix + name
		//
		//		goType = getTypeName(ii)
		//		expr = fmt.Sprintf("%v{P: unsafe.Pointer(%v) }", goType, paramName)
		//		if ifcType == gi.INFO_TYPE_OBJECT {
		//			expr = fmt.Sprintf("%vWrap%v(unsafe.Pointer(%v))",
		//				getPkgPrefix(ii.Namespace()), name, paramName)
		//		}
		//	}
		//}
		//ii.Unref()

	case gi.TYPE_TAG_ARRAY:
		//arrType := argTypeInfo.ArrayType()
		//if arrType == gi.ARRAY_TYPE_C {
		//	elemTypeInfo := argTypeInfo.ParamType(0)
		//	elemTypeTag := elemTypeInfo.Tag()
		//
		//	elemType := getArgumentType(elemTypeTag)
		//	if elemType != "" && !elemTypeInfo.IsPointer() {
		//		cgoType = "C.gpointer"
		//		cType = "gpointer"
		//		goType = "gi." + elemType + "Array"
		//		expr = fmt.Sprintf("%s{P: unsafe.Pointer(%v)}", goType, paramName)
		//		// TODO length
		//
		//	} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && !elemTypeInfo.IsPointer() {
		//		goType = "unsafe.Pointer"
		//		cType = "gpointer"
		//		cgoType = "C.gpointer"
		//		expr = fmt.Sprintf("unsafe.Pointer(%v)", paramName)
		//	}
		//
		//	elemTypeInfo.Unref()
		//}

	case gi.TYPE_TAG_ERROR:
		//cType = "GError**"
		//cgoType = "**C.GError"
		//goType = "unsafe.Pointer"
		//expr = fmt.Sprintf("unsafe.Pointer(%v)", paramName)
	}

	return &parseCbRetResult1{
		cgoType: cgoType,
		cType:   cType,
		goType:  goType,
		expr:    expr,
	}
}

type parseCbArgTypeDirInOutResult1 struct {
	cgoType string
	cType   string
	goType  string
	expr    string
}

func parseCbArgTypeDirInOut1(paramName string, argTypeInfo *gi.TypeInfo) *parseCbArgTypeDirInOutResult1 {
	tag := argTypeInfo.Tag()
	isPtr := argTypeInfo.IsPointer()

	cgoType := "C.gpointer"
	cType := "gpointer"
	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO_CB tag: %v, isPtr: %v*/", tag, isPtr)
	expr := fmt.Sprintf("unsafe.Pointer(%v)", paramName)

	return &parseCbArgTypeDirInOutResult1{
		cgoType: cgoType,
		cType:   cType,
		goType:  goType,
		expr:    expr,
	}
}

type parseCbArgTypeDirOutResult1 struct {
	cgoType string
	cType   string
	goType  string
	expr    string
}

func parseCbArgTypeDirOut1(paramName string, argTypeInfo *gi.TypeInfo) *parseCbArgTypeDirOutResult1 {
	tag := argTypeInfo.Tag()
	isPtr := argTypeInfo.IsPointer()

	cgoType := "C.gpointer"
	cType := "gpointer"
	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO_CB tag: %v, isPtr: %v*/", tag, isPtr)
	expr := fmt.Sprintf("unsafe.Pointer(%v)", paramName)

	return &parseCbArgTypeDirOutResult1{
		cgoType: cgoType,
		cType:   cType,
		goType:  goType,
		expr:    expr,
	}
}

type parseCbArgTypeDirInResult1 struct {
	cgoType string
	cType   string
	goType  string
	expr    string
}

func parseCbArgTypeDirIn1(paramName string, argTypeInfo *gi.TypeInfo) *parseCbArgTypeDirInResult1 {
	tag := argTypeInfo.Tag()
	isPtr := argTypeInfo.IsPointer()

	cgoType := "C.gpointer"
	cType := "gpointer"
	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO_CB tag: %v, isPtr: %v*/", tag, isPtr)
	expr := fmt.Sprintf("unsafe.Pointer(%v)", paramName)

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		if isPtr {
			cgoType = "*C.gchar"
			cType = "gchar*"
			goType = "string"
			expr = fmt.Sprintf("gi.GoString(unsafe.Pointer(%v))", paramName)
		}

	case gi.TYPE_TAG_BOOLEAN,
		gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型
		if !isPtr {
			cType = getCTypeWithTag(tag)
			cgoType = "C." + cType
			goType = getTypeWithTag(tag)
			expr = fmt.Sprintf("%v(%v)", goType, paramName)
			if tag == gi.TYPE_TAG_BOOLEAN {
				expr = fmt.Sprintf("gi.Int2Bool(int(%v))", paramName)
			}
		} else {
			cType = getCTypeWithTag(tag) + "*"
			cgoType = "*C." + getCTypeWithTag(tag)
			goType = "*" + getTypeWithTag(tag)
			expr = fmt.Sprintf("(%v)(unsafe.Pointer(%v))", goType, paramName)
		}

	case gi.TYPE_TAG_UNICHAR:
		if !isPtr {
			cType = "gunichar"
			cgoType = "C.gunichar"
			goType = "rune"
			expr = fmt.Sprintf("rune(%v)", paramName)
		}

	case gi.TYPE_TAG_VOID:
		if isPtr {
			cgoType = "C.gpointer"
			cType = "gpointer"
			goType = "unsafe.Pointer"
			expr = fmt.Sprintf("unsafe.Pointer(%v)", paramName)
		}

	case gi.TYPE_TAG_INTERFACE:
		ii := argTypeInfo.Interface()
		ifcType := ii.Type()
		if ifcType == gi.INFO_TYPE_ENUM || ifcType == gi.INFO_TYPE_FLAGS {
			if !isPtr {
				identPrefix := getCIdentifierPrefix(ii)
				name := ii.Name()
				cType = identPrefix + name
				cgoType = "C." + cType

				name = getTypeName(ii) // 加上可能的包前缀
				if ifcType == gi.INFO_TYPE_ENUM {
					goType = getEnumTypeName(name)
				} else {
					// flags
					goType = getFlagsTypeName(name)
				}
				expr = fmt.Sprintf("%v(%v)", goType, paramName)
			}
		} else if ifcType == gi.INFO_TYPE_STRUCT || ifcType == gi.INFO_TYPE_UNION ||
			ifcType == gi.INFO_TYPE_OBJECT || ifcType == gi.INFO_TYPE_INTERFACE {
			if isPtr {
				identPrefix := getCIdentifierPrefix(ii)
				name := ii.Name()
				if identPrefix == "cairo" {
					if name == "Context" {
						name = "_t"
					} else {
						name = "_" + strings.ToLower(name) + "_t"
					}
				}
				cType = identPrefix + name + "*"
				cgoType = "*C." + identPrefix + name

				goType = getTypeName(ii)
				expr = fmt.Sprintf("%v{P: unsafe.Pointer(%v) }", goType, paramName)
				if ifcType == gi.INFO_TYPE_OBJECT {
					expr = fmt.Sprintf("%vWrap%v(unsafe.Pointer(%v))",
						getPkgPrefix(ii.Namespace()), name, paramName)
				}
			}
		}
		ii.Unref()

	case gi.TYPE_TAG_ARRAY:
		arrType := argTypeInfo.ArrayType()
		if arrType == gi.ARRAY_TYPE_C {
			elemTypeInfo := argTypeInfo.ParamType(0)
			elemTypeTag := elemTypeInfo.Tag()

			elemType := getArgumentType(elemTypeTag)
			if elemType != "" && !elemTypeInfo.IsPointer() {
				cgoType = "C.gpointer"
				cType = "gpointer"
				goType = "gi." + elemType + "Array"
				expr = fmt.Sprintf("%s{P: unsafe.Pointer(%v)}", goType, paramName)
				// TODO length

			} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && !elemTypeInfo.IsPointer() {
				goType = "unsafe.Pointer"
				cType = "gpointer"
				cgoType = "C.gpointer"
				expr = fmt.Sprintf("unsafe.Pointer(%v)", paramName)
			}

			elemTypeInfo.Unref()
		}

	case gi.TYPE_TAG_ERROR:
		cType = "GError**"
		cgoType = "**C.GError"
		goType = "unsafe.Pointer"
		expr = fmt.Sprintf("unsafe.Pointer(%v)", paramName)
	}
	return &parseCbArgTypeDirInResult1{
		cType:   cType,
		cgoType: cgoType,
		goType:  goType,
		expr:    expr,
	}
}

func getCIdentifierPrefix(bi *gi.BaseInfo) string {
	ns := bi.Namespace()
	return gi.DefaultRepository().CPrefix(ns)
}

func getCTypeWithTag(tag gi.TypeTag) (type0 string) {
	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		type0 = "gboolean" // typedef gint gboolean
	case gi.TYPE_TAG_INT8:
		type0 = "gint8"
	case gi.TYPE_TAG_UINT8:
		type0 = "guint8"

	case gi.TYPE_TAG_INT16:
		type0 = "gint16"
	case gi.TYPE_TAG_UINT16:
		type0 = "guint16"

	case gi.TYPE_TAG_INT32:
		type0 = "gint32"
	case gi.TYPE_TAG_UINT32:
		type0 = "guint32"

	case gi.TYPE_TAG_INT64:
		type0 = "gint64"
	case gi.TYPE_TAG_UINT64:
		type0 = "guint64"

	case gi.TYPE_TAG_FLOAT:
		type0 = "gfloat"
	case gi.TYPE_TAG_DOUBLE:
		type0 = "gdouble"

	case gi.TYPE_TAG_UNICHAR:
		type0 = "gunichar" // typedef guint32 gunichar
	}
	return
}
