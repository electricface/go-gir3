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
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/electricface/go-gir3/gi"
	"golang.org/x/xerrors"
)

func toGiScopeExpr(scopeType gi.ScopeType) string {
	scope := "gi.ScopeInvalid"
	switch scopeType {
	case gi.SCOPE_TYPE_INVALID:
		scope = "gi.ScopeInvalid"
	case gi.SCOPE_TYPE_ASYNC:
		scope = "gi.ScopeAsync"
	case gi.SCOPE_TYPE_CALL:
		scope = "gi.ScopeCall"
	case gi.SCOPE_TYPE_NOTIFIED:
		scope = "gi.ScopeNotified"
	}
	return scope
}

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

func toCamelCase(name, sep string) string {
	var out bytes.Buffer
	for _, word := range strings.Split(name, sep) {
		if word == "" {
			continue
		}
		word = strings.ToLower(word)
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

func (vr *VarReg) registerParam(idx int, name string) string {
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

// copyFileContent copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContent(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer func() {
		_ = in.Close()
	}()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cErr := out.Close()
		if err == nil {
			err = cErr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func getGoPath() string {
	gopath := os.Getenv("GOPATH")
	paths := strings.Split(gopath, ":")
	if len(paths) > 0 {
		return strings.TrimSpace(paths[0])
	}
	log.Fatal("do not set env var GOPATH")
	return ""
}

// cleanFiles 删除目录 dir 下的所有文件
func cleanFiles(dir string) error {
	fileInfoList, err := ioutil.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return xerrors.Errorf("read lib.in dir: %w", err)
		}
	} else {
		// 删除 dir 文件夹下所有文件
		for _, info := range fileInfoList {
			file := filepath.Join(dir, info.Name())
			log.Println("remove file", file)
			err = os.Remove(file)
			if err != nil {
				return xerrors.Errorf("remove file: %w")
			}
		}
	}
	return nil
}

func syncFilesToLibIn(libInDir, outDir string) error {
	// sync go-gir -> lib.in
	if _optNamespace == "GObject" || _optNamespace == "Gio" {
		// 不多次复制，只处理 namespace 为 GLib 一次。
		return nil
	}

	// 删除 libInDir 文件夹下所有文件
	err := cleanFiles(libInDir)
	if err != nil {
		return xerrors.Errorf("clean files: %w", err)
	}

	fileInfoList, err := ioutil.ReadDir(outDir)
	if err != nil {
		return xerrors.Errorf("read out dir: %w", err)
	}

	// 复制 go-gir 文件夹下所有 .go 但是不是 _auto.go 的，所有 *config.json。

	// 要复制的文件名列表
	var srcNames []string

	for _, info := range fileInfoList {
		name := info.Name()
		ext := filepath.Ext(name)
		if (ext == ".go" && !strings.HasSuffix(name, "_auto.go")) ||
			strings.HasSuffix(name, "config.json") {

			srcNames = append(srcNames, name)
		}
	}

	if len(srcNames) == 0 {
		return nil
	}
	err = os.Mkdir(libInDir, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return xerrors.Errorf("make dir: %w", err)
		}
	}
	for _, name := range srcNames {
		src := filepath.Join(outDir, name)
		dst := filepath.Join(libInDir, name)
		log.Printf("copy %s -> %s\n", src, dst)
		err := copyFileContent(src, dst)
		if err != nil {
			return xerrors.Errorf("copy file content from %q to %q: %w", src, dst, err)
		}
	}

	return nil
}

func syncFilesToOut(libInDir, outDir string) error {
	// sync lib.in -> go-gir
	if _optNamespace == "GObject" || _optNamespace == "Gio" {
		// 不多次复制，只处理 namespace 为 GLib 一次。
		return nil
	}

	// 删除 outDir 文件夹下所有文件
	err := cleanFiles(outDir)
	if err != nil {
		return xerrors.Errorf("clean files: %w", err)
	}

	fileInfoList, err := ioutil.ReadDir(libInDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return xerrors.Errorf("read dir: %w", err)
		}
	} else {
		for _, info := range fileInfoList {
			name := info.Name()
			ext := filepath.Ext(name)
			if ext != ".go" && ext != ".json" {
				// 只复制 .go 和 .json 文件
				continue
			}

			src := filepath.Join(libInDir, name)
			dst := filepath.Join(outDir, name)
			log.Printf("copy %s -> %s\n", src, dst)
			err = copyFileContent(src, dst)
			if err != nil {
				return xerrors.Errorf("copy file content from %q to %q: %w", src, dst, err)
			}
		}
	}
	return nil
}

// flag --sync-gi
func syncLibGiToOut() error {
	gopath := getGoPath()
	outDir := filepath.Join(gopath, "src", _girPkgPath, "gi")
	err := cleanFiles(outDir)
	if err != nil {
		return xerrors.Errorf("clean files: %w", err)
	}

	const giDir = "./gi-lite"
	fileInfos, err := ioutil.ReadDir(giDir)
	if err != nil {
		return xerrors.Errorf("read dir: %w", err)
	}
	for _, info := range fileInfos {
		src := filepath.Join(giDir, info.Name())
		dst := filepath.Join(outDir, info.Name())
		log.Printf("copy %s -> %s\n", src, dst)
		err = copyFileContent(src, dst)
		if err != nil {
			return xerrors.Errorf("copy file content from %q to %q: %w", src, dst, err)
		}
	}
	return nil
}
