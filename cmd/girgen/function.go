package main

import (
	"fmt"
	"github.com/electricface/go-gir3/gi"
	"strings"
)

func pFunction(s *SourceFile, fi *gi.FunctionInfo) {
	symbol := fi.Symbol()
	s.GoBody.Pn("// %s", symbol)
	symbols = append(symbols, symbol)

	fnName := fi.Name()

	// 函数内参数分配器
	var varReg VarReg
	// 函数形参列表
	var args []string
	// 函数返回类型列表 TODO: 可能需要扩展包含变量名称
	var retTypes []string

	// 准备传递给 invoker.Call 中的参数的代码之前的语句
	var beforeArgLines []string
	// 准备传递给 invoker.Call 中的参数的语句
	var newArgLines []string
	// 传递给 invoker.Call 中的参数列表
	var argNames []string

	// 在 invoker.Call 执行后需要执行的语句
	var afterCallLines []string

	numArg := fi.NumArg()
	for i := 0; i < numArg; i++ {
		fiArg := fi.Arg(i)
		argTypeInfo := fiArg.Type()
		dir := fiArg.Direction()

		varArg := varReg.alloc(fiArg.Name())
		if dir == gi.DIRECTION_IN || dir == gi.DIRECTION_INOUT {
			// 作为 go 函数的输入参数之一

			type0 := "TODO_TYPE"
			if dir == gi.DIRECTION_IN {
				parseResult := parseArgType(varArg, argTypeInfo, &varReg)

				type0 = parseResult.type0
				beforeArgLines = append(beforeArgLines, parseResult.beforeArgLines...)

				varArg := varReg.alloc("arg_"+ varArg)
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
	// 是否返回空
	var isRetVoid bool
	if gi.TYPE_TAG_VOID == retTypeInfo.Tag() {
		isRetVoid = true
	} else {
		retTypes = append(retTypes, "TODO_RET_TYPE")
	}

	fnFlags := fi.Flags()
	if fnFlags&gi.FUNCTION_THROWS != 0 {
		// 需要把 **GError err 加入参数列表，需要返回 error
		retTypes = append(retTypes, "error")
	}

	argsJoined := strings.Join(args, ", ")
	retTypesJoined := strings.Join(retTypes, ", ")
	if len(retTypes) > 1 {
		retTypesJoined = "(" + retTypesJoined + ")"
	}
	s.GoBody.Pn("func %s(%s) %s {", fnName, argsJoined, retTypesJoined)

	s.GoBody.Pn("invoker, err := invokerCache.Get(%s, %q, \"\")", symbol, fnName)
	s.GoBody.Pn("if err != nil {")

	// TODO
	s.GoBody.Pn("log.Fatal(err)")

	s.GoBody.Pn("}") // end if err != nil

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

	var varRet string
	callArgRet := "nil"
	if !isRetVoid {
		// 有返回值
		varRet = varReg.alloc("ret")
		callArgRet = "&" + varRet
		s.GoBody.Pn("var %s gi.Argument", varRet)
	}
	s.GoBody.Pn("invoker.Call(%s, %s)", callArgArgs, callArgRet)

	for _, line := range afterCallLines {
		s.GoBody.Pn(line)
	}

	s.GoBody.Pn("}") // end func
}

var goKeywords = []string{
	// keywords:
	"break", "default", "func", "interface", "select",
	"case", "defer", "go", "map", "struct",
	"chan", "else", "goto", "package", "switch",
	"const", "fallthrough", "if", "range", "type",
	"continue", "for", "import", "return", "var",

	// funcs:
	"append", "cap", "close", "complex", "copy", "delete", "imag",
	"len", "make", "new", "panic", "print", "println", "real", "recover",
}

var goKeywordMap map[string]struct{}

func init() {
	goKeywordMap = make(map[string]struct{})
	for _, kw := range goKeywords {
		goKeywordMap[kw] = struct{}{}
	}
}

type VarReg struct {
	vars []varNameIdx
}

type varNameIdx struct {
	name string
	idx int
}

func (vr *VarReg) alloc(prefix string) string {
	var found bool
	newVarIdx  := 0
	if len(vr.vars) > 0 {
		for i := len(vr.vars) - 1; i >=0; i-- {
			// 从尾部开始查找
			nameIdx := vr.vars[i]
			if prefix == nameIdx.name {
				found = true
				newVarIdx = nameIdx.idx + 1
				break
			}
		}
	}
	if !found {
		// try keyword
		_, ok := goKeywordMap[prefix]
		if ok {
			// 和关键字重名了
			newVarIdx = 1
		}
	}
	nameIdx := varNameIdx{name: prefix, idx: newVarIdx}
	vr.vars = append(vr.vars, nameIdx)
	return nameIdx.String()
}


func (v varNameIdx) String() string {
	if v.idx == 0 {
		return v.name
	}
	// TODO 可能需要处理 v.name 以数字结尾的情况
	return fmt.Sprintf("%s%d", v.name, v.idx)
}

type parseArgTypeResult struct {
	newArg string // gi.NewArgument 用的
	type0  string // go函数形参中的类型
	beforeArgLines []string // 在 arg_xxx = gi.NewXXXArgument 之前执行的语句
	afterCallLines []string  // 在 invoker.Call() 之后执行的语句
}

func parseArgType(varArg string, ti *gi.TypeInfo, varReg *VarReg) *parseArgTypeResult {
	//dir := arg.Direction()
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
		varPArg := varReg.alloc("p_" + varArg)
		beforeArgLines = append(beforeArgLines,
			fmt.Sprintf("%s := gi.CString(%s)", varPArg, varArg))
		newArg = fmt.Sprintf("gi.NewStringArgument(%s)", varPArg)
		afterCallLines = append(afterCallLines,
			fmt.Sprintf("gi.Free(%s)", varPArg))
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

	return &parseArgTypeResult{
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
