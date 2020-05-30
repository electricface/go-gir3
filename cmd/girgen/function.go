package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/electricface/go-gir3/gi"
)

// 给 InvokeCache.Get() 用的 index 的
var globalFuncNextIdx int

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

func pFunction(s *SourceFile, fi *gi.FunctionInfo) {
	symbol := fi.Symbol()
	fiName := fi.Name()
	// 用于黑名单识别函数的名字
	identifyName := fiName
	container := fi.Container()
	if container != nil {
		identifyName = container.Name() + "." + fiName
	}
	if strSliceContains(globalCfg.Black, identifyName) {
		s.GoBody.Pn("\n// black function %s\n", identifyName)
		return
	}

	s.GoBody.Pn("// %s", symbol)
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
		s.GoBody.Pn("// container is not nil, container is %s", container.Name())
		if fnFlags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
			// 表示 C 函数是构造器
			s.GoBody.Pn("// is constructor")
		} else if fnFlags&gi.FUNCTION_IS_METHOD != 0 {
			// 表示 C 函数是方法
			s.GoBody.Pn("// is method")
			addReceiver = true
		} else {
			// 可能 C 函数还是可以作为方法的，只不过没有处理好参数，如果第一个参数是指针类型，就大概率是方法。
			if fi.NumArg() > 0 {
				s.GoBody.Pn("// is method")
				arg0 := fi.Arg(0)
				arg0Type := arg0.Type()
				s.GoBody.Pn("// arg0Type tag: %v, isPtr: %v", arg0Type.Tag(), arg0Type.IsPointer())
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
				s.GoBody.Pn("// num arg is 0")
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
		s.GoBody.Pn("// container is nil")
	}

	numArg := fi.NumArg()
	for i := argIdxStart; i < numArg; i++ {
		fiArg := fi.Arg(i)
		argTypeInfo := fiArg.Type()
		dir := fiArg.Direction()
		switch dir {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			numOutArgs++
			if varOutArgs == "" {
				varOutArgs = varReg.alloc("outArgs")
			}
		}

		paramName := varReg.alloc(fiArg.Name())

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
			parseResult := parseArgTypeDirOut(argTypeInfo, &varReg)
			type0 := parseResult.type0
			retParams = append(retParams, paramName+" "+type0)

			varArg := varReg.alloc("arg_" + paramName)
			argNames = append(argNames, varArg)
			newArgLines = append(newArgLines, fmt.Sprintf("%v := gi.NewPointerArgument(unsafe.Pointer(&%v[%v]))", varArg, varOutArgs, outArgIdx))
			afterCallLines = append(afterCallLines, fmt.Sprintf("%v = %v[%v].%v", paramName, varOutArgs, outArgIdx, parseResult.expr))

			outArgIdx++
		}

		argTypeInfo.Unref()
		fiArg.Unref()
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

	retTypeInfo := fi.ReturnType()
	defer retTypeInfo.Unref()

	var varRet string
	var varResult string
	var parseRetTypeResult *parseRetTypeResult

	// 是否**无**返回值
	var isRetVoid bool
	if gi.TYPE_TAG_VOID == retTypeInfo.Tag() {
		// 无返回值
		isRetVoid = true
	} else {
		// 有返回值
		varRet = varReg.alloc("ret")
		varResult = varReg.alloc("result")
		parseRetTypeResult = parseRetType(varRet, retTypeInfo, &varReg)
		retParams = append([]string{varResult + " " + parseRetTypeResult.type0}, retParams...)
	}

	paramsJoined := strings.Join(params, ", ")

	retParamsJoined := strings.Join(retParams, ", ")
	if len(retParams) > 0 {
		retParamsJoined = "(" + retParamsJoined + ")"
	}
	// 输出目标函数头部
	s.GoBody.Pn("func %s %s(%s) %s {", receiver, fnName, paramsJoined, retParamsJoined)

	varInvoker := varReg.alloc("iv")
	if container == nil {
		s.GoBody.Pn("%s, %s := _I.Get(%d, %q, \"\")", varInvoker, varErr, funcIdx, fiName)
	} else {
		s.GoBody.Pn("%s, %s := _I.Get(%d, %q, %q)", varInvoker, varErr, funcIdx, container.Name(), fiName)
	}

	{ // 处理 invoker 获取失败的情况

		s.GoBody.Pn("if %s != nil {", varErr)

		if isThrows {
			// 使用 err 变量返回错误
		} else {
			// 把 err 打印出来
			s.GoBody.Pn("log.Println(\"WARN:\", %s) /*go:log*/", varErr)
		}
		s.GoBody.Pn("return")

		s.GoBody.Pn("}") // end if err != nil
	}

	if numOutArgs > 0 {
		s.GoBody.Pn("var %s [%d]gi.Argument", varOutArgs, numOutArgs)
	}

	for _, line := range beforeArgLines {
		s.GoBody.Pn(line)
	}

	for _, line := range newArgLines {
		s.GoBody.Pn(line)
	}

	callArgArgs := "nil"
	if len(argNames) > 0 {
		// 比如输出 args := []gi.Argument{arg0,arg1}
		varArgs := varReg.alloc("args")
		s.GoBody.Pn("%s := []gi.Argument{%s}", varArgs, strings.Join(argNames, ", "))
		callArgArgs = varArgs
	}

	callArgRet := "nil"
	if !isRetVoid {
		// 有返回值
		callArgRet = "&" + varRet
		s.GoBody.Pn("var %s gi.Argument", varRet)
	}
	callArgOutArgs := "nil"
	if numOutArgs > 0 {
		callArgOutArgs = fmt.Sprintf("&%s[0]", varOutArgs)
	}
	s.GoBody.Pn("%s.Call(%s, %s, %s)", varInvoker, callArgArgs, callArgRet, callArgOutArgs)

	if !isRetVoid && parseRetTypeResult != nil {
		s.GoBody.Pn("%s%s = %s", varResult, parseRetTypeResult.field, parseRetTypeResult.expr)
	}

	for _, line := range afterCallLines {
		s.GoBody.Pn(line)
	}

	if len(retParams) > 0 {
		s.GoBody.Pn("return")
	}

	s.GoBody.Pn("}") // end func
}

type parseRetTypeResult struct {
	expr  string // 转换 argument 为返回值类型的表达式
	field string // expr 要给 result 的什么字段设置，比如 .P 字段
	type0 string // 目标函数中返回值类型
}

func parseRetType(varRet string, ti *gi.TypeInfo, varReg *VarReg) *parseRetTypeResult {
	debugMsg := ""
	isPtr := ti.IsPointer()
	tag := ti.Tag()
	debugMsg = fmt.Sprintf("isPtr: %v, tag: %v", isPtr, tag)

	type0 := fmt.Sprintf("int/*TODO_TYPE %s*/", debugMsg)
	expr := varRet + ".Int()/*TODO*/"
	field := ""

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

	case gi.TYPE_TAG_INTERFACE:
		if isPtr {
			bi := ti.Interface()
			type0 = getTypeName(bi)
			expr = fmt.Sprintf("%s.Pointer()", varRet)
			field = ".P"

			bi.Unref()
		}
		// else 不是 pointer 的 interface 太奇怪了
	}

	return &parseRetTypeResult{
		field: field,
		expr:  expr,
		type0: type0,
	}
}

type parseArgTypeDirOutResult struct {
	expr  string // 转换 arguemnt 为返回值类型的表达式
	type0 string // 目标函数中返回值类型
}

func parseArgTypeDirOut(ti *gi.TypeInfo, varReg *VarReg) *parseArgTypeDirOutResult {
	expr := "Int()/*TODO*/"
	type0 := "int/*TODO_TYPE*/"
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

	case gi.TYPE_TAG_INTERFACE:
		// TODO

	}

	return &parseArgTypeDirOutResult{
		expr:  expr,
		type0: type0,
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
		newArgExpr = fmt.Sprintf("gi.New%sArgument(%s)", argType, varArg)
		type0 = getTypeWithTag(tag)

	case gi.TYPE_TAG_VOID:
		if isPtr {
			// ti 指的类型就是 void* , 翻译为 unsafe.Pointer
			type0 = "unsafe.Pointer"
			newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%s)", varArg)
		}

	case gi.TYPE_TAG_INTERFACE:
		if isPtr {
			bi := ti.Interface()
			type0 = getTypeName(bi)

			newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%s.P)", varArg)
			bi.Unref()
		}
	}

	return &parseArgTypeDirInResult{
		newArgExpr:     newArgExpr,
		type0:          type0,
		beforeArgLines: beforeArgLines,
		afterCallLines: afterCallLines,
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
		typeName += fmt.Sprintf("/*gir:%s*/", pkgBase)
	}
	return typeName
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
