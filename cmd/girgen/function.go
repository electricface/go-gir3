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
	"strconv"
	"strings"

	"github.com/electricface/go-gir3/gi"
)

// 给 InvokeCache.Get() 用的 index 的
var _funcNextIdx int

var _numTodoFunc int
var _numFunc int

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
	name := _symbolNameMap[symbol]
	if name != "" {
		return name
	}
	return getFunctionName(fi)
}

/*

{ // begin func

beforeNewArgLines

newArgLines

call

afterCallLines

setParamLines

beforeRetLines

return

} // end func

*/

type pFuncContext struct {
	fi        *gi.FunctionInfo
	container *gi.BaseInfo
	// 目标函数形参列表，元素是 "名字 类型"
	params []string
	// 目标函数返回参数列表，元素是 "名字 类型"
	retParams []string
	// 准备传递给 invoker.Call 中的参数的代码之前的语句, 在 newArgLines 之前的语句。
	beforeNewArgLines []string
	// 准备传递给 invoker.Call 中的参数的语句
	newArgLines []string
	// 传递给 invoker.Call 的参数列表
	argNames []string
	// 在 invoker.Call 执行后需要执行的语句
	afterCallLines []string
	// 设置生成 Go 函数的返回值变量的语句
	// setParamLine 类似 param1 = outArgs[1].Int(), 或 param1 = rune(outArgs[1].Uint32())
	// 或 param1.P = outArgs[1].Pointer()
	setParamLines []string
	// 在 return 返回之前的语句
	beforeRetLines []string
	// 函数开头的注释
	commentLines []string
	// 生成 go函数的名称
	fnName string
	// 函数接收者部分
	receiver string
	// 是否抛出错误，如果为 true，则 C 函数中最后一个参数是 **GError err
	isThrows bool
	// 是否 C 函数 **无** 返回值
	isRetVoid bool
	funcIdx   int
	idxLv1    int
	idxLv2    int
	// direction 为 inout 或 out 的参数个数
	numOutArgs int
	// 函数内变量名称分配器
	varReg VarReg
	// 必要变量名
	varRet     string
	varResult  string
	varErr     string
	varOutArgs string
}

func pFunction(s *SourceFile, fi *gi.FunctionInfo, idxLv1, idxLv2 int) {
	symbol := fi.Symbol()

	var ctx pFuncContext
	ctx.fi = fi
	ctx.idxLv1 = idxLv1
	ctx.idxLv2 = idxLv2
	ctx.commentLines = append(ctx.commentLines, symbol, "")
	ctx.fnName = getFunctionNameFinal(fi)
	ctx.funcIdx = _funcNextIdx
	// NOTE: 注意不要调用 container 的 Unref 方法，fi.Container() 没有转移所有权。
	ctx.container = fi.Container()

	_funcNextIdx++
	_numFunc++

	ctx.varErr = ctx.varReg.alloc("err")

	argIdxStart := ctx.pFuncReceiver()

	// lenArgMap 是数组长度参数的集合，键是长度参数的 index
	lenArgMap := make(map[int]struct{})

	// 键是 user_data 参数的索引，值是 callback 参数的索引，
	// 提供一个从 user_data 参数找到 callback 参数的信息。
	closureMap := make(map[int]int)

	numArgs := fi.NumArg()
	for argIdx := argIdxStart; argIdx < numArgs; argIdx++ {
		argInfo := fi.Arg(argIdx)
		paramName := ctx.varReg.registerParam(argIdx, argInfo.Name())

		paramComment := fmt.Sprintf("[ %v ] trans: %v", paramName, argInfo.OwnershipTransfer())
		dir := argInfo.Direction()
		if dir == gi.DIRECTION_OUT || dir == gi.DIRECTION_INOUT {
			paramComment += fmt.Sprintf(", dir: %v", dir)
		}
		ctx.commentLines = append(ctx.commentLines, paramComment, "")

		closureIdx := argInfo.Closure() // 该参数的 user_data 的参数的索引
		if closureIdx > 0 {
			if argIdxStart == 1 {
				closureIdx++
			}
			closureMap[closureIdx] = argIdx
		}

		argTypeInfo := argInfo.Type()
		typeTag := argTypeInfo.Tag()
		if typeTag == gi.TYPE_TAG_ARRAY {
			lenArgIdx := argTypeInfo.ArrayLength() // 是该数组类型参数的长度参数的 index
			if lenArgIdx >= 0 {
				lenArgMap[lenArgIdx] = struct{}{}
			}
		}

		argTypeInfo.Unref()
		argInfo.Unref()
	}

	retTypeInfo := fi.ReturnType()
	defer retTypeInfo.Unref()
	retTypeTag := retTypeInfo.Tag()
	if retTypeTag == gi.TYPE_TAG_ARRAY {
		// 返回值是数组类型
		lenArgIdx := retTypeInfo.ArrayLength()
		if lenArgIdx >= 0 {
			lenArgMap[lenArgIdx] = struct{}{}
		}
	}

	// 开始处理每个参数
	var outArgIdx int
	for argIdx := argIdxStart; argIdx < numArgs; argIdx++ {
		argInfo := fi.Arg(argIdx)
		argTypeInfo := argInfo.Type()
		dir := argInfo.Direction()
		isCallerAlloc := argInfo.IsCallerAllocates()

		switch dir {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			var asRet bool // 该参数是否作为生成 Go 函数的返回值
			if dir == gi.DIRECTION_INOUT {
				asRet = true
			} else {
				// dir out
				asRet = shouldArgAsReturn(argTypeInfo, isCallerAlloc)
			}

			if asRet {
				ctx.numOutArgs++
				if ctx.varOutArgs == "" {
					ctx.varOutArgs = ctx.varReg.alloc("outArgs")
				}
			}
		}

		paramName := ctx.varReg.getParam(argIdx)

		switch dir {
		case gi.DIRECTION_IN:
			// 处理方向为 in 的参数
			var callbackArgInfo *gi.ArgInfo
			if callbackIdx, ok := closureMap[argIdx]; ok {
				callbackArgInfo = fi.Arg(callbackIdx)
				// TODO 应该把 paramName 改成更好的名字
				paramName = ctx.varReg.alloc("fn")
			}

			ctx.pFuncArgDirIn(paramName, argTypeInfo, callbackArgInfo)
		case gi.DIRECTION_INOUT:
			// TODO：处理 dir 为 inout 的
			type0 := "int/*TODO:DIR_INOUT*/"
			ctx.params = append(ctx.params, paramName+" "+type0)
		case gi.DIRECTION_OUT:
			// 处理方向为 out 的参数
			// isArgLen 表示本参数是某个输出数组的长度
			_, isArgLen := lenArgMap[argIdx]
			ctx.pFuncArgDirOut(paramName, argInfo, isArgLen, &outArgIdx)
		}

		argTypeInfo.Unref()
		argInfo.Unref()
	}

	if fi.Flags()&gi.FUNCTION_THROWS != 0 {
		// C 函数会抛出错误
		ctx.isThrows = true
		ctx.numOutArgs++
		if ctx.varOutArgs == "" {
			ctx.varOutArgs = ctx.varReg.alloc("outArgs")
		}
		varArg := ctx.varReg.alloc("arg_" + ctx.varErr)
		ctx.argNames = append(ctx.argNames, varArg)
		ctx.newArgLines = append(ctx.newArgLines, fmt.Sprintf("%v := gi.NewPointerArgument(unsafe.Pointer(&%v[%v]))", varArg, ctx.varOutArgs, outArgIdx))
		ctx.afterCallLines = append(ctx.afterCallLines, fmt.Sprintf("%v = gi.ToError(%v[%v].%v)", ctx.varErr, ctx.varOutArgs, outArgIdx, "Pointer()"))
		ctx.retParams = append(ctx.retParams, ctx.varErr+" error")
	}

	ctx.pFuncRetType()

	b := &SourceBlock{}
	ctx.print(b)
	if b.containsTodo() { // 检查生成的代码里是否含有 TO-DO，如果有表示没处理好这个函数。
		_numTodoFunc++
	}
	s.GoBody.addBlock(b)
}

func (ctx *pFuncContext) pFuncReceiver() (argIdxStart int) {
	if ctx.container == nil {
		return
	}
	hasReceiver := false
	fi := ctx.fi
	fnFlags := fi.Flags()
	if fnFlags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
		// 表示 C 函数是构造器
	} else if fnFlags&gi.FUNCTION_IS_METHOD != 0 {
		// 表示 C 函数是方法
		hasReceiver = true
	} else {
		// 可能 C 函数还是可以作为方法的，只不过没有处理好参数，如果第一个参数是指针类型，就大概率是方法。
		if fi.NumArg() > 0 {
			arg0 := fi.Arg(0)
			arg0Type := arg0.Type()
			if arg0Type.IsPointer() && arg0Type.Tag() == gi.TYPE_TAG_INTERFACE {
				ii := arg0Type.Interface()
				if ii.Name() == ctx.container.Name() {
					hasReceiver = true
					// 从 1 开始
					argIdxStart = 1
				}
				ii.Unref()
			}

			if !hasReceiver {
				// 不能作为方法, 作为函数
				ctx.fnName = ctx.container.Name() + ctx.fnName + "1"
				// TODO: 适当消除 1 后缀
			}
		} else {
			// 比如 io_channel_error_quark 方法，被重命名为IOChannel.error_quark，这算是 IOChannel 的 static 方法，
			ctx.fnName = ctx.container.Name() + ctx.fnName + "1"
		}
	}

	if hasReceiver {
		// 容器是否是 interface 类型的
		isContainerIfc := false
		if ctx.container.Type() == gi.INFO_TYPE_INTERFACE {
			isContainerIfc = true
		}

		receiverType := ctx.container.Name()
		if isContainerIfc {
			receiverType = "*" + receiverType + "Ifc"
		}

		varV := ctx.varReg.alloc("v")
		ctx.receiver = fmt.Sprintf("(%s %s)", varV, receiverType)
		varArgV := ctx.varReg.alloc("arg_v")
		getPtrExpr := fmt.Sprintf("%s.P", varV)
		if isContainerIfc {
			getPtrExpr = fmt.Sprintf("*(*unsafe.Pointer)(unsafe.Pointer(%v))", varV)
		}
		ctx.newArgLines = append(ctx.newArgLines, fmt.Sprintf("%v := gi.NewPointerArgument(%s)",
			varArgV, getPtrExpr))
		ctx.argNames = append(ctx.argNames, varArgV)
	}
	return
}

func (ctx *pFuncContext) pFuncArgDirOut(paramName string, argInfo *gi.ArgInfo, isArgLen bool, outArgIdx *int) {
	isCallerAlloc := argInfo.IsCallerAllocates()
	argTypeInfo := argInfo.Type()
	defer argTypeInfo.Unref()
	parseResult := parseArgTypeDirOut(paramName, argTypeInfo, &ctx.varReg, isCallerAlloc,
		argInfo.OwnershipTransfer())
	type0 := parseResult.type0
	if isArgLen {
		ctx.afterCallLines = append(ctx.afterCallLines,
			fmt.Sprintf("var %v %v; _ = %v", paramName, type0, paramName))
		// 加上 _ = % 是为了防止编译报错， 目前 pango 包还有问题
		//# github.com/linuxdeepin/go-gir/pango-1.0
		//pango-1.0/pango_auto.go:3633:6: num_scripts declared but not used
		//pango-1.0/pango_auto.go:4180:6: n_attrs declared but not used
		//pango-1.0/pango_auto.go:4204:6: n_attrs declared but not used
		//pango-1.0/pango_auto.go:7226:6: n_families declared but not used
	} else if parseResult.isRet {
		// 作为目标函数的返回值之一
		ctx.retParams = append(ctx.retParams, paramName+" "+type0)
	}

	varArg := ctx.varReg.alloc("arg_" + paramName)
	ctx.argNames = append(ctx.argNames, varArg)

	if parseResult.isRet {
		ctx.newArgLines = append(ctx.newArgLines,
			fmt.Sprintf("%v := gi.NewPointerArgument(unsafe.Pointer(&%v[%v]))",
				varArg, ctx.varOutArgs, *outArgIdx))
		getValExpr := fmt.Sprintf("%v[%v].%v", ctx.varOutArgs, *outArgIdx, parseResult.expr)

		setParamLine := fmt.Sprintf("%v%v = %v",
			paramName, parseResult.field, getValExpr)

		if parseResult.needTypeCast { // 如果需要加上类型转换
			setParamLine = fmt.Sprintf("%v%v = %v(%s)",
				paramName, parseResult.field, type0, getValExpr)
		}

		ctx.setParamLines = append(ctx.setParamLines, setParamLine)
		*outArgIdx++
	} else {
		// out 类型的参数，但依旧作为生成 Go 函数的参数，一定是指针类型
		ctx.params = append(ctx.params, paramName+" "+parseResult.type0)
		ctx.newArgLines = append(ctx.newArgLines,
			fmt.Sprintf("%v := gi.NewPointerArgument(%v)", varArg, parseResult.expr))
	}

	ctx.beforeRetLines = append(ctx.beforeRetLines, parseResult.beforeRetLines...)
}

func (ctx *pFuncContext) pFuncArgDirIn(paramName string, argTypeInfo *gi.TypeInfo, callbackArgInfo *gi.ArgInfo) {
	parseResult := parseArgTypeDirIn(paramName, argTypeInfo, &ctx.varReg, callbackArgInfo)

	if callbackArgInfo != nil {
		callbackArgInfo.Unref()
	}

	type0 := parseResult.type0
	ctx.beforeNewArgLines = append(ctx.beforeNewArgLines, parseResult.beforeArgLines...)

	varArg := ctx.varReg.alloc("arg_" + paramName)
	ctx.argNames = append(ctx.argNames, varArg)
	ctx.newArgLines = append(ctx.newArgLines, fmt.Sprintf("%v := %v", varArg, parseResult.newArgExpr))

	ctx.afterCallLines = append(ctx.afterCallLines, parseResult.afterCallLines...)
	if type0 != "" {
		// 如果需要隐藏参数，则把它的类型设置为空。
		ctx.params = append(ctx.params, paramName+" "+type0)
	}
}

func (ctx *pFuncContext) pFuncRetType() {
	fi := ctx.fi
	retTypeInfo := fi.ReturnType()
	defer retTypeInfo.Unref()

	if gi.TYPE_TAG_VOID == retTypeInfo.Tag() && !retTypeInfo.IsPointer() {
		// 无返回值
		ctx.isRetVoid = true
	} else {
		// 有返回值
		ctx.varRet = ctx.varReg.alloc("ret")
		ctx.varResult = ctx.varReg.alloc("result")
		parseRetTypeResult := parseRetType(ctx.varRet, retTypeInfo, &ctx.varReg, fi, fi.CallerOwns())
		// 把返回值加在 retParams 列表最前面
		ctx.retParams = append([]string{ctx.varResult + " " + parseRetTypeResult.type0}, ctx.retParams...)

		ctx.commentLines = append(ctx.commentLines, fmt.Sprintf(
			"[ %v ] trans: %v", ctx.varResult, fi.CallerOwns()), "")

		// 设置返回值 result
		ctx.beforeRetLines = append(ctx.beforeRetLines,
			fmt.Sprintf("%s%s = %s", ctx.varResult, parseRetTypeResult.field, parseRetTypeResult.expr))
		if parseRetTypeResult.zeroTerm {
			ctx.beforeRetLines = append(ctx.beforeRetLines, fmt.Sprintf("%v.SetLenZT()", ctx.varResult))
		}
	}
}

func (ctx *pFuncContext) print(b *SourceBlock) {
	// 用于黑名单识别函数的名字
	identifyName := ctx.fnName
	if ctx.container != nil {
		identifyName = ctx.container.Name() + "." + ctx.fnName
	}

	if strSliceContains(_cfg.Black, identifyName) {
		b.Pn("\n// black function %s\n", identifyName)
		return
	}

	// 目标函数为生成的 Go 函数
	// 输出目标函数前面的注释文档
	if ctx.fi.IsDeprecated() {
		b.Pn("// Deprecated\n//")
	}
	for _, line := range ctx.commentLines {
		b.Pn("// %v", line)
	}

	// 输出目标函数头部
	paramsJoined := strings.Join(ctx.params, ", ")
	retParamsJoined := strings.Join(ctx.retParams, ", ")
	if len(ctx.retParams) > 0 {
		retParamsJoined = "(" + retParamsJoined + ")"
	}
	b.Pn("func %s %s(%s) %s {", ctx.receiver, ctx.fnName, paramsJoined, retParamsJoined)

	ctx.printBody(b)
	b.Pn("}") // end func
}

// 输出目标函数的实现 body
func (ctx *pFuncContext) printBody(b *SourceBlock) {
	varInvoker := ctx.varReg.alloc("iv")

	ctx.printInvokerGet(b, varInvoker)

	if ctx.numOutArgs > 0 {
		b.Pn("var %s [%d]gi.Argument", ctx.varOutArgs, ctx.numOutArgs)
	}

	for _, line := range ctx.beforeNewArgLines {
		b.Pn(line)
	}

	for _, line := range ctx.newArgLines {
		b.Pn(line)
	}

	ctx.printInvokerCall(b, varInvoker)

	for _, line := range ctx.afterCallLines {
		b.Pn(line)
	}

	for _, line := range ctx.setParamLines {
		b.Pn(line)
	}

	for _, line := range ctx.beforeRetLines {
		b.Pn(line)
	}

	if len(ctx.retParams) > 0 {
		b.Pn("return")
	}
}

// 输出对 invoker.Call 的调用
func (ctx *pFuncContext) printInvokerCall(b *SourceBlock, varInvoker string) {
	callArgArgs := "nil" // 用于 iv.Call 的第一个参数，由它传入 C 函数的所有参数
	if len(ctx.argNames) > 0 {
		// 比如输出 args := []gi.Argument{arg0,arg1}
		varArgs := ctx.varReg.alloc("args")
		b.Pn("%s := []gi.Argument{%s}", varArgs, strings.Join(ctx.argNames, ", "))
		callArgArgs = varArgs
	}

	callArgRet := "nil" // 用于 iv.Call 的第二个参数，由它传入 C 函数的返回值
	if !ctx.isRetVoid {
		// 有返回值
		callArgRet = "&" + ctx.varRet
		b.Pn("var %s gi.Argument", ctx.varRet)
	}
	callArgOutArgs := "nil" // 用于 iv.Call 的第三个参数
	if ctx.numOutArgs > 0 {
		callArgOutArgs = fmt.Sprintf("&%s[0]", ctx.varOutArgs)
	}
	b.Pn("%s.Call(%s, %s, %s)", varInvoker, callArgArgs, callArgRet, callArgOutArgs)
}

// 输出对 invoker.Get 的调用
func (ctx *pFuncContext) printInvokerGet(b *SourceBlock, varInvoker string) {
	useGet1 := false
	if _optNamespace == "GObject" || _optNamespace == "Gio" {
		useGet1 = true
	}

	// Get1(id uint, ns, nameLv1, nameLv2 string, idxLv1, idxLv2 int, infoType InfoType, flags FindMethodFlags)
	//id: funcIdx
	// ns: quote _optNamespace
	// nameLv1: quote fiName | quote container.Name()
	// nameLv2: "" | quote fiName
	// idxLv1: idxLv1
	// idxLv2: idxLv2
	// infoType: gi.INFO_TYPE_FUNCTION | gi.INFO_TYPE_XX (XX is STRUCT,UNION,OBJECT,INTERFACE)
	// flags: 0 or gi.FindMethodNoCallFind
	getArgs := []interface{}{ctx.funcIdx} // id
	if useGet1 {
		// Get1 比 Get 多了一个 ns 参数。
		getArgs = append(getArgs, strconv.Quote(_optNamespace)) // ns
	}
	// 处理 nameLv1, nameLv2 参数
	fiName := ctx.fi.Name()
	if ctx.container == nil {
		getArgs = append(getArgs, strconv.Quote(fiName)) // nameLv1
		getArgs = append(getArgs, `""`)                  // nameLv2
	} else {
		getArgs = append(getArgs, strconv.Quote(ctx.container.Name())) // nameLv1
		getArgs = append(getArgs, strconv.Quote(fiName))               // nameLv2
	}

	getArgs = append(getArgs, ctx.idxLv1) // idxLv1
	getArgs = append(getArgs, ctx.idxLv2) // idxLv2

	// 处理 infoType 参数
	infoType := "FUNCTION"
	if ctx.container != nil {
		switch ctx.container.Type() {
		case gi.INFO_TYPE_STRUCT:
			infoType = "STRUCT"
		case gi.INFO_TYPE_UNION:
			infoType = "UNION"
		case gi.INFO_TYPE_OBJECT:
			infoType = "OBJECT"
		case gi.INFO_TYPE_INTERFACE:
			infoType = "INTERFACE"
		}
	}
	getArgs = append(getArgs, "gi.INFO_TYPE_"+infoType) // infoType

	// 处理 flags 参数
	findMethodFlags := "0"
	if _optNamespace == "GObject" && ctx.container != nil && ctx.container.Name() == "ObjectClass" {
		// 因为调用 StructInfo.FindMethod 方法去查找 GObject.ObjectClass 的方法会导致崩溃，所以加上这个 flag 来规避。
		findMethodFlags = "gi.FindMethodNoCallFind"
	}
	getArgs = append(getArgs, findMethodFlags) // flags

	// 输出 _I.Get 调用
	b.P("%v, %v := _I.Get", varInvoker, ctx.varErr)
	if useGet1 {
		b.P("1")
	}
	getArgsStr := make([]string, len(getArgs))
	for i, v := range getArgs {
		getArgsStr[i] = fmt.Sprintf("%v", v)
	}
	b.Pn("(%v)", strings.Join(getArgsStr, ", "))

	{ // 处理 invoker 获取失败的情况

		b.Pn("if %s != nil {", ctx.varErr)

		if ctx.isThrows {
			// 使用 err 变量返回错误
		} else {
			// 把 err 打印出来
			b.Pn("log.Println(\"WARN:\", %s)", ctx.varErr)
		}
		b.Pn("return")

		b.Pn("}") // end if err != nil
	}
}

type parseRetTypeResult struct {
	expr     string // 转换 argument 为返回值类型的表达式
	field    string // expr 要给 result 的什么字段设置，比如 .P 字段
	type0    string // 目标函数中返回值类型
	zeroTerm bool
}

func parseRetType(varRet string, ti *gi.TypeInfo, varReg *VarReg, fi *gi.FunctionInfo,
	transfer gi.Transfer) *parseRetTypeResult {

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
		expr = varRet + ".String()"
		if transfer == gi.TRANSFER_NOTHING {
			expr += ".Copy()"
		} else {
			expr += ".Take()"
		}
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

			if fiFlags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
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

	case gi.TYPE_TAG_GLIST:
		type0 = getGLibType("List")
		expr = fmt.Sprintf("%v.Pointer()", varRet)
		field = ".P"

	case gi.TYPE_TAG_GSLIST:
		type0 = getGLibType("SList")
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
	isRet          bool // 是否作为返回值
}

func shouldArgAsReturn(ti *gi.TypeInfo, isCallerAlloc bool) bool {
	result := true
	tag := ti.Tag()
	switch tag {
	case gi.TYPE_TAG_INTERFACE:
		bi := ti.Interface()
		biType := bi.Type()
		if isCallerAlloc {
			if biType == gi.INFO_TYPE_STRUCT {
				result = false
			}
		}
		bi.Unref()

	case gi.TYPE_TAG_ARRAY:
		if isCallerAlloc {
			result = false
		}
	}
	return result
}

func parseArgTypeDirOut(paramName string, ti *gi.TypeInfo, varReg *VarReg,
	isCallerAlloc bool, transfer gi.Transfer) *parseArgTypeDirOutResult {

	tag := ti.Tag()

	expr := "Int()/*TODO*/"
	type0 := getDebugType("tag: %v", tag)
	needTypeCast := false
	field := ""
	isRet := true
	var beforeRetLines []string

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		// 字符串类型
		// 产生类似如下代码：
		// outArg1 = &outArgs[0].String().Take()
		//                       ^--------------
		expr = "String()"
		if transfer == gi.TRANSFER_NOTHING {
			expr += ".Copy()"
		} else {
			expr += ".Take()"
		}
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
		bi := ti.Interface()
		defer bi.Unref()
		biType := bi.Type()
		isPtr := ti.IsPointer()

		type0 = getDebugType("tag: ifc, biType: %v, callerAlloc: %v, isPtr: %v", biType, isCallerAlloc, isPtr)
		if isPtr && !isCallerAlloc {
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
			} else if biType == gi.INFO_TYPE_STRUCT {
				if isCallerAlloc {
					isRet = false
					type0 = getTypeName(bi)
					expr = paramName + ".P"
				} else {
					type0 = getTypeName(bi)
					field = ".P"
					expr = "Pointer()"
				}
			}
		}

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

	case gi.TYPE_TAG_GLIST:
		type0 = getGLibType("List")
		expr = "Pointer()"
		field = ".P"

	case gi.TYPE_TAG_GSLIST:
		type0 = getGLibType("SList")
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

		if isCallerAlloc {
			isRet = false
			// type
			// expr 用于 newArgExpr， arg_param := gi.NewPointerArgument($expr)
			type0 = "unsafe.Pointer /*TODO:TYPE*/"
			expr = paramName + "/*TODO*/"

			if arrType == gi.ARRAY_TYPE_C {
				elemTypeInfo := ti.ParamType(0)
				elemTypeTag := elemTypeInfo.Tag()
				type0 = fmt.Sprintf("unsafe.Pointer /*TODO array type c, elemTypeTag: %v*/", elemTypeTag)
				elemType := getArgumentType(elemTypeTag)
				if elemType != "" && !elemTypeInfo.IsPointer() {
					type0 = "gi." + elemType + "Array"
					expr = paramName + ".P"
				} else if elemTypeTag == gi.TYPE_TAG_UTF8 || elemTypeTag == gi.TYPE_TAG_FILENAME {
					type0 = "gi.CStrArray"
					expr = paramName + ".P"
				} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && elemTypeInfo.IsPointer() {
					type0 = "gi.PointerArray"
					expr = paramName + ".P"
				} else if elemTypeTag == gi.TYPE_TAG_INTERFACE && !elemTypeInfo.IsPointer() {
					type0 = "unsafe.Pointer"
					expr = paramName
				}

				elemTypeInfo.Unref()
			}

		} else {
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

	}

	return &parseArgTypeDirOutResult{
		expr:           expr,
		type0:          type0,
		needTypeCast:   needTypeCast,
		field:          field,
		beforeRetLines: beforeRetLines,
		isRet:          isRet,
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

	case gi.TYPE_TAG_GTYPE:
		str = "GType"
	}
	return
}

// addPrefixIForType 给类型加上 I 前缀，换成接口类型。
func addPrefixIForType(type0 string) string {
	if strings.Contains(type0, ".") {
		// gobject.Object => gobject.IObject
		type0 = strings.Replace(type0, ".", ".I", 1)
	} else {
		// Object => IObject
		type0 = "I" + type0
	}
	return type0
}

type parseArgTypeDirInResult struct {
	newArgExpr     string   // 创建 Argument 的表达式，比如 gi.NewIntArgument()
	type0          string   // 目标函数形参中的类型
	beforeArgLines []string // 在 arg_xxx = gi.NewXXXArgument 之前执行的语句
	afterCallLines []string // 在 invoker.Call() 之后执行的语句
}

func parseArgTypeDirIn(varArg string, ti *gi.TypeInfo, varReg *VarReg, callbackArgInfo *gi.ArgInfo) *parseArgTypeDirInResult {
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

			if callbackArgInfo != nil {
				type0 = "func(v interface{})"
				varCId := varReg.alloc("cId")

				scopeType := callbackArgInfo.Scope()
				scope := "0"
				switch scopeType {
				case gi.SCOPE_TYPE_ASYNC:
					scope = "gi.ScopeAsync"
				case gi.SCOPE_TYPE_CALL:
					scope = "gi.ScopeCall"
				case gi.SCOPE_TYPE_NOTIFIED:
					scope = "gi.ScopeNotified"
				}

				beforeArgLines = append(beforeArgLines, fmt.Sprintf("%v := gi.RegisterFunc(%v, %v)", varCId, varArg, scope))
				newArgExpr = fmt.Sprintf("gi.NewPointerArgumentU(%v)", varCId)
				// NOTE: 当 scope 为 async 时不能在 iv.Call 调用返回之后立即 unregister func，因为异步函数会立即返回，然后立即取消注册，应该延后到异步回调函数
				// 运行之后再 unregister func。
				if scopeType == gi.SCOPE_TYPE_CALL {
					afterCallLines = append(afterCallLines, fmt.Sprintf("gi.UnregisterFunc(%v)", varCId))
				}
			}
		}

	case gi.TYPE_TAG_INTERFACE:
		bi := ti.Interface()
		biType := bi.Type()
		if isPtr {
			type0 = getTypeName(bi)
			newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%s.P)", varArg)

			biType := bi.Type()
			if biType == gi.INFO_TYPE_OBJECT || biType == gi.INFO_TYPE_INTERFACE {
				type0 = addPrefixIForType(type0)

				// 生成检查接口变量是否为 nil 的代码。如果不处理接口变量为 nil, 那么如果接口变量为 nil，
				// 则会导致 $varArg.P_XXX() 这里 panic。
				varTmp := varReg.alloc("tmp")
				beforeArgLines = append(beforeArgLines,
					fmt.Sprintf("var %v unsafe.Pointer", varTmp),
					fmt.Sprintf("if %v != nil {", varArg),
					fmt.Sprintf("%v = %v.P_%v()", varTmp, varArg, bi.Name()),
					"}", // end if
				)
				newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v)", varTmp)
			}

		} else {
			debugMsg = fmt.Sprintf("isPtr: %v, tag: %v, biType: %v", isPtr, tag, biType)
			type0 = fmt.Sprintf("int/*TODO_TYPE %s*/", debugMsg)

			if biType == gi.INFO_TYPE_FLAGS {
				type0 = getFlagsTypeName(getTypeName(bi))
				newArgExpr = fmt.Sprintf("gi.NewIntArgument(int(%v))", varArg)
			} else if biType == gi.INFO_TYPE_ENUM {
				type0 = getEnumTypeName(getTypeName(bi))
				newArgExpr = fmt.Sprintf("gi.NewIntArgument(int(%v))", varArg)
			} else if biType == gi.INFO_TYPE_CALLBACK {
				type0 = "" // 隐藏此参数，不出现在目标函数的参数列表中
				newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%vGetPointer_my%v())",
					getPkgPrefix(bi.Namespace()), bi.Name())
			} else if biType == gi.INFO_TYPE_UNRESOLVED {
				// 如果发现此种未解析的类型，应该使用黑名单屏蔽。
				type0 = "int /* TYPE_UNRESOLVED */"
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

	case gi.TYPE_TAG_GLIST:
		type0 = getGLibType("List")
		newArgExpr = fmt.Sprintf("gi.NewPointerArgument(%v.P)", varArg)

	case gi.TYPE_TAG_GSLIST:
		type0 = getGLibType("SList")
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
		return "g." + type0
	}
}

// 判断命名空间 ns 是否和当前命名空间一样。
func isSameNamespace(ns string) bool {
	if ns == _optNamespace {
		return true
	}
	switch _optNamespace {
	case "GObject":
		// gobject 有依赖 glib
		if ns == "GLib" {
			return true
		}
	case "Gio":
		// gio 有依赖 glib 和 gobject
		if ns == "GLib" || ns == "GObject" {
			return true
		}
	}
	return false
}

func getTypeName(bi *gi.BaseInfo) string {
	pkgPrefix := getPkgPrefix(bi.Namespace())
	return pkgPrefix + bi.Name()
}

// 根据命名空间 ns （不含版本）获取Go语言包的前缀，比如 ns 为 Gtk， 结果为 "gtk."。
// 还有自动导入 ns 指代的 Go语言包功能。
func getPkgPrefix(ns string) string {
	if isSameNamespace(ns) {
		return ""
	}
	pkgBase := ""
	for _, dep := range _deps {
		if strings.HasPrefix(dep, ns+"-") {
			pkgBase = strings.ToLower(dep)
			break
		}
	}

	ret := strings.ToLower(ns) + "."
	if ret == "glib." || ret == "gobject." || ret == "gio." {
		ret = "g."
	}
	if pkgBase != "" {
		_sourceFile.AddGirImport(pkgBase)
	}
	return ret
}

// 给生成源代码的 import 部分加上 github.com/linuxdeepin/go-gir/g-2.0 这样的包。
func addGirImport(ns string) {
	pkgBase := ""
	for _, dep := range _deps {
		if strings.HasPrefix(dep, ns+"-") {
			pkgBase = strings.ToLower(dep)
			break
		}
	}
	if pkgBase != "" {
		_sourceFile.AddGirImport(pkgBase)
	}
}

// 获取 namespace 的所有依赖，比如 Atk-1.0 的依赖是 GObject-2.0 和 GLib-2.0。
func getAllDeps(repo *gi.Repository, namespace string) []string {
	if namespace == "" {
		namespace = _optNamespace
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

	// 收集结果并去重
	resultMap := make(map[string]struct{})
	for _, dep := range deps {
		resultMap[dep] = struct{}{}
	}
	for _, dep := range deps {
		// 递归获取每个依赖 dep 的所有依赖，然后加入 resultMap
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
