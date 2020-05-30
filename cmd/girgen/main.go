package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/electricface/go-gir3/gi"
)

const girPkgPath = "github.com/electricface/go-gir"

var optNamespace string
var optVersion string
var optDir string

func init() {
	flag.StringVar(&optNamespace, "n", "", "namespace")
	flag.StringVar(&optVersion, "v", "", "version")
	flag.StringVar(&optDir, "d", "", "output directory")
}

var globalStructNamesMap = make(map[string]struct{}) // 键是所有 struct 类型名。
var globalSymbolNameMap = make(map[string]string)    // 键是 c 符号， value 是方法名，是调整过的方法名。
var globalDeps []string
var globalCfg *config

func main() {
	flag.Parse()
	if optDir == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			log.Fatal(errors.New("do not set env var GOPATH"))
		}
		optDir = filepath.Join(gopath, "src", girPkgPath,
			strings.ToLower(optNamespace+"-"+optVersion))
	}

	configFile := filepath.Join(optDir, "config.json")
	var cfg config
	err := loadConfig(configFile, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	globalCfg = &cfg

	repo := gi.DefaultRepository()
	_, err = repo.Require(optNamespace, optVersion, gi.REPOSITORY_LOAD_FLAG_LAZY)
	if err != nil {
		log.Fatal(err)
	}

	deps := getAllDeps(repo, optNamespace)
	log.Printf("deps: %#v\n", deps)
	globalDeps = deps

	//loadedNs := repo.LoadedNamespaces()
	//log.Println("loadedNs:", loadedNs)

	pkg := strings.ToLower(optNamespace)
	sourceFile := NewSourceFile(pkg)

	sourceFile.AddGoImport("github.com/electricface/go-gir3/gi")
	sourceFile.AddGoImport("unsafe")

	sourceFile.GoBody.Pn("var _I = gi.NewInvokerCache(%q)", optNamespace)
	sourceFile.GoBody.Pn("var _ unsafe.Pointer")
	sourceFile.GoBody.Pn("func init() {")
	sourceFile.GoBody.Pn("repo := gi.DefaultRepository()")
	sourceFile.GoBody.Pn("_, err := repo.Require(%q, %q, gi.REPOSITORY_LOAD_FLAG_LAZY)",
		optNamespace, optVersion)
	sourceFile.GoBody.Pn("if err != nil {")
	sourceFile.GoBody.Pn("    panic(err)")
	sourceFile.GoBody.Pn("}") // end if

	sourceFile.GoBody.Pn("}") // end func

	numInfos := repo.NumInfo(optNamespace)
	for i := 0; i < numInfos; i++ {
		bi := repo.Info(optNamespace, i)
		name := bi.Name()
		switch bi.Type() {
		case gi.INFO_TYPE_STRUCT, gi.INFO_TYPE_UNION, gi.INFO_TYPE_OBJECT, gi.INFO_TYPE_INTERFACE:
			globalStructNamesMap[name] = struct{}{}
		}
		bi.Unref()
	}

	// 处理函数命名冲突
	forEachFunctionInfo(repo, optNamespace, handleFuncNameClash)

	for i := 0; i < numInfos; i++ {
		bi := repo.Info(optNamespace, i)
		name := bi.Name()
		switch bi.Type() {
		case gi.INFO_TYPE_FUNCTION:
			log.Println(name, "FUNCTION")
			if !strSliceContains(globalCfg.Black, name) {
				fi := gi.ToFunctionInfo(bi)
				pFunction(sourceFile, fi)
			}
		case gi.INFO_TYPE_CALLBACK:
		case gi.INFO_TYPE_STRUCT:
			log.Println(name, "STRUCT")
			si := gi.ToStructInfo(bi)
			pStruct(sourceFile, si)

		case gi.INFO_TYPE_BOXED:
			// TODO 什么是 BOXED?
		case gi.INFO_TYPE_ENUM:
			log.Println(name, "ENUM")
			ei := gi.ToEnumInfo(bi)
			pEnum(sourceFile, ei, true)

		case gi.INFO_TYPE_FLAGS:
			log.Println(name, "FLAGS")
			ei := gi.ToEnumInfo(bi)
			pEnum(sourceFile, ei, false)

		case gi.INFO_TYPE_OBJECT:
			log.Println(name, "OBJECT")
			oi := gi.ToObjectInfo(bi)
			pObject(sourceFile, oi)

		case gi.INFO_TYPE_INTERFACE:
			log.Println(name, "INTERFACE")
			ii := gi.ToInterfaceInfo(bi)
			pInterface(sourceFile, ii)

		case gi.INFO_TYPE_CONSTANT:
			// TODO 常量
		case gi.INFO_TYPE_UNION:
			log.Println(name, "UNION")
			ui := gi.ToUnionInfo(bi)
			pUnion(sourceFile, ui)

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

	err = os.MkdirAll(optDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	outFile := filepath.Join(optDir, pkg+"_auto.go")
	sourceFile.Save(outFile)
}

func pEnum(s *SourceFile, enum *gi.EnumInfo, isEnum bool) {
	name := enum.Name()
	type0 := name
	if isEnum {
		type0 += "Enum"
	} else {
		// is Flags
		if !strings.HasSuffix(type0, "Flags") {
			type0 += "Flags"
		}
		// NOTE： 为了避免 enum 和函数重名了，比如 enum FileTest 和 file_test 函数, 就加上 Flags 后缀。
	}
	s.GoBody.Pn("type %s int", type0)
	s.GoBody.Pn("const (")
	num := enum.NumValue()
	for i := 0; i < num; i++ {
		value := enum.Value(i)
		val := value.Value()
		memberName := name + snake2Camel(value.Name())
		if i == 0 {
			s.GoBody.Pn("%s %s = %v", memberName, type0, val)
		} else {
			s.GoBody.Pn("%s = %v", memberName, val)
		}
		value.Unref()
	}
	s.GoBody.Pn(")") // end const
}

func pStruct(s *SourceFile, si *gi.StructInfo) {
	name := si.Name()

	if si.IsGTypeStruct() {
		return
	}

	s.GoBody.Pn("// Struct %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}")

	numMethod := si.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := si.Method(i)
		pFunction(s, fi)
	}
}

func pUnion(s *SourceFile, ui *gi.UnionInfo) {
	name := ui.Name()
	s.GoBody.Pn("// Union %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}")

	numMethod := ui.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := ui.Method(i)
		pFunction(s, fi)
	}
}

func pInterface(s *SourceFile, ii *gi.InterfaceInfo) {
	name := ii.Name()
	s.GoBody.Pn("// Interface %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}")

	numMethod := ii.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := ii.Method(i)
		pFunction(s, fi)
	}
}

func pObject(s *SourceFile, oi *gi.ObjectInfo) {
	name := oi.Name()
	s.GoBody.Pn("// Object %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}")

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

	if _, ok := globalStructNamesMap[fn]; ok {
		// 方法名和结构体名冲突了
		globalSymbolNameMap[symbol] = fn + "F"
	}
}
