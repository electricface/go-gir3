package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/electricface/go-gir3/gi"
)

// 给 InvokeCache.Get() 用的 index 的
var globalFuncNextIdx int

var globalNumTodoFunc int

func getFunctionName(fi *gi.FunctionInfo) string {
	fiName := fi.Name()
	fnName := snake2Camel(fiName)

	fnFlags := fi.Flags()
	if fnFlags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
		// 表示 C 函数是构造器
		fnName = getConstructorName(fi.Container().Name(), fnName)
	}
	return fnName
}

func getFunctionNameFinal(fi *gi.FunctionInfo) string {
	// 只用于 pFunction() 中
	symbol := fi.Symbol()
	name := globalSymbolNameMap[symbol]
	if name != "" {
		return name
	}
	return getFunctionName(fi)
}

/*

{ // begin func

beforeArgLines

newArgLines

call

afterCallLines

setParamLines

beforeRetLines

return

} // end func

*/

func pFunction(s *SourceFile, fi *gi.FunctionInfo) {
	b := &SourceBlock{}
	symbol := fi.Symbol()
	fiName := fi.Name()
	// 用于黑名单识别函数的名字
	identifyName := fiName
	container := fi.Container()
	if container != nil {
		identifyName = container.Name() + "." + fiName
	}
	if strSliceContains(globalCfg.Black, identifyName) {
		b.Pn("\n// black function %s\n", identifyName)
		return
	}

	b.Pn("// %s", symbol)
	funcIdx := globalFuncNextIdx
	globalFuncNextIdx++

	fnName := getFunctionNameFinal(fi)

	// 函数内变量名称分配器
	var varReg VarReg
	// 目标函数形参列表，元素是 "名字 类型"
	var params []string
	// 目标函数返回参数列表，元素是 "名字 类型"
	var retParams []string

	// 准备传递给 invoker.Call 中的参数的代码之前的语句
	var beforeArgLines []string
	// 准备传递给 invoker.Call 中的参数的语句
	var newArgLines []string
	// 传递给 invoker.Call 中的参数列表
	var argNames []string

	// 在 invoker.Call 执行后需要执行的语句
	var afterCallLines []string

	var setParamLines []string

	var beforeRetLines []string

	// direction 为 inout 或 out 的参数个数
	var numOutArgs int
	var outArgIdx int

	var varOutArgs string
	var receiver string

	// 如果为 true，则 C 函数函数中最后一个是 **GError err
	var isThrows bool

	fnFlags := fi.Flags()
	varErr := varReg.alloc("err")
	if fnFlags&gi.FUNCTION_THROWS != 0 {
		isThrows = true
	}

	argIdxStart := 0
	if container != nil {
		addReceiver := false
		log.Println("container is not nil")
		b.Pn("// container is not nil, container is %s", container.Name())
		if fnFlags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
			// 表示 C 函数是构造器
			b.Pn("// is constructor")
		} else if fnFlags&gi.FUNCTION_IS_METHOD != 0 {
			// 表示 C 函数是方法
			b.Pn("// is method")
			addReceiver = true
		} else {
			// 可能 C 函数还是可以作为方法的，只不过没有处理好参数，如果第一个参数是指针类型，就大概率是方法。
			if fi.NumArg() > 0 {
				b.Pn("// is method")
				arg0 := fi.Arg(0)
				arg0Type := arg0.Type()
				b.Pn("// arg0Type tag: %v, isPtr: %v", arg0Type.Tag(), arg0Type.IsPointer())
				if arg0Type.IsPointer() && arg0Type.Tag() == gi.TYPE_TAG_INTERFACE {
					ii := arg0Type.Interface()
					if ii.Name() == container.Name() {
						addReceiver = true
						// 从 1 开始
						argIdxStart = 1
					}
					ii.Unref()
				}

				if !addReceiver {
					// 不能作为方法, 作为函数
					fnName = container.Name() + fnName + "1"
					// TODO: 适当消除 1 后缀
				}
			} else {
				b.Pn("// num arg is 0")
				// 比如 io_channel_error_quark 方法，被重命名为IOChannel.error_quark，这算是 IOChannel 的 static 方法，
				// 但是 Go 里没有类的概念，于是直接忽略这个方法了，但任然会为在 namespace 顶层的 io_channel_error_quark 方法自动生成代码。
				return
			}
		}

		if addReceiver {
			// 容器是 interface 类型的
			isContainerIfc := false
			if container.Type() == gi.INFO_TYPE_INTERFACE {
				isContainerIfc = true
			}

			receiverType := container.Name()
			if isContainerIfc {
				receiverType = "*" + receiverType + "Ifc"
			}

			varV := varReg.alloc("v")
			receiver = fmt.Sprintf("(%s %s)", varV, receiverType)
			varArgV := varReg.alloc("arg_v")
			getPtrExpr := fmt.Sprintf("%s.P", varV)
			if isContainerIfc {
				getPtrExpr = fmt.Sprintf("*(*unsafe.Pointer)(unsafe.Pointer(%v))", varV)
			}
			newArgLines = append(newArgLines, fmt.Sprintf("%v := gi.NewPointerArgument(%s)",
				varArgV, getPtrExpr))
			argNames = append(argNames, varArgV)
		}
	} else {
		b.Pn("// container is nil")
	}

	// lenArgMap 是数组长度参数的集合，键是长度参数的 index
	lenArgMap := make(map[int]struct{})
	numArgs := fi.NumArg()
	for i := argIdxStart; i < numArgs; i++ {
		argInfo := fi.Arg(i)
		varReg.regParam(i, argInfo.Name())
		argType := argInfo.Type()

		typeTag := argType.Tag()
		if typeTag == gi.TYPE_TAG_ARRAY {
			lenArgIdx := argType.ArrayLength()
			if lenArgIdx >= 0 {
				lenArgMap[lenArgIdx] = struct{}{}
				b.Pn("// arg %v %v lenArgIdx %v", i, argInfo.Name(), lenArgIdx)
			}
		}

		argType.Unref()
		argInfo.Unref()
	}
	retTypeInfo := fi.ReturnType()
	defer retTypeInfo.Unref()
	retTypeTag := retTypeInfo.Tag()
	if retTypeTag == gi.TYPE_TAG_ARRAY {
		lenArgIdx := retTypeInfo.ArrayLength()
		if lenArgIdx >= 0 {
			lenArgMap[lenArgIdx] = struct{}{}
			b.Pn("// ret lenArgIdx %v", lenArgIdx)
		}
	}

	for i := argIdxStart; i < numArgs; i++ {
		argInfo := fi.Arg(i)
		argTypeInfo := argInfo.Type()
		dir := argInfo.Direction()
		switch dir {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			numOutArgs++
			if varOutArgs == "" {
				varOutArgs = varReg.alloc("outArgs")
			}
		}

		paramName := varReg.getParam(i)

		if dir == gi.DIRECTION_IN || dir == gi.DIRECTION_INOUT {
			// 作为目标函数的输入参数之一

			type0 := "int/*TODO:TYPE*/"
			if dir == gi.DIRECTION_IN {
				parseResult := parseArgTypeDirIn(paramName, argTypeInfo, &varReg)

				type0 = parseResult.type0
				beforeArgLines = append(beforeArgLines, parseResult.beforeArgLines...)

				varArg := varReg.alloc("arg_" + paramName)
				argNames = append(argNames, varArg)
				newArgLines = append(newArgLines, fmt.Sprintf("%v := %v", varArg, parseResult.newArgExpr))

				afterCallLines = append(afterCallLines, parseResult.afterCallLines...)
			} else {
				// TODO：处理 dir 为 inout 的
			}

			params = append(params, paramName+" "+type0)

		} else if dir == gi.DIRECTION_OUT {
			// 作为目标函数的返回值之一
			parseResult := parseArgTypeDirOut(paramName, argTypeInfo, &varReg)
			type0 := parseResult.type0
			if _, ok := lenArgMap[i]; ok {
				// 参数是数组的长度
				afterCallLines = append(afterCallLines,
					fmt.Sprintf("var %v %v; _ = %v", paramName, type0, paramName))
			} else {
				retParams = append(retParams, paramName+" "+type0)
			}

			varArg := varReg.alloc("arg_" + paramName)
			argNames = append(argNames, varArg)
			newArgLines = append(newArgLines, fmt.Sprintf("%v := gi.NewPointerArgument(unsafe.Pointer(&%v[%v]))", varArg, varOutArgs, outArgIdx))
			getValExpr := fmt.Sprintf("%v[%v].%v", varOutArgs, outArgIdx, parseResult.expr)

			setParamLine := fmt.Sprintf("%v%v = %v",
				paramName, parseResult.field, getValExpr)

			if parseResult.needTypeCast {
				setParamLine = fmt.Sprintf("%v%v = %v(%s)",
					paramName, parseResult.field, type0, getValExpr)
			}

			// setParamLine 类似 param1 = outArgs[1].Int(), 或 param1 = rune(outArgs[1].Uint32())
			// 或 param1.P = outArgs[1].Pointer()
			setParamLines = append(setParamLines, setParamLine)

			beforeRetLines = append(beforeRetLines, parseResult.beforeRetLines...)

			outArgIdx++
		}

		argTypeInfo.Unref()
		argInfo.Unref()
	}
	if isThrows {
		numOutArgs++
		if varOutArgs == "" {
			varOutArgs = varReg.alloc("outArgs")
		}
		varArg := varReg.alloc("arg_" + varErr)
		argNames = append(argNames, varArg)
		newArgLines = append(newArgLines, fmt.Sprintf("%v := gi.NewPointerArgument(unsafe.Pointer(&%v[%v]))", varArg, varOutArgs, outArgIdx))
		afterCallLines = append(afterCallLines, fmt.Sprintf("%v = gi.ToError(%v[%v].%v)", varErr, varOutArgs, outArgIdx, "Pointer()"))
		retParams = append(retParams, varErr+" error")
	}

	var varRet string
	var varResult string
	var parseRetTypeResult *parseRetTypeResult

	// 是否**无**返回值
	var isRetVoid bool
	if gi.TYPE_TAG_VOID == retTypeInfo.Tag() && !retTypeInfo.IsPointer() {
		// 无返回值
		isRetVoid = true
	} else {
		// 有返回值
		varRet = varReg.alloc("ret")
		varResult = varReg.alloc("result")
		parseRetTypeResult = parseRetType(varRet, retTypeInfo, &varReg, fi)
		// 把返回值加在 retParams 列表最前面
		retParams = append([]string{varResult + " " + parseRetTypeResult.type0}, retParams...)
	}

	paramsJoined := strings.Join(params, ", ")

	retParamsJoined := strings.Join(retParams, ", ")
	if len(retParams) > 0 {
		retParamsJoined = "(" + retParamsJoined + ")"
	}
	// 输出目标函数头部
	b.Pn("func %s %s(%s) %s {", receiver, fnName, paramsJoined, retParamsJoined)

	varInvoker := varReg.alloc("iv")
	if container == nil {
		b.Pn("%s, %s := _I.Get(%d, %q, \"\")", varInvoker, varErr, funcIdx, fiName)
	} else {
		b.Pn("%s, %s := _I.Get(%d, %q, %q)", varInvoker, varErr, funcIdx, container.Name(), fiName)
	}

	{ // 处理 invoker 获取失败的情况

		b.Pn("if %s != nil {", varErr)

		if isThrows {
			// 使用 err 变量返回错误
		} else {
			// 把 err 打印出来
			b.Pn("log.Println(\"WARN:\", %s)", varErr)
		}
		b.Pn("return")

		b.Pn("}") // end if err != nil
	}

	if numOutArgs > 0 {
		b.Pn("var %s [%d]gi.Argument", varOutArgs, numOutArgs)
	}

	for _, line := range beforeArgLines {
		b.Pn(line)
	}

	for _, line := range newArgLines {
		b.Pn(line)
	}

	callArgArgs := "nil"
	if len(argNames) > 0 {
		// 比如输出 args := []gi.Argument{arg0,arg1}
		varArgs := varReg.alloc("args")
		b.Pn("%s := []gi.Argument{%s}", varArgs, strings.Join(argNames, ", "))
		callArgArgs = varArgs
	}

	callArgRet := "nil"
	if !isRetVoid {
		// 有返回值
		callArgRet = "&" + varRet
		b.Pn("var %s gi.Argument", varRet)
	}
	callArgOutArgs := "nil"
	if numOutArgs > 0 {
		callArgOutArgs = fmt.Sprintf("&%s[0]", varOutArgs)
	}
	b.Pn("%s.Call(%s, %s, %s)", varInvoker, callArgArgs, callArgRet, callArgOutArgs)

	for _, line := range afterCallLines {
		b.Pn(line)
	}

	for _, line := range setParamLines {
		b.Pn(line)
	}

	if !isRetVoid && parseRetTypeResult != nil {
		b.Pn("%s%s = %s", varResult, parseRetTypeResult.field, parseRetTypeResult.expr)
		if parseRetTypeResult.zeroTerm {
			b.Pn("%v.SetLenZT()", varResult)
		}
	}

	for _, line := range beforeRetLines {
		b.Pn(line)
	}

	if len(retParams) > 0 {
		b.Pn("return")
	}

	b.Pn("}") // end func
	if b.containsTodo() {
		globalNumTodoFunc++
	}
	s.GoBody.addBlock(b)
}

type parseRetTypeResult struct {
	expr     string // 转换 argument 为返回值类型的表达式
	field    string // expr 要给 result 的什么字段设置，比如 .P 字段
	type0    string // 目标函数中返回值类型
	zeroTerm bool
}

func parseRetType(varRet string, ti *gi.TypeInfo, varReg *VarReg, fi *gi.FunctionInfo) *parseRetTypeResult {
	isPtr := ti.IsPointer()
	tag := ti.Tag()
	type0 := getDebugType("isPtr: %v, tag: %v", isPtr, tag)
	expr := varRet + ".Int()/*TODO*/"
	field := ""
	zeroTerm := false
	fiFlags := fi.Flags()

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		// 字符串类型
		// 产生类似如下代码：
		// result = ret.String().Take()
		expr = varRet + ".String().Take()"
		type0 = "string"

	case gi.TYPE_TAG_BOOLEAN,
		gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型
		// 产生类似如下代码：
		// result = ret.Bool()
		expr = fmt.Sprintf("%s.%s()", varRet, getArgumentType(tag))
		type0 = getTypeWithTag(tag)

	case gi.TYPE_TAG_UNICHAR:
		// 产生如下代码：
		// result = rune(ret.Uint32())
		expr = fmt.Sprintf("rune(%v.Uint32())", varRet)
		type0 = "rune"

	case gi.TYPE_TAG_INTERFACE:
		bi := ti.Interface()
		biType := bi.Type()
		if isPtr {
			type0 = getTypeName(bi)

			if fiFlags & gi.FUNCTION_IS_CONSTRUCTOR != 0 {
				container := fi.Container()
				if container != nil {
					type0 = getTypeName(container)
					container.Unref()
				}
			}

			expr = fmt.Sprintf("%v.Pointer()", varRet)
			field = ".P"

		} else {
			if biType == gi.INFO_TYPE_FLAGS {
				type0 = getFlagsTypeName(getTypeName(bi))
				expr = fmt.Sprintf("%v(%v.Int())", type0, varRet)
			} else if biType == gi.INFO_TYPE_ENUM {
				type0 = getEnumTypeName(getTypeName(bi))
				expr = fmt.Sprintf("%v(%v.Int())", type0, varRet)
			}
		}
		bi.Unref()

	case gi.TYPE_TAG_ERROR:
		type0 = getGLibType("Error")
		expr = fmt.Sprintf("%v.Pointer()", varRet)
		field = ".P"

	case gi.TYPE_TAG_GTYPE:
		type0 = "gi.GType"
		expr = fmt.Sprintf("gi.GType(%v.Uint())", varRet)

	case gi.TYPE_TAG_GHASH:
		type0 = getGLibType("HashTable")
		expr = fmt.Sprintf("%v.Pointer()", varRet)
		field = ".P"

	case gi.TYPE_TAG_VOID:
		isPtr := ti.IsPointer()
		if isPtr {
			type0 = "unsafe.Pointer"
			expr = varRet + ".Pointer()"
		}

	case gi.TYPE_TAG_ARRAY:
		arrType := ti.ArrayType()
		lenArgIdx := ti.ArrayLength()
		isZeroTerm := ti.IsZeroTerminated()

		type0 = getDebugType("array type: %v, isZeroTerm: %v", arrType, isZeroTerm)

		if arrType == gi.ARRAY_TYPE_C {
			elemTypeInfo := ti.ParamType(0)
			elemTypeTag := elemTypeInfo.Tag()

			type0 = getDebugType("array type c, elemTypeTag: %v, isPtr: %v", elemTypeTag, elemTypeInfo.IsPointer())

			elemType := getArgumentType(elemTypeTag)
			if elemType != "" && !elemTypeInfo.IsPointer() {
				type0 = "gi." + elemType + "Array"

				argName := "0"
				if lenArgIdx >= 0 {
					argInfo := fi.Arg(lenArgIdx)
					argName = argInfo.Name()
					argInfo.Unref()
				}
				expr = fmt.Sprintf("%v{ P: %v.Pointer(), Len: int(%s) }", type0, varRet, argName)

			} else if elemTypeTag == gi.TYPE_TAG_UTF8 || elemTypeTag == gi.TYPE_TAG_FILENAME {
				type0 = "gi.CStrArray"
				lenExpr := "-1" // zero-terminated 以零结尾的数组
				if isZeroTerm {
					zeroTerm = true
				} else {
					lenExpr = "int(" + varReg.getParam(lenArgIdx) + ")"
				}
				expr = fmt.Sprintf("%v{ P: %v.Pointer(), Len: %v }", type0, varRet, lenExpr)
			} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && elemTypeInfo.IsPointer() {
				type0 = "gi.PointerArray"
				lenExpr := "-1" // zero-terminated 以零结尾的数组
				if isZeroTerm {
					zeroTerm = true
				} else {
					lenExpr = "int(" + varReg.getParam(lenArgIdx) + ")"
				}
				expr = fmt.Sprintf("%v{ P: %v.Pointer(), Len: %v }", type0, varRet, lenExpr)
			} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && !elemTypeInfo.IsPointer() {
				type0 = "unsafe.Pointer"
				expr = varRet + ".Pointer()"
			}

			elemTypeInfo.Unref()
		} else if arrType == gi.ARRAY_TYPE_BYTE_ARRAY {
			type0 = getGLibType("ByteArray")
			expr = fmt.Sprintf("%v.Pointer()", varRet)
			field = ".P"
		}
	}

	return &parseRetTypeResult{
		field:    field,
		expr:     expr,
		type0:    type0,
		zeroTerm: zeroTerm,
	}
}

func getDebugType(format string, args ...interface{}) string {
	debugMsg := fmt.Sprintf(format, args...)
	type0 := fmt.Sprintf("int/*TODO_TYPE %s*/", debugMsg)
	return type0
}

type parseArgTypeDirOutResult struct {
	expr           string // 转换 arguemnt 为返回值类型的表达式
	type0          string // 目标函数中返回值类型
	needTypeCast   bool   // 是否需要类型转换
	field          string // 表达式赋值的字段
	beforeRetLines []string
}

func parseArgTypeDirOut(paramName string, ti *gi.TypeInfo, varReg *VarReg) *parseArgTypeDirOutResult {
	expr := "Int()/*TODO*/"
	type0 := "int/*TODO_TYPE*/"
	needTypeCast := false
	field := ""
	var beforeRetLines []string

	tag := ti.Tag()
	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		// 字符串类型
		// 产生类似如下代码：
		// outArg1 = &outArgs[0].String().Take()
		//                       ^--------------
		expr = "String().Take()"
		type0 = "string"

	case gi.TYPE_TAG_BOOLEAN,
		gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型
		// 产生类似如下代码：
		// outArg1 = &outArgs[0].Bool()
		//                       ^_____
		expr = fmt.Sprintf("%s()", getArgumentType(tag))
		type0 = getTypeWithTag(tag)

	case gi.TYPE_TAG_UNICHAR:
		expr = "Uint32()"
		type0 = "rune"
		needTypeCast = true

	case gi.TYPE_TAG_INTERFACE:
		isPtr := ti.IsPointer()
		bi := ti.Interface()
		biType := bi.Type()
		if isPtr {
			if biType == gi.INFO_TYPE_OBJECT || biType == gi.INFO_TYPE_INTERFACE ||
				biType == gi.INFO_TYPE_STRUCT {

				type0 = getTypeName(bi)
				expr = "Pointer()"
				field = ".P"
			} else {
				debugMsg := fmt.Sprintf("tagIfc biType: %v", biType)
				expr = fmt.Sprintf("Int()/*TODO %s*/", debugMsg)
				// 目前这里只发现了在 pango_tab_array_get_tabs 中 biType 为 enum
			}

		} else {
			if biType == gi.INFO_TYPE_FLAGS {
				type0 = getFlagsTypeName(getTypeName(bi))
				expr = "Int()"
				needTypeCast = true
			} else if biType == gi.INFO_TYPE_ENUM {
				type0 = getEnumTypeName(getTypeName(bi))
				expr = "Int()"
				needTypeCast = true
			}
		}
		bi.Unref()

	case gi.TYPE_TAG_ERROR:
		type0 = getGLibType("Error")
		expr = "Pointer()"
		field = ".P"

	case gi.TYPE_TAG_GTYPE:
		type0 = "gi.GType"
		expr = "Uint()"
		needTypeCast = true

	case gi.TYPE_TAG_GHASH:
		type0 = getGLibType("HashTable")
		expr = "Pointer()"
		field = ".P"

	case gi.TYPE_TAG_VOID:
		isPtr := ti.IsPointer()
		if isPtr {
			type0 = "unsafe.Pointer"
			expr = "Pointer()"
		}

	case gi.TYPE_TAG_ARRAY:
		arrType := ti.ArrayType()
		lenArgIdx := ti.ArrayLength()
		if arrType == gi.ARRAY_TYPE_C {

			elemTypeInfo := ti.ParamType(0)
			elemTypeTag := elemTypeInfo.Tag()
			type0 = getDebugType("array type c, elemTypeTag: %v", elemTypeTag)

			elemType := getArgumentType(elemTypeTag)
			if elemType != "" && !elemTypeInfo.IsPointer() {
				type0 = "gi." + elemType + "Array"
				expr = "Pointer()"
				field = ".P"

				if lenArgIdx >= 0 {
					lenArgName := varReg.getParam(lenArgIdx)
					beforeRetLines = append(beforeRetLines,
						fmt.Sprintf("%v.Len = int(%v)", paramName, lenArgName))
				}

			} else if elemTypeTag == gi.TYPE_TAG_UTF8 || elemTypeTag == gi.TYPE_TAG_FILENAME {
				type0 = "gi.CStrArray"
				expr = "Pointer()"
				field = ".P"
			} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && elemTypeInfo.IsPointer() {
				type0 = "gi.PointerArray"
				expr = "Pointer()"
				field = ".P"

				if lenArgIdx >= 0 {
					lenArgName := varReg.getParam(lenArgIdx)
					beforeRetLines = append(beforeRetLines,
						fmt.Sprintf("%v.Len = int(%v)", paramName, lenArgName))
				} else {
					beforeRetLines = append(beforeRetLines,
						fmt.Sprintf("%v.Len = -1", paramName))

					// 注意: 可能不一定是 Zero Term 的
					beforeRetLines = append(beforeRetLines,
						fmt.Sprintf("%v.SetLenZT()", paramName))
				}
			} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && !elemTypeInfo.IsPointer() {
				type0 = "unsafe.Pointer"
				expr = "Pointer()"
			}

			elemTypeInfo.Unref()

		} else if arrType == gi.ARRAY_TYPE_BYTE_ARRAY {
			type0 = getGLibType("ByteArray")
			expr = "Pointer()"
			field = ".P"
		}
	}

	return &parseArgTypeDirOutResult{
		expr:           expr,
		type0:          type0,
		needTypeCast:   needTypeCast,
		field:          field,
		beforeRetLines: beforeRetLines,
	}
}

func parseArgTypeDirInOut() {
	// TODO
}

func getTypeWithTag(tag gi.TypeTag) (type0 string) {
	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		type0 = "bool"
	case gi.TYPE_TAG_INT8:
		type0 = "int8"
	case gi.TYPE_TAG_UINT8:
		type0 = "uint8"

	case gi.TYPE_TAG_INT16:
		type0 = "int16"
	case gi.TYPE_TAG_UINT16:
		type0 = "uint16"

	case gi.TYPE_TAG_INT32:
		type0 = "int32"
	case gi.TYPE_TAG_UINT32:
		type0 = "uint32"

	case gi.TYPE_TAG_INT64:
		type0 = "int64"
	case gi.TYPE_TAG_UINT64:
		type0 = "uint64"

	case gi.TYPE_TAG_FLOAT:
		type0 = "float32"
	case gi.TYPE_TAG_DOUBLE:
		type0 = "float64"

	case gi.TYPE_TAG_UNICHAR:
		type0 = "rune"
	}
	return
}

func getArgumentType(tag gi.TypeTag) (str string) {
	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		str = "Bool"
	case gi.TYPE_TAG_INT8:
		str = "Int8"
	case gi.TYPE_TAG_UINT8:
		str = "Uint8"

	case gi.TYPE_TAG_INT16:
		str = "Int16"
	case gi.TYPE_TAG_UINT16:
		str = "Uint16"

	case gi.TYPE_TAG_INT32:
		str = "Int32"
	case gi.TYPE_TAG_UINT32:
		str = "Uint32"

	case gi.TYPE_TAG_INT64:
		str = "Int64"
	case gi.TYPE_TAG_UINT64:
		str = "Uint64"

	case gi.TYPE_TAG_FLOAT:
		str = "Float"
	case gi.TYPE_TAG_DOUBLE:
		str = "Double"

	case gi.TYPE_TAG_UNICHAR:
		str = "Uint32"
	}
	return
}

type parseArgTypeDirInResult struct {
	newArgExpr     string   // 创建 Argument 的表达式，比如 gi.NewIntArgument()
	type0          string   // 目标函数形参中的类型
	beforeArgLines []string // 在 arg_xxx = gi.NewXXXArgument 之前执行的语句
	afterCallLines []string // 在 invoker.Call() 之后执行的语句
}

func parseArgTypeDirIn(varArg string, ti *gi.TypeInfo, varReg *VarReg) *parseArgTypeDirInResult {
	// 处理 direction 为 in 的情况
	var beforeArgLines []string
	var afterCallLines []string

	tag := ti.Tag()
	isPtr := ti.IsPointer()

	debugMsg := ""
	debugMsg = fmt.Sprintf("isPtr: %v, tag: %v", isPtr, tag)
	type0 := fmt.Sprintf("int/*TODO_TYPE %s*/", debugMsg)
	newArgExpr := fmt.Sprintf("gi.NewIntArgument(%s)/*TODO*/", varArg)

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		// 字符串类型
		// 产生类似如下代码：
		// c_arg1 = gi.CString(arg1)
		// arg_arg1 = gi.NewStringArgument(c_arg1)
		//            ^---------------------------
		// after call:
		// gi.Free(c_arg1)
		varCArg := varReg.alloc("c_" + varArg)
		beforeArgLines = append(beforeArgLines,
			fmt.Sprintf("%s := gi.CString(%s)", varCArg, varArg))
		newArgExpr = fmt.Sprintf("gi.NewStringArgument(%s)", varCArg)
		afterCallLines = append(afterCallLines,
			fmt.Sprintf("gi.Free(%s)", varCArg))
		type0 = "string"

	case gi.TYPE_TAG_BOOLEAN,
		gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型

		argType := getArgumentType(tag)
		newArgExpr = fmt.Sprintf("gi.New%vArgument(%v)", argType, varArg)
		type0 = getTypeWithTag(tag)

	case gi.TYPE_TAG_UNICHAR:
		newArgExpr = fmt.Sprintf("gi.NewUint32Argument(uint32(%v))", varArg)
		type0 = "rune"

	case gi.TYPE_TAG_VOID:
		if isPtr {
			// ti 指的类型就是 void* , 翻译为 unsafe.Pointer
			type0 = "unsafe.Pointer"
			newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%s)", varArg)
		}

	case gi.TYPE_TAG_INTERFACE:
		bi := ti.Interface()
		biType := bi.Type()
		if isPtr {
			type0 = getTypeName(bi)
			newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%s.P)", varArg)

			if bi.Type() == gi.INFO_TYPE_OBJECT {
				if strings.Contains(type0, ".") {
					// gobject.Object => gobject.IObject
					type0 = strings.Replace(type0, ".", ".I", 1)
				} else {
					// Object => IObject
					type0 = "I" + type0
				}
				newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%s.P_%s())", varArg, bi.Name())
			}

		} else {
			if biType == gi.INFO_TYPE_FLAGS {
				type0 = getFlagsTypeName(getTypeName(bi))
				newArgExpr = fmt.Sprintf("gi.NewIntArgument(int(%v))", varArg)
			} else if biType == gi.INFO_TYPE_ENUM {
				type0 = getEnumTypeName(getTypeName(bi))
				newArgExpr = fmt.Sprintf("gi.NewIntArgument(int(%v))", varArg)
			}
		}
		bi.Unref()

	case gi.TYPE_TAG_ERROR:
		type0 = getGLibType("Error")
		newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v.P)", varArg)

	case gi.TYPE_TAG_GTYPE:
		type0 = "gi.GType"
		newArgExpr = fmt.Sprintf("gi.NewUintArgument(uint(%v))", varArg)

	case gi.TYPE_TAG_GHASH:
		type0 = getGLibType("HashTable")
		newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v.P)", varArg)

	case gi.TYPE_TAG_ARRAY:
		arrType := ti.ArrayType()
		if arrType == gi.ARRAY_TYPE_C {
			elemTypeInfo := ti.ParamType(0)
			elemTypeTag := elemTypeInfo.Tag()
			type0 = getDebugType("array type c, elemTypeTag: %v", elemTypeTag)

			elemType := getArgumentType(elemTypeTag)
			if elemType != "" && !elemTypeInfo.IsPointer() {
				type0 = "gi." + elemType + "Array"
				newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%s.P)", varArg)

			} else if elemTypeTag == gi.TYPE_TAG_UTF8 || elemTypeTag == gi.TYPE_TAG_FILENAME {
				type0 = "gi.CStrArray"
				newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v.P)", varArg)
			} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && elemTypeInfo.IsPointer() {
				type0 = "gi.PointerArray"
				newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v.P)", varArg)
			} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && !elemTypeInfo.IsPointer() {
				type0 = "unsafe.Pointer"
				newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v)", varArg)
			}

			elemTypeInfo.Unref()
		} else if arrType == gi.ARRAY_TYPE_BYTE_ARRAY {
			type0 = getGLibType("ByteArray")
			newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v.P)", varArg)
		}
	}

	return &parseArgTypeDirInResult{
		newArgExpr:     newArgExpr,
		type0:          type0,
		beforeArgLines: beforeArgLines,
		afterCallLines: afterCallLines,
	}
}

func getGLibType(type0 string) string {
	if isSameNamespace("GLib") {
		return type0
	} else {
		addGirImport("GLib")
		return "glib." + type0
	}
}

func isSameNamespace(ns string) bool {
	if ns == optNamespace {
		return true
	}
	return false
}

func getTypeName(bi *gi.BaseInfo) string {
	ns := bi.Namespace()
	if isSameNamespace(ns) {
		return bi.Name()
	}

	pkgBase := ""
	for _, dep := range globalDeps {
		if strings.HasPrefix(dep, ns+"-") {
			pkgBase = strings.ToLower(dep)
			break
		}
	}

	typeName := strings.ToLower(ns) + "." + bi.Name()
	if pkgBase != "" {
		globalSourceFile.AddGirImport(pkgBase)
	}
	return typeName
}

func addGirImport(ns string) {
	pkgBase := ""
	for _, dep := range globalDeps {
		if strings.HasPrefix(dep, ns+"-") {
			pkgBase = strings.ToLower(dep)
			break
		}
	}
	if pkgBase != "" {
		globalSourceFile.AddGirImport(pkgBase)
	}
}

func getAllDeps(repo *gi.Repository, namespace string) []string {
	if namespace == "" {
		namespace = optNamespace
	}
	if strings.Contains(namespace, "-") {
		nameVer := strings.SplitN(namespace, "-", 2)
		namespace = nameVer[0]
		version := nameVer[1]
		_, err := repo.Require(namespace, version, gi.REPOSITORY_LOAD_FLAG_LAZY)
		if err != nil {
			log.Fatal(err)
		}
	}

	deps := repo.ImmediateDependencies(namespace)
	log.Printf("ns %s, deps %v\n", namespace, deps)
	if len(deps) == 0 {
		return nil
	}

	resultMap := make(map[string]struct{})
	for _, dep := range deps {
		resultMap[dep] = struct{}{}
	}
	for _, dep := range deps {
		deps0 := getAllDeps(repo, dep)
		for _, dep0 := range deps0 {
			resultMap[dep0] = struct{}{}
		}
	}
	keys := make([]string, 0, len(resultMap))
	for key := range resultMap {
		keys = append(keys, key)
	}
	return keys
}
