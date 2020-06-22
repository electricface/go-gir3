package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/electricface/go-gir3/cmd/girgen/xmlp"
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
			parseResult := parseCbArgTypeDirOut(paramName, argTypeInfo)
			paramNameTypes = append(paramNameTypes, paramName+" "+parseResult.cgoType)
			cParamTypeNames = append(cParamTypeNames, parseResult.cType+" "+paramName)

			fieldName := "F_" + paramName
			fields = append(fields, fieldName+" "+parseResult.goType)

			fieldSetLine := fmt.Sprintf("%v: %v,", fieldName, parseResult.expr)
			fieldSetLines = append(fieldSetLines, fieldSetLine)

		case gi.DIRECTION_INOUT:
			parseResult := parseCbArgTypeDirInOut(paramName, argTypeInfo)
			paramNameTypes = append(paramNameTypes, paramName+" "+parseResult.cgoType)
			cParamTypeNames = append(cParamTypeNames, parseResult.cType+" "+paramName)

			fieldName := "F_" + paramName
			fields = append(fields, fieldName+" "+parseResult.goType)

			fieldSetLine := fmt.Sprintf("%v: %v,", fieldName, parseResult.expr)
			fieldSetLines = append(fieldSetLines, fieldSetLine)
		}

		argTypeInfo.Unref()
		argInfo.Unref()
	}

	myFuncName := "my" + _optNamespace + name

	s.CBody.Pn("extern void %s(%v);", myFuncName, strings.Join(cParamTypeNames, ", "))
	s.CBody.Pn("static void* getPointer_%v() {", myFuncName)
	s.CBody.Pn("return (void*)(%v);", myFuncName)
	s.CBody.Pn("}")

	s.GoBody.Pn("type %vStruct struct {", name)
	for _, field := range fields {
		s.GoBody.Pn(field)
	}
	s.GoBody.Pn("}")

	s.GoBody.Pn("func GetPointer_my%v() unsafe.Pointer {", name)
	s.GoBody.Pn("return unsafe.Pointer(C.getPointer_%v())", myFuncName)
	s.GoBody.Pn("}")

	s.GoBody.Pn("//export %v", myFuncName)
	s.GoBody.Pn("func %v(%v) {", myFuncName, strings.Join(paramNameTypes, ", "))

	if foundUserData {
		varFn := varReg.alloc("fn")
		s.GoBody.Pn("%v := gi.GetFunc(uint(uintptr(user_data)))", varFn)
		varArgs := varReg.alloc("args")
		s.GoBody.Pn("%v := &%vStruct{", varArgs, name)
		for _, line := range fieldSetLines {
			s.GoBody.Pn(line)
		}

		s.GoBody.Pn("}") // end struct
		s.GoBody.Pn("%v(%v)", varFn, varArgs)
	} else {
		// 没有 user_data 参数
		// TODO
		s.GoBody.Pn("// TODO: not found user_data")
	}

	s.GoBody.Pn("}") // end func
}

type parseCbArgTypeDirInOutResult struct {
	cgoType string
	cType   string
	goType  string
	expr    string
}

func parseCbArgTypeDirInOut(paramName string, argTypeInfo *gi.TypeInfo) *parseCbArgTypeDirInOutResult {
	tag := argTypeInfo.Tag()
	isPtr := argTypeInfo.IsPointer()

	cgoType := "C.gpointer"
	cType := "gpointer"
	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO_CB tag: %v, isPtr: %v*/", tag, isPtr)
	expr := fmt.Sprintf("unsafe.Pointer(%v)", paramName)

	return &parseCbArgTypeDirInOutResult{
		cgoType: cgoType,
		cType:   cType,
		goType:  goType,
		expr:    expr,
	}
}

type parseCbArgTypeDirOutResult struct {
	cgoType string
	cType   string
	goType  string
	expr    string
}

func parseCbArgTypeDirOut(paramName string, argTypeInfo *gi.TypeInfo) *parseCbArgTypeDirOutResult {
	tag := argTypeInfo.Tag()
	isPtr := argTypeInfo.IsPointer()

	cgoType := "C.gpointer"
	cType := "gpointer"
	goType := "unsafe.Pointer" + fmt.Sprintf("/*TODO_CB tag: %v, isPtr: %v*/", tag, isPtr)
	expr := fmt.Sprintf("unsafe.Pointer(%v)", paramName)

	return &parseCbArgTypeDirOutResult{
		cgoType: cgoType,
		cType:   cType,
		goType:  goType,
		expr:    expr,
	}
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
	return &parseCbArgTypeDirInResult{
		cType:   cType,
		cgoType: cgoType,
		goType:  goType,
		expr:    expr,
	}
}

func getCIdentifierPrefix(bi *gi.BaseInfo) string {
	ns := bi.Namespace()

	if ns == _xRepo.Namespace.Name {
		return _xRepo.Namespace.CIdentifierPrefixes
	}

	repo := xmlp.GetLoadedRepo(ns)
	return repo.Namespace.CIdentifierPrefixes
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
