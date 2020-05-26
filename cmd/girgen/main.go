package main

import (
	"flag"
	"log"
	"path/filepath"
	"strings"

	"github.com/electricface/go-gir3/gi"
)

var optNamespace string
var optVersion string
var optPkg string
var optDir string

func init() {
	flag.StringVar(&optNamespace, "n", "", "namespace")
	flag.StringVar(&optVersion, "v", "", "version")
	flag.StringVar(&optPkg, "p", "", "pkg")
	flag.StringVar(&optDir, "d", "", "output directory")
}

func main() {
	flag.Parse()

	repo := gi.DefaultRepository()
	_, err := repo.Require(optNamespace, optVersion, gi.REPOSITORY_LOAD_FLAG_LAZY)
	if err != nil {
		log.Fatal(err)
	}

	pkg := optPkg
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

	num := repo.NumInfo(optNamespace)
	for i := 0; i < num; i++ {
		bi := repo.Info(optNamespace, i)
		name := bi.Name()
		switch bi.Type() {
		case gi.INFO_TYPE_FUNCTION:
			log.Println(name, "FUNCTION")
			fi := gi.ToFunctionInfo(bi)
			pFunction(sourceFile, fi)

		case gi.INFO_TYPE_CALLBACK:
		case gi.INFO_TYPE_STRUCT:
			log.Println(name, "STRUCT")
			si := gi.ToStructInfo(bi)
			pStruct(sourceFile, si)

		case gi.INFO_TYPE_BOXED:
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
			num := oi.NumMethod()
			for j := 0; j < num; j++ {
				//fi := oi.Method(j)
				//checkFi(fi)
			}
		case gi.INFO_TYPE_INTERFACE:
			log.Println(name, "INTERFACE")
			info := gi.ToInterfaceInfo(bi)
			num := info.NumMethod()
			for j := 0; j < num; j++ {
				//fi := info.Method(j)
				//checkFi(fi)
			}

		case gi.INFO_TYPE_CONSTANT:
		case gi.INFO_TYPE_UNION:
			log.Println(name, "UNION")

		case gi.INFO_TYPE_VALUE:
		case gi.INFO_TYPE_SIGNAL:
		case gi.INFO_TYPE_VFUNC:
		case gi.INFO_TYPE_PROPERTY:
		case gi.INFO_TYPE_FIELD:
		case gi.INFO_TYPE_ARG:
		case gi.INFO_TYPE_TYPE:
		}
		bi.Unref()
	}

	outFile := filepath.Join(optDir, optPkg+"_auto.go")
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
	s.GoBody.Pn("// Struct %s", name)

	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    Ptr unsafe.Pointer")
	s.GoBody.Pn("}")

	numMethod := si.NumMethod()
	for i := 0; i < numMethod; i++ {
		fi := si.Method(i)
		pFunction(s, fi)
	}
}
