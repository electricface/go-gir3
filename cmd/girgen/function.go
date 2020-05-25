package main

import (
	"fmt"
	"strings"

	"github.com/electricface/go-gir3/gi"
)

var globalFuncNextIdx int

func pFunction(s *SourceFile, fi *gi.FunctionInfo) {
	symbol := fi.Symbol()
	s.GoBody.Pn("// %s", symbol)
	funcIdx := globalFuncNextIdx
	globalFuncNextIdx++

	fnName := fi.Name()

	// 函数内参数分配器
	var varReg VarReg
	// 目标函数形参列表
	var args []string
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
	var numArgOut int

	numArg := fi.NumArg()
	for i := 0; i < numArg; i++ {
		fiArg := fi.Arg(i)
		argTypeInfo := fiArg.Type()
		dir := fiArg.Direction()
		switch dir {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			numArgOut++
		}

		varArg := varReg.alloc(fiArg.Name())
		if dir == gi.DIRECTION_IN || dir == gi.DIRECTION_INOUT {
			// 作为 go 函数的输入参数之一

			type0 := "TODO_TYPE"
			if dir == gi.DIRECTION_IN {
				parseResult := parseArgTypeDirIn(varArg, argTypeInfo, &varReg)

				type0 = parseResult.type0
				beforeArgLines = append(beforeArgLines, parseResult.beforeArgLines...)

				varArg := varReg.alloc("arg_" + varArg)
				argNames = append(argNames, varArg)
				newArgLines = append(newArgLines, fmt.Sprintf("%s := %s", varArg, parseResult.newArg))

				afterCallLines = append(afterCallLines, parseResult.afterCallLines...)
			}

			args = append(args, varArg+" "+type0)
		} else if dir == gi.DIRECTION_OUT {
			// 作为 go 函数的返回值之一
			// TODO
		}

		argTypeInfo.Unref()
		fiArg.Unref()
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
		retParams = append(retParams, varResult+" "+parseRetTypeResult.type0)
	}

	fnFlags := fi.Flags()
	varErr := varReg.alloc("err")
	var isThrows bool
	if fnFlags&gi.FUNCTION_THROWS != 0 {
		// TODO: 需要把 **GError err 加入参数列表
		isThrows = true
		retParams = append(retParams, varErr+" error")
	}

	argsJoined := strings.Join(args, ", ")

	retParamsJoined := strings.Join(retParams, ", ")
	if len(retParams) > 0 {
		retParamsJoined = "(" + retParamsJoined + ")"
	}
	// 输出目标函数头部
	s.GoBody.Pn("func %s(%s) %s {", fnName, argsJoined, retParamsJoined)

	varInvoker := varReg.alloc("iv")
	s.GoBody.Pn("%s, %s := _I.Get(%d, %q, \"\")", varInvoker, varErr, funcIdx, fnName)

	{ // 处理 invoker 获取失败的情况

		s.GoBody.Pn("if err != nil {")

		if isThrows {
			// 使用 err 变量返回错误
		} else {
			// 把 err 打印出来
			s.GoBody.Pn("log.Println(\"WARN:\", err) /*go:log*/")
		}
		s.GoBody.Pn("return")

		s.GoBody.Pn("}") // end if err != nil
	}

	var varCMemArgs string
	if numArgOut > 0 {
		varCMemArgs = varReg.alloc("cma")
		s.GoBody.Pn("%v := gi.AllocArgs(%v)", varCMemArgs, numArgOut)
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
	s.GoBody.Pn("%s.Call(%s, %s)", varInvoker, callArgArgs, callArgRet)

	if !isRetVoid && parseRetTypeResult != nil {
		s.GoBody.Pn("%s = %s", varResult, parseRetTypeResult.expr)
	}

	for _, line := range afterCallLines {
		s.GoBody.Pn(line)
	}

	if numArgOut > 0 {
		s.GoBody.Pn("%v.Free()", varCMemArgs)
	}

	if !isRetVoid {
		s.GoBody.Pn("return")
	}

	s.GoBody.Pn("}") // end func
}

type parseRetTypeResult struct {
	expr  string // 转换 arguemnt 为返回值类型的表达式
	type0 string // 目标函数中返回值类型
}

func parseRetType(varRet string, ti *gi.TypeInfo, varReg *VarReg) *parseRetTypeResult {
	expr := ""
	type0 := ""
	tag := ti.Tag()
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
		expr = fmt.Sprintf("%s.%s()", varRet, getArgumentMethodPart(tag))
		type0 = getTypeTagType(tag)

	default:
		// 未知类型
		expr = varRet + ".TODO()"
		type0 = "TODO_TYPE"
	}

	return &parseRetTypeResult{
		expr:  expr,
		type0: type0,
	}
}

func parseArgTypeDirOut() {

}

func parseArgTypeDirInOut() {

}

type parseArgTypeDirInResult struct {
	newArg         string   // gi.NewArgument 用的
	type0          string   // go函数形参中的类型
	beforeArgLines []string // 在 arg_xxx = gi.NewXXXArgument 之前执行的语句
	afterCallLines []string // 在 invoker.Call() 之后执行的语句
}

// TODO 重命名它
func getTypeTagType(tag gi.TypeTag) (type0 string) {
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

func getArgumentMethodPart(tag gi.TypeTag) (str string) {
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

func parseArgTypeDirIn(varArg string, ti *gi.TypeInfo, varReg *VarReg) *parseArgTypeDirInResult {
	// 目前只考虑 direction 为 in 的情况
	var newArg string
	var beforeArgLines []string
	var afterCallLines []string
	var type0 string

	tag := ti.Tag()
	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		// 字符串类型
		// 产生类似如下代码：
		// pArg = gi.CString(arg)
		// arg = gi.NewStringArgument(pArg)
		// after call:
		// gi.Free(pArg)
		varCArg := varReg.alloc("c_" + varArg)
		beforeArgLines = append(beforeArgLines,
			fmt.Sprintf("%s := gi.CString(%s)", varCArg, varArg))
		newArg = fmt.Sprintf("gi.NewStringArgument(%s)", varCArg)
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

		middle := ""
		switch tag {
		case gi.TYPE_TAG_BOOLEAN:
			middle = "Bool"
		case gi.TYPE_TAG_INT8:
			middle = "Int8"
		case gi.TYPE_TAG_UINT8:
			middle = "Uint8"

		case gi.TYPE_TAG_INT16:
			middle = "Int16"
		case gi.TYPE_TAG_UINT16:
			middle = "Uint16"

		case gi.TYPE_TAG_INT32:
			middle = "Int32"
		case gi.TYPE_TAG_UINT32:
			middle = "Uint32"

		case gi.TYPE_TAG_INT64:
			middle = "Int64"
		case gi.TYPE_TAG_UINT64:
			middle = "Uint64"

		case gi.TYPE_TAG_FLOAT:
			middle = "Float"
		case gi.TYPE_TAG_DOUBLE:
			middle = "Double"
		}
		newArg = fmt.Sprintf("gi.New%sArgument(%s)", middle, varArg)

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

	default:
		// 未知类型
		type0 = "TODO_TYPE"
		newArg = fmt.Sprintf("gi.NewTODOArgument(%s)", varArg)
	}

	return &parseArgTypeDirInResult{
		newArg:         newArg,
		type0:          type0,
		beforeArgLines: beforeArgLines,
		afterCallLines: afterCallLines,
	}
}

/*
direction: in
作为参数
direction: out
作为返回值

direction: inout
作为参数，之后要把参数给修改了 *arg =
*/
