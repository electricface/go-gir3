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
		}
		ii.Unref()
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

	if ns == globalXRepo.Namespace.Name {
		return globalXRepo.Namespace.CIdentifierPrefixes
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
