package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/electricface/go-gir3/cmd/girgen/xmlp"
	"github.com/electricface/go-gir3/gi"
)

const girPkgPath = "github.com/electricface/go-gir"

var _optNamespace string
var _optVersion string
var _optDir string
var _optOutputFile string
var _optCfgFile string
var _optPkg string

func init() {
	log.SetFlags(log.Lshortfile)
	flag.StringVar(&_optNamespace, "n", "", "namespace")
	flag.StringVar(&_optVersion, "v", "", "version")
	flag.StringVar(&_optDir, "d", "", "output directory")
	flag.StringVar(&_optOutputFile, "f", "", "output file")
	flag.StringVar(&_optCfgFile, "c", "", "config file")
	flag.StringVar(&_optPkg, "p", "", "package")
}

var _structNamesMap = make(map[string]struct{}) // 键是所有 struct 类型名。
var _symbolNameMap = make(map[string]string)    // 键是 c 符号， value 是方法名，是调整过的方法名。
var _deps []string
var _cfg *config
var _sourceFile *SourceFile
var _xRepo *xmlp.Repository

func getGoPath() string {
	gopath := os.Getenv("GOPATH")
	paths := strings.Split(gopath, ":")
	if len(paths) > 0 {
		return strings.TrimSpace(paths[0])
	}
	return ""
}

func main() {
	flag.Parse()
	if _optDir == "" {
		gopath := getGoPath()
		if gopath == "" {
			log.Fatal(errors.New("do not set env var GOPATH"))
		}
		_optDir = filepath.Join(gopath, "src", girPkgPath,
			strings.ToLower(_optNamespace+"-"+_optVersion))
	}

	pkg := strings.ToLower(_optNamespace)
	if _optPkg != "" {
		pkg = _optPkg
	}

	outFile := filepath.Join(_optDir, pkg+"_auto.go")
	if _optOutputFile != "" {
		outFile = _optOutputFile
	}
	log.Print("outFile:", outFile)

	outDir := filepath.Dir(outFile)
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	genStateFile := filepath.Join(outDir, "genState.json")
	if _optNamespace == "GObject" || _optNamespace == "Gio" {
		var gs genState
		err = loadGenState(genStateFile, &gs)
		if err != nil {
			log.Fatalln(err)
		}
		nsOrder := []string{"GLib", "GObject", "Gio"}
		var prevNs string
		for i, ns := range nsOrder {
			if ns == _optNamespace {
				prevNs = nsOrder[i-1]
			}
		}
		if gs.PrevNamespace != prevNs {
			log.Fatalf("prev namespace is not %v", prevNs)
		}

		_funcNextIdx = gs.FuncNextId
		_getTypeNextId = gs.GetTypeNextId
	}

	configFile := filepath.Join(outDir, "config.json")
	if _optCfgFile != "" {
		configFile = filepath.Join(outDir, _optCfgFile)
	}

	var cfg config
	err = loadConfig(configFile, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	_cfg = &cfg

	repo := gi.DefaultRepository()
	_, err = repo.Require(_optNamespace, _optVersion, gi.REPOSITORY_LOAD_FLAG_LAZY)
	if err != nil {
		log.Fatal(err)
	}
	xRepo, err := xmlp.Load(_optNamespace, _optVersion)
	if err != nil {
		log.Fatal(err)
	}
	_xRepo = xRepo

	deps := getAllDeps(repo, _optNamespace)
	log.Printf("deps: %#v\n", deps)
	_deps = deps

	sourceFile := NewSourceFile(pkg)
	_sourceFile = sourceFile

	for _, cInclude := range xRepo.CIncludes() {
		sourceFile.AddCInclude("<" + cInclude.Name + ">")
	}
	for _, cInclude := range cfg.CIncludes {
		sourceFile.AddCInclude(cInclude)
	}

	for _, pkg := range xRepo.Packages {
		sourceFile.AddCPkg(pkg.Name)
	}

	sourceFile.AddGoImport("gi,github.com/electricface/go-gir3/gi-lite")
	sourceFile.AddGoImport("unsafe")
	sourceFile.AddGoImport("log")

	if _optNamespace == "Gio" || _optNamespace == "GObject" {
		// 不再输出 var _ID
		sourceFile.GoBody.Pn("var _ gi.GType")
	} else {
		sourceFile.GoBody.Pn("var _I = gi.NewInvokerCache(%q)", _optNamespace)
	}
	sourceFile.GoBody.Pn("var _ unsafe.Pointer")
	sourceFile.GoBody.Pn("var _ *log.Logger")
	sourceFile.GoBody.Pn("func init() {")
	sourceFile.GoBody.Pn("repo := gi.DefaultRepository()")
	sourceFile.GoBody.Pn("_, err := repo.Require(%q, %q, gi.REPOSITORY_LOAD_FLAG_LAZY)",
		_optNamespace, _optVersion)
	sourceFile.GoBody.Pn("if err != nil {")
	sourceFile.GoBody.Pn("    panic(err)")
	sourceFile.GoBody.Pn("}") // end if

	sourceFile.GoBody.Pn("}") // end func

	numInfos := repo.NumInfo(_optNamespace)
	for i := 0; i < numInfos; i++ {
		bi := repo.Info(_optNamespace, i)
		name := bi.Name()
		switch bi.Type() {
		case gi.INFO_TYPE_STRUCT, gi.INFO_TYPE_UNION, gi.INFO_TYPE_OBJECT, gi.INFO_TYPE_INTERFACE:
			_structNamesMap[name] = struct{}{}
		}
		bi.Unref()
	}

	// 处理函数命名冲突
	forEachFunctionInfo(repo, _optNamespace, handleFuncNameClash)
	var constants []string

	for i := 0; i < numInfos; i++ {
		bi := repo.Info(_optNamespace, i)
		switch bi.Type() {
		case gi.INFO_TYPE_FUNCTION:
			fi := gi.ToFunctionInfo(bi)
			pFunction(sourceFile, fi)

		case gi.INFO_TYPE_CALLBACK:
			ci := gi.ToCallableInfo(bi)
			pCallback(sourceFile, ci)

		case gi.INFO_TYPE_STRUCT:
			si := gi.ToStructInfo(bi)
			pStruct(sourceFile, si)

		case gi.INFO_TYPE_UNION:
			ui := gi.ToUnionInfo(bi)
			pUnion(sourceFile, ui)

		case gi.INFO_TYPE_OBJECT:
			oi := gi.ToObjectInfo(bi)
			pObject(sourceFile, oi)

		case gi.INFO_TYPE_INTERFACE:
			ii := gi.ToInterfaceInfo(bi)
			pInterface(sourceFile, ii)

		case gi.INFO_TYPE_ENUM:
			ei := gi.ToEnumInfo(bi)
			pEnum(sourceFile, ei, true)

		case gi.INFO_TYPE_FLAGS:
			ei := gi.ToEnumInfo(bi)
			pEnum(sourceFile, ei, false)

		case gi.INFO_TYPE_CONSTANT:
			ci := gi.ToConstantInfo(bi)
			constants = pConstant(constants, ci)

			//case gi.INFO_TYPE_BOXED:
			//case gi.INFO_TYPE_VALUE:
			//case gi.INFO_TYPE_SIGNAL:
			//case gi.INFO_TYPE_VFUNC:
			//case gi.INFO_TYPE_PROPERTY:
			//case gi.INFO_TYPE_FIELD:
			//case gi.INFO_TYPE_ARG:
			//case gi.INFO_TYPE_TYPE:
		}
		bi.Unref()
	}

	// print constants
	sourceFile.GoBody.Pn("// constants\nconst (")
	for i := 0; i+1 < len(constants); i += 2 {
		sourceFile.GoBody.Pn("%s = %s", constants[i], constants[i+1])
	}
	sourceFile.GoBody.Pn(")")

	if _optNamespace == "GLib" || _optNamespace == "GObject" {
		err = saveGenState(genStateFile, &genState{
			PrevNamespace: _optNamespace,
			FuncNextId:    _funcNextIdx,
			GetTypeNextId: _getTypeNextId,
		})
		if err != nil {
			log.Fatalln(err)
		}
	} else if _optNamespace == "Gio" {
		err = os.Remove(genStateFile)
		if err != nil {
			log.Println("WARN:", err)
		}
	}

	if _optNamespace == "GLib" || _optNamespace == "Gio" || _optNamespace == "GObject" {
		// 修正 gio 和 gobject 的 import
		var temp []string
		for _, imp := range sourceFile.GoImports {
			if strings.HasSuffix(imp, "glib-2.0\"") ||
				strings.HasSuffix(imp, "gobject-2.0\"") {
				// ignore
			} else {
				temp = append(temp, imp)
			}
		}
		sourceFile.GoImports = temp
	} else {
		// 修正其它更上层的包（比如 Gdk、Gtk）对 g-2.0 包的导入。
		var temp []string
		useG := false
		for _, imp := range sourceFile.GoImports {
			if strings.HasSuffix(imp, "glib-2.0\"") ||
				strings.HasSuffix(imp, "gobject-2.0\"") ||
				strings.HasSuffix(imp, "gio-2.0\"") {
				// ignore
				useG = true
			} else {
				temp = append(temp, imp)
			}
		}
		sourceFile.GoImports = temp
		if useG {
			sourceFile.AddGirImport("g-2.0")
		}
	}

	err = sourceFile.Save(outFile)
	if err != nil {
		log.Fatal("failed to save: ", err)
	}

	log.Printf("stat %v TODO/ALL %d/%d %.2f%%\n", _optNamespace, _numTodoFunc, _numFunc,
		float64(_numTodoFunc)/float64(_numFunc)*100)
}

func pConstant(constants []string, ci *gi.ConstantInfo) []string {
	val := ci.Value()
	valStr, ok := val.(string)
	if ok {
		constants = append(constants, ci.Name(), strconv.Quote(valStr))
	} else {
		constants = append(constants, ci.Name(), fmt.Sprintf("%v", val))
	}
	return constants
}

func getFlagsTypeName(type0 string) string {
	// NOTE： 为了避免 flags 和函数重名了，比如 flags FileTest 和 file_test 函数, 就加上 Flags 后缀。
	if strings.HasSuffix(type0, "Flags") {
		return type0
	}
	return type0 + "Flags"
}

func getEnumTypeName(type0 string) string {
	return type0 + "Enum"
}

func pEnum(s *SourceFile, ei *gi.EnumInfo, isEnum bool) {
	if ei.IsDeprecated() {
		markDeprecated(s)
	}
	name := ei.Name()
	type0 := name
	if isEnum {
		s.GoBody.Pn("// Enum %v", name)
		type0 = getEnumTypeName(type0)
	} else {
		// is Flags
		s.GoBody.Pn("// Flags %v", name)
		type0 = getFlagsTypeName(type0)
	}
	s.GoBody.Pn("type %s int", type0)
	s.GoBody.Pn("const (")
	num := ei.NumValue()
	for i := 0; i < num; i++ {
		value := ei.Value(i)
		val := value.Value()
		memberName := name + snake2Camel(value.Name())
		s.GoBody.Pn("%s %s = %v", memberName, type0, val)
		value.Unref()
	}
	s.GoBody.Pn(")") // end const

	// NOTE: enum 和 flags 也有类型的
	pGetTypeFunc(s, name)
}

func pStruct(s *SourceFile, si *gi.StructInfo) {
	name := si.Name()

	if si.IsGTypeStruct() {
		// 过滤掉对象的 Class 结构，比如 Object 的 ObjectClass
		s.GoBody.Pn("// ignore GType struct %s", name)
		return
	}
	if si.IsDeprecated() {
		markDeprecated(s)
	}

	// 过滤掉 Class 结尾的，并且还存在去掉 Class 后缀后还存在的类型的结构。
	// 目前它只过滤掉了 gobject 的 TypePluginClass 结构， 而 TypePlugin 是接口。
	if strings.HasSuffix(name, "Class") {
		repo := gi.DefaultRepository()
		nameRmClass := strings.TrimSuffix(name, "Class")
		bi := repo.FindByName(_optNamespace, nameRmClass)
		if !bi.IsNil() {
			s.GoBody.Pn("// ignore Class struct %s, type of %s is %s",
				name, nameRmClass, bi.Type())
			bi.Unref()
			return
		}
	}

	s.GoBody.Pn("// Struct %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}")

	size := si.Size()
	if size > 0 {
		s.GoBody.Pn("const SizeOfStruct%v = %v", name, size)
	}

	pGetTypeFunc(s, name)

	numMethod := si.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := si.Method(i)
		pFunction(s, fi)
	}
}

// 给 XXXGetType 用的 id
var _getTypeNextId int

func pGetTypeFunc(s *SourceFile, name string) {
	s.GoBody.Pn("func %sGetType() gi.GType {", name)

	if _optNamespace == "GObject" || _optNamespace == "Gio" {
		s.GoBody.Pn("ret := _I.GetGType1(%v, %q, %q)", _getTypeNextId, _optNamespace, name)
	} else {
		s.GoBody.Pn("ret := _I.GetGType(%v, %q)", _getTypeNextId, name)
	}

	s.GoBody.Pn("return ret")
	s.GoBody.Pn("}")

	_getTypeNextId++
}

func pUnion(s *SourceFile, ui *gi.UnionInfo) {
	if ui.IsDeprecated() {
		markDeprecated(s)
	}
	name := ui.Name()
	s.GoBody.Pn("// Union %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}")

	size := ui.Size()
	if size > 0 {
		s.GoBody.Pn("const SizeOfUnion%v = %v", name, size)
	}

	pGetTypeFunc(s, name)

	numMethod := ui.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := ui.Method(i)
		pFunction(s, fi)
	}
}

func pInterface(s *SourceFile, ii *gi.InterfaceInfo) {
	if ii.IsDeprecated() {
		markDeprecated(s)
	}
	name := ii.Name()
	s.GoBody.Pn("// Interface %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    %sIfc", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}") // end struct

	s.GoBody.Pn("type %sIfc struct{}", name)

	s.GoBody.P("type I%s interface {", name)
	s.GoBody.Pn("P_%s() unsafe.Pointer }", name)
	s.GoBody.Pn("func (v %s) P_%s() unsafe.Pointer { return v.P }", name, name)

	pGetTypeFunc(s, name)

	numMethod := ii.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := ii.Method(i)
		pFunction(s, fi)
	}
}

func pObject(s *SourceFile, oi *gi.ObjectInfo) {
	name := oi.Name()
	if oi.IsDeprecated() {
		markDeprecated(s)
	}
	s.GoBody.Pn("// Object %s", name)
	s.GoBody.Pn("type %s struct {", name)

	// object 有的接口
	numIfcs := oi.NumInterface()
	for i := 0; i < numIfcs; i++ {
		ii := oi.Interface(i)
		typeName := getTypeName(gi.ToBaseInfo(ii))
		s.GoBody.Pn("%sIfc", typeName)
		ii.Unref()
	}

	// object 继承关系
	parent := oi.Parent()
	if parent != nil {
		parentTypeName := getTypeName(gi.ToBaseInfo(parent))
		s.GoBody.Pn("%s", parentTypeName)
		parent.Unref()
	} else {
		s.GoBody.Pn("P unsafe.Pointer")
	}

	s.GoBody.Pn("}") // end struct

	if parent != nil {
		// 只有有 parent 的 object 才提供 WrapXXX 方法
		s.GoBody.P("func Wrap%s(p unsafe.Pointer) (r %s) {", name, name)
		s.GoBody.P("r.P = p;")
		s.GoBody.Pn("return }")
	}

	s.GoBody.P("type I%s interface {", name)
	s.GoBody.Pn("P_%s() unsafe.Pointer }", name)
	s.GoBody.Pn("func (v %s) P_%s() unsafe.Pointer { return v.P }", name, name)

	for i := 0; i < numIfcs; i++ {
		ii := oi.Interface(i)
		ifcTypeName := ii.Name()
		s.GoBody.Pn("func (v %s) P_%s() unsafe.Pointer { return v.P }", name, ifcTypeName)
		ii.Unref()
	}

	pGetTypeFunc(s, name)

	numMethod := oi.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := oi.Method(i)
		pFunction(s, fi)
	}
}

func forEachFunctionInfo(repo *gi.Repository, namespace string, fn func(fi *gi.FunctionInfo)) {
	numInfos := repo.NumInfo(namespace)
	for i := 0; i < numInfos; i++ {
		bi := repo.Info(namespace, i)
		switch bi.Type() {
		case gi.INFO_TYPE_FUNCTION:
			fi := gi.ToFunctionInfo(bi)
			fn(fi)
		case gi.INFO_TYPE_STRUCT:
			si := gi.ToStructInfo(bi)
			numMethods := si.NumMethod()
			for i := 0; i < numMethods; i++ {
				fi := si.Method(i)
				fn(fi)
				fi.Unref()
			}
		case gi.INFO_TYPE_UNION:
			ui := gi.ToUnionInfo(bi)
			numMethods := ui.NumMethod()
			for i := 0; i < numMethods; i++ {
				fi := ui.Method(i)
				fn(fi)
				fi.Unref()
			}
		case gi.INFO_TYPE_OBJECT:
			oi := gi.ToObjectInfo(bi)
			numMethods := oi.NumMethod()
			for i := 0; i < numMethods; i++ {
				fi := oi.Method(i)
				fn(fi)
				fi.Unref()
			}
		case gi.INFO_TYPE_INTERFACE:
			ii := gi.ToInterfaceInfo(bi)
			numMethods := ii.NumMethod()
			for i := 0; i < numMethods; i++ {
				fi := ii.Method(i)
				fn(fi)
				fi.Unref()
			}
		}
		bi.Unref()
	}
}

func handleFuncNameClash(fi *gi.FunctionInfo) {
	symbol := fi.Symbol()
	fn := getFunctionName(fi)

	if _, ok := _structNamesMap[fn]; ok {
		// 方法名和结构体名冲突了
		_symbolNameMap[symbol] = fn + "F"
	}
}
