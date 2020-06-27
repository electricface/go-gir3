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
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/xerrors"
)

type SourceFile struct {
	Pkg       string
	CPkgList  []string
	CIncludes []string
	Header    *SourceBody
	CHeader   *SourceBody
	CBody     *SourceBody

	GoImports []string
	GoBody    *SourceBody
}

func NewSourceFile(pkg string) *SourceFile {
	sf := &SourceFile{
		Pkg:     pkg,
		Header:  &SourceBody{},
		CHeader: &SourceBody{},
		CBody:   &SourceBody{},
		GoBody:  &SourceBody{},
	}

	sf.CBody.sourceFile = sf
	sf.GoBody.sourceFile = sf
	return sf
}

func (s *SourceFile) Print() {
	err := s.writeTo(os.Stdout)
	if err != nil {
		log.Println("WARN:", err)
	}
}

func (s *SourceFile) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return xerrors.Errorf("create file: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println("WARN: failed to close file:", err)
		}
	}()

	err = s.writeTo(f)
	if err != nil {
		return xerrors.Errorf("write to file: %w", err)
	}

	fmtOut, err := exec.Command("go", "fmt", filename).CombinedOutput()
	if len(fmtOut) > 0 {
		_, err := fmt.Fprintf(os.Stdout, "go fmt output:\n%s", fmtOut)
		if err != nil {
			log.Println("WARN:", err)
		}
	}
	if err != nil {
		return xerrors.Errorf("run go fmt: %w", err)
	}
	return nil
}

func (s *SourceFile) writeTo(w io.Writer) error {
	_, err := w.Write(s.Header.buf.Bytes())
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, "package "+s.Pkg+"\n")
	if err != nil {
		return err
	}

	if len(s.CPkgList) > 0 ||
		len(s.CIncludes) > 0 ||
		len(s.CHeader.buf.Bytes()) > 0 ||
		len(s.CBody.buf.Bytes()) > 0 {

		_, err = io.WriteString(w, "/*\n")
		if err != nil {
			return err
		}
		if len(s.CPkgList) > 0 {
			sort.Strings(s.CPkgList)
			str := "#cgo pkg-config: " + strings.Join(s.CPkgList, " ") + "\n"
			_, err = io.WriteString(w, str)
			if err != nil {
				return err
			}
		}

		sort.Strings(s.CIncludes)
		for _, inc := range s.CIncludes {
			_, err = io.WriteString(w, "#include "+inc+"\n")
			if err != nil {
				return err
			}
		}

		_, err = w.Write(s.CHeader.buf.Bytes())
		if err != nil {
			return err
		}
		_, err = w.Write(s.CBody.buf.Bytes())
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, "*/\n")
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "import \"C\"\n")
		if err != nil {
			return err
		}
	}

	sort.Strings(s.GoImports)
	for _, imp := range s.GoImports {
		_, err = io.WriteString(w, "import "+imp+"\n")
		if err != nil {
			return err
		}
	}

	_, err = w.Write(s.GoBody.buf.Bytes())
	return err
}

func (s *SourceFile) AddCPkg(cPkg string) {
	for _, cPkg0 := range s.CPkgList {
		if cPkg0 == cPkg {
			return
		}
	}
	s.CPkgList = append(s.CPkgList, cPkg)
}

func (s *SourceFile) AddCInclude(inc string) {
	for _, inc0 := range s.CIncludes {
		if inc0 == inc {
			return
		}
	}
	s.CIncludes = append(s.CIncludes, inc)
}

// unsafe => "unsafe"
// or x,github.com/path/ => x "path"
func (s *SourceFile) AddGoImport(imp string) {
	var importStr string
	if strings.Contains(imp, ",") {
		parts := strings.SplitN(imp, ",", 2)
		importStr = fmt.Sprintf("%s %q", parts[0], parts[1])
	} else {
		importStr = `"` + imp + `"`
	}

	for _, imp0 := range s.GoImports {
		if imp0 == importStr {
			return
		}
	}
	s.GoImports = append(s.GoImports, importStr)
}

func (s *SourceFile) AddGirImport(name string) {
	fullPath := _girPkgPath + "/" + name
	s.AddGoImport(fullPath)
}

type SourceBody struct {
	sourceFile *SourceFile
	buf        bytes.Buffer
}

// gir:glib -> project_root/glib-2.0
// go:.util -> project_root/util
// go:string -> string
// ch:<stdlib.h>

var requireReg = regexp.MustCompile(`/\*(\w+):(.+?)\*/`)

func (v *SourceBody) writeStr(str string) {
	subMatchResults := requireReg.FindAllStringSubmatch(str, -1)

	for _, subMatchResult := range subMatchResults {
		typ := subMatchResult[1]
		arg := subMatchResult[2]

		switch typ {
		case "go":
			if strings.HasPrefix(arg, ".") {
				//v.sourceFile.AddGoImport(filepath.Join(getGirProjectRoot(), arg[1:]))
				// TODO:

			} else {
				v.sourceFile.AddGoImport(arg)
			}

		case "gir":
			v.sourceFile.AddGirImport(arg)

		case "ch":
			v.sourceFile.AddCInclude(arg)
		}
	}

	if len(subMatchResults) > 0 {
		str = requireReg.ReplaceAllString(str, "")
	}

	v.buf.WriteString(str)
}

func (v *SourceBody) Pn(format string, a ...interface{}) {
	v.P(format, a...)
	v.buf.WriteByte('\n')
}

func (v *SourceBody) P(format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	v.writeStr(str)
}

func (v *SourceBody) WriteString(str string) {
	v.buf.WriteString(str)
}

func (v *SourceBody) addBlock(block *SourceBlock) {
	v.buf.Write(block.buf.Bytes())
}

type SourceBlock struct {
	buf bytes.Buffer
}

func (v *SourceBlock) Pn(format string, a ...interface{}) {
	v.P(format, a...)
	v.buf.WriteByte('\n')
}

func (v *SourceBlock) P(format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	v.buf.WriteString(str)
}

func (v *SourceBlock) containsTodo() bool {
	return bytes.Contains(v.buf.Bytes(), []byte("TODO"))
}
