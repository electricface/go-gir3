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
