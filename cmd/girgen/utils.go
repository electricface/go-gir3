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
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

func strSliceContains(slice []string, str string) bool {
	for _, v := range slice {
		if str == v {
			return true
		}
	}
	return false
}

// CamelCase to snake_case
func camel2Snake(name string) string {
	var buf bytes.Buffer
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i != 0 {
				buf.WriteByte('_')
			}
			buf.WriteRune(unicode.ToLower(r))
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// snake_case to CamelCase
func snake2Camel(name string) string {
	//name = strings.ToLower(name)
	var out bytes.Buffer
	for _, word := range strings.Split(name, "_") {
		word = strings.ToLower(word)
		//if subst, ok := config.word_subst[word]; ok {
		//out.WriteString(subst)
		//continue
		//}

		if word == "" {
			out.WriteString("_")
			continue
		}
		out.WriteString(strings.ToUpper(word[0:1]))
		out.WriteString(word[1:])
	}
	return out.String()
}

var _keywords = []string{
	// Go 语言关键字:
	"break", "default", "func", "interface", "select",
	"case", "defer", "go", "map", "struct",
	"chan", "else", "goto", "package", "switch",
	"const", "fallthrough", "if", "range", "type",
	"continue", "for", "import", "return", "var",

	// Go 语言内建函数:
	"append", "cap", "close", "complex", "copy", "delete", "imag",
	"len", "make", "new", "panic", "print", "println", "real", "recover",

	// 全局变量
	"_I",
}

var _keywordsMap map[string]struct{}

func init() {
	_keywordsMap = make(map[string]struct{})
	for _, kw := range _keywords {
		_keywordsMap[kw] = struct{}{}
	}
}

type VarReg struct {
	vars     []varNameIdx
	paramMap map[int]string
}

type varNameIdx struct {
	name string
	idx  int
}

func (vr *VarReg) regParam(idx int, name string) string {
	if vr.paramMap == nil {
		vr.paramMap = make(map[int]string)
	}
	name = vr.alloc(name)
	vr.paramMap[idx] = name
	return name
}

func (vr *VarReg) getParam(idx int) string {
	return vr.paramMap[idx]
}

func (vr *VarReg) alloc(prefix string) string {
	var found bool
	newVarIdx := 0
	if len(vr.vars) > 0 {
		for i := len(vr.vars) - 1; i >= 0; i-- {
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
		_, ok := _keywordsMap[prefix]
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

func getConstructorName(containerName, fnName string) string {
	if fnName == "New" {
		return "New" + containerName
	}
	if strings.HasPrefix(fnName, "New") {
		// DesktopAppInfo.NewFromFilename  => NewDesktopAppInfoFromFilename
		return "New" + containerName + strings.TrimPrefix(fnName, "New")
	}
	// DesktopAppInfo.CreateWithPath => DesktopAppInfoCreateWithPath
	return containerName + fnName
}

func markDeprecated(s *SourceFile) {
	s.GoBody.Pn("// Deprecated\n//")
}
