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
)

type SourceFile struct {
	Pkg       string
	CPkgs     []string
	CIncludes []string
	CHeader   *SourceBody
	CBody     *SourceBody

	GoImports []string
	GoBody    *SourceBody
}

func NewSourceFile(pkg string) *SourceFile {
	sf := &SourceFile{
		Pkg: pkg,

		CHeader: &SourceBody{},
		CBody:   &SourceBody{},
		GoBody:  &SourceBody{},
	}

	sf.CBody.sourceFile = sf
	sf.GoBody.sourceFile = sf
	return sf
}

func (v *SourceFile) Print() {
	v.WriteTo(os.Stdout)
}

func (v *SourceFile) Save(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal("fail to create file:", err)
	}
	defer f.Close()

	v.WriteTo(f)

	err = f.Sync()
	if err != nil {
		log.Fatal("fail to sync file:", err)
	}

	err = exec.Command("go", "fmt", filename).Run()
	if err != nil {
		log.Fatal("failed to format file:", filename)
	}
}

func (v *SourceFile) WriteTo(w io.Writer) {
	io.WriteString(w, "package "+v.Pkg+"\n")

	if len(v.CPkgs) > 0 ||
		len(v.CIncludes) > 0 ||
		len(v.CHeader.buf.Bytes()) > 0 ||
		len(v.CBody.buf.Bytes()) > 0 {

		io.WriteString(w, "/*\n")
		if len(v.CPkgs) != 0 {
			str := "#cgo pkg-config: " + strings.Join(v.CPkgs, " ") + "\n"
			io.WriteString(w, str)
		}

		sort.Strings(v.CIncludes)
		for _, inc := range v.CIncludes {
			io.WriteString(w, "#include "+inc+"\n")
		}

		w.Write(v.CHeader.buf.Bytes())
		w.Write(v.CBody.buf.Bytes())

		io.WriteString(w, "*/\n")
		io.WriteString(w, "import \"C\"\n")
	}

	sort.Strings(v.GoImports)
	for _, imp := range v.GoImports {
		io.WriteString(w, "import "+imp+"\n")
	}

	w.Write(v.GoBody.buf.Bytes())
}

func (s *SourceFile) AddCPkg(pkg string) {
	s.CPkgs = append(s.CPkgs, pkg)
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

// TODO
//func (s *SourceFile) AddGirImport(ns string) {
//	repo := gi.GetLoadedRepo(ns)
//	if repo == nil {
//		panic("failed to get loaded repo " + ns)
//	}
//	base := strings.ToLower(repo.Namespace.Name) + "-" + repo.Namespace.Version
//	fullPath := filepath.Join(getGirProjectRoot(), base)
//	s.AddGoImport(fullPath)
//}

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
			//v.sourceFile.AddGirImport(arg)
			// TODO:

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
