package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_snake2Camel(t *testing.T) {
	ret := snake2Camel("a_bc_DEF")
	assert.Equal(t, "ABcDef", ret)
}

func TestVarRegAlloc(t *testing.T) {
	var varReg VarReg
	varArg := varReg.alloc("arg")
	assert.Equal(t, "arg", varArg)

	varArg = varReg.alloc("arg")
	assert.Equal(t, "arg1", varArg)

	varArg = varReg.alloc("arg")
	assert.Equal(t, "arg2", varArg)

	varArg = varReg.alloc("arg")
	assert.Equal(t, "arg3", varArg)

	// 测试关键字
	varArg = varReg.alloc("type")
	assert.Equal(t, "type1", varArg)

	varArg = varReg.alloc("type")
	assert.Equal(t, "type2", varArg)
}

func Test_getC(t *testing.T) {
	assert.Equal(t, "NewKeyFile", getConstructorName("KeyFile", "New"))
	assert.Equal(t, "NewDesktopAppInfoFromFilename",
		getConstructorName("DesktopAppInfo", "NewFromFilename"))
	assert.Equal(t, "KeyFileCreateWithPath", getConstructorName("KeyFile", "CreateWithPath"))
}
