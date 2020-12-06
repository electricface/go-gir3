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
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/electricface/go-gir3/cmd/girgen/xmlp"
	"github.com/electricface/go-gir3/gi"
)

var _girPkgPath = "github.com/linuxdeepin/go-gir"

const fileHeader = `/*
 * Copyright (C) 2019 ~ $year Uniontech Software Technology Co.,Ltd
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

// Code generated by "girgen"; DO NOT EDIT.

`

var _optNamespace string
var _optVersion string
var _optDir string
var _optOutputFile string
var _optCfgFile string
var _optPkg string
var _optSyncGi bool

var _xRepo *xmlp.Repository

func init() {
	log.SetFlags(log.Lshortfile)
	flag.StringVar(&_optNamespace, "n", "", "namespace")
	flag.StringVar(&_optVersion, "v", "", "version")
	flag.StringVar(&_optDir, "d", "", "output directory")
	flag.StringVar(&_optOutputFile, "f", "", "output file")
	flag.StringVar(&_optCfgFile, "c", "", "config file")
	flag.StringVar(&_optPkg, "p", "", "package")
	flag.BoolVar(&_optSyncGi, "sync-gi", false, "sync gi to out dir")
}

var _structNamesMap = make(map[string]struct{}) // 键是所有 struct 类型名。
var _symbolNameMap = make(map[string]string)    // 键是 c 符号， value 是方法名，是调整过的方法名。

// 它是 getAllDeps() 的返回结果
var _deps []string

var _cfg *config
var _sourceFile *SourceFile

var _sigNamesMap = make(map[string]struct{})

func main() {
	flag.Parse()

	if _optSyncGi {
		err := syncLibGiToOut()
		if err != nil {
			log.Fatalf("failed to sync lib gi to out: %v", err)
		}
		return
	}

	envGirPkgPath := os.Getenv("GIR_PKG_PATH")
	if envGirPkgPath != "" {
		_girPkgPath = envGirPkgPath
	}

	gopath := getGoPath()
	if _optDir == "" {
		_optDir = filepath.Join(gopath, "src", _girPkgPath,
			strings.ToLower(_optNamespace+"-"+_optVersion))
	}

	pkg := strings.ToLower(_optNamespace)
	if _optPkg != "" {
		pkg = _optPkg
	}

	outFile := filepath.Join(_optDir, pkg+"_auto.go")
	if _optOutputFile != "" {
		outFile = strings.Replace(_optOutputFile, "%GOPATH%", gopath, 1)
	}
	log.Print("outFile:", outFile)

	outDir := filepath.Dir(outFile)
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	// 目录名，比如 g-2.0, gtk-3.0
	dirName := filepath.Base(outDir)
	libInInfo, err := os.Stat("lib.in")
	if err != nil {
		log.Fatal(err)
	}
	if !libInInfo.IsDir() {
		log.Fatal("lib.in is not a directory")
	}
	libInDir := filepath.Join("lib.in", dirName)

	// mode dev, 默认, sync file go-gir -> lib.in
	envMode := os.Getenv("GIRGEN_SYNC_MODE")
	if envMode == "" || envMode == "dev" {
		err = syncFilesToLibIn(libInDir, outDir)
		if err != nil {
			log.Fatalf("failed to sync files to lib.in: %v", err)
		}
	} else if envMode == "build" {
		// mode build, sync file lib.in -> go-gir
		err = syncFilesToOut(libInDir, outDir)
		if err != nil {
			log.Fatalf("failed to syn files to out: %v", err)
		}
	} else {
		log.Fatalf("invalid sync mode %q", envMode)
	}

	genStateFile := filepath.Join(outDir, "genState.json")
	if _optNamespace == "GObject" || _optNamespace == "Gio" {
		var state genState
		err = loadGenState(genStateFile, &state)
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
		if state.PrevNamespace != prevNs {
			log.Fatalf("prev namespace is not %v", prevNs)
		}

		_funcNextIdx = state.FuncNextId
		_getTypeNextId = state.GetTypeNextId
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

	yearStr := strconv.Itoa(time.Now().Year())
	header := strings.Replace(fileHeader, "$year", yearStr, 1)
	sourceFile.Header.WriteString(header)

	for _, define := range cfg.CDefines {
		sourceFile.AddCDefine(define)
	}

	for _, cInclude := range xRepo.CIncludes() {
		sourceFile.AddCInclude("<" + cInclude.Name + ">")
	}
	for _, cInclude := range cfg.CIncludes {
		sourceFile.AddCInclude(cInclude)
	}
	for _, cPkg := range cfg.CPkgList {
		sourceFile.AddCPkg(cPkg)
	}

	for _, pkg := range xRepo.Packages {
		sourceFile.AddCPkg(pkg.Name)
	}

	sourceFile.AddGirImport("gi")
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

	for idxLv1 := 0; idxLv1 < numInfos; idxLv1++ {
		bi := repo.Info(_optNamespace, idxLv1)
		switch bi.Type() {
		case gi.INFO_TYPE_FUNCTION:
			fi := gi.ToFunctionInfo(bi)
			pFunction(sourceFile, fi, idxLv1, 0)

		case gi.INFO_TYPE_CALLBACK:
			ci := gi.ToCallableInfo(bi)
			pCallback(sourceFile, ci)

		case gi.INFO_TYPE_STRUCT:
			si := gi.ToStructInfo(bi)
			pStruct(sourceFile, si, idxLv1)

		case gi.INFO_TYPE_UNION:
			ui := gi.ToUnionInfo(bi)
			pUnion(sourceFile, ui, idxLv1)

		case gi.INFO_TYPE_OBJECT:
			oi := gi.ToObjectInfo(bi)
			pObject(sourceFile, oi, idxLv1)

		case gi.INFO_TYPE_INTERFACE:
			ii := gi.ToInterfaceInfo(bi)
			pInterface(sourceFile, ii, idxLv1)

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

	pSignalNameConstants(sourceFile)

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

func pSignal(si *gi.SignalInfo) {
	name := si.Name()
	_sigNamesMap[name] = struct{}{}
}

func pSignalNameConstants(sf *SourceFile) {
	if len(_sigNamesMap) == 0 {
		return
	}
	names := make([]string, len(_sigNamesMap))
	i := 0
	for sigName := range _sigNamesMap {
		names[i] = sigName
		i++
	}
	sort.Strings(names)
	sf.GoBody.Pn("const (")
	for _, sigName := range names {
		name := toCamelCase(sigName, "-")
		sf.GoBody.Pn("Sig%v = %q", name, sigName)
	}
	sf.GoBody.Pn(")") // end const
}

func pConstant(constants []string, ci *gi.ConstantInfo) []string {
	val := ci.Value()
	if val == nil {
		// 忽略这个常量
		return constants
	}

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
	name := getTypeName(ei.Name())
	var type0 string
	if isEnum {
		s.GoBody.Pn("// Enum %v", name)
		type0 = getEnumTypeName(name)
	} else {
		// is Flags
		s.GoBody.Pn("// Flags %v", name)
		type0 = getFlagsTypeName(name)
	}
	s.GoBody.Pn("type %s int", type0)
	s.GoBody.Pn("const (")
	num := ei.NumValue()
	for i := 0; i < num; i++ {
		value := ei.Value(i)
		val := value.Value()
		memberName := name + snake2Camel(value.Name())
		if memberName == type0 {
			// 成员和类型重名了
			memberName += "0"
		}
		s.GoBody.Pn("%s %s = %v", memberName, type0, val)
		value.Unref()
	}
	s.GoBody.Pn(")") // end const

	// NOTE: enum 和 flags 也有类型的
	pGetTypeFunc(s, name, ei.Name())
}

func pStruct(s *SourceFile, si *gi.StructInfo, idxLv1 int) {
	name := si.Name()

	repo := gi.DefaultRepository()
	numMethods := si.NumMethod()
	if si.IsGTypeStruct() {
		// 过滤掉对象的 Class 结构，比如 gtk.Window 的 WindowClass
		if numMethods == 0 {
			s.GoBody.Pn("// ignore GType struct %v\n", name)
			return
		}
	}

	if strings.HasSuffix(name, "Private") && numMethods == 0 {
		nameTrim := strings.TrimSuffix(name, "Private")
		bi := repo.FindByName(_optNamespace, nameTrim)
		if !bi.IsNil() {
			s.GoBody.Pn("// ignore private struct %v, type of %v is %v\n",
				name, nameTrim, bi.Type())
			bi.Unref()
			return
		}
	}

	// 过滤掉 Class 结尾的，并且还存在去掉 Class 后缀后还存在的类型的结构。
	// 目前它只过滤掉了 gobject 的 TypePluginClass 结构， 而 TypePlugin 是接口。
	if strings.HasSuffix(name, "Class") && numMethods == 0 {
		nameTrim := strings.TrimSuffix(name, "Class")
		bi := repo.FindByName(_optNamespace, nameTrim)
		if !bi.IsNil() {
			s.GoBody.Pn("// ignore class struct %v, type of %v is %v\n",
				name, nameTrim, bi.Type())
			bi.Unref()
			return
		}
	}

	typeDef, _ := _xRepo.GetType(name)
	var xStructInfo *xmlp.StructInfo
	if typeDef != nil {
		xStructInfo = typeDef.(*xmlp.StructInfo)
	}

	if si.IsDeprecated() {
		markDeprecated(s)
	}

	s.GoBody.Pn("// Struct %s", name)
	s.GoBody.Pn("type %s struct {", name)
	s.GoBody.Pn("    P unsafe.Pointer")
	s.GoBody.Pn("}")

	size := si.Size()
	if size > 0 {
		s.GoBody.Pn("const SizeOfStruct%v = %v", name, size)
	}

	pGetTypeFunc(s, name, "")

	for idxLv2 := 0; idxLv2 < numMethods; idxLv2++ {
		fi := si.Method(idxLv2)
		pFunction(s, fi, idxLv1, idxLv2)
	}

	numFields := si.NumField()
	if !strSliceContains(_cfg.DeniedFieldsStructs, name) && numFields > 0 {
		pStructPFunc(s, si)
		for i := 0; i < numFields; i++ {
			field := si.Field(i)
			fieldName := field.Name()

			if strSliceContains(_cfg.DeniedFields, fmt.Sprintf("%v.%v", name, fieldName)) {
				s.GoBody.Pn("// denied field %v.%v\n", name, fieldName)
				continue
			}

			var xField *xmlp.Field
			if xStructInfo != nil {
				xField = xStructInfo.GetFieldByName(fieldName)
			}

			flags := field.Flags()
			if flags&gi.FIELD_IS_READABLE == gi.FIELD_IS_READABLE {
				// is readable
				pStructGetFunc(s, field, name, xField)
			}
			if flags&gi.FIELD_IS_WRITABLE == gi.FIELD_IS_WRITABLE {
				// is writable
				//pStructSetFunc()
			}

			field.Unref()
		}
	}
}

func pStructPFunc(s *SourceFile, si *gi.StructInfo) {
	ns := si.Namespace()
	repo := gi.DefaultRepository()
	cPrefix := repo.CPrefix(ns)
	structName := si.Name()
	cTypeName := cPrefix + structName
	if cPrefix == "cairo" {
		if structName == "Context" {
			cTypeName = "cairo_t"
		} else {
			cTypeName = cPrefix + "_" + camel2Snake(structName) + "_t"
		}
	}
	s.GoBody.Pn("\nfunc (v %v) p() %v {", structName, "*C."+cTypeName)
	s.GoBody.Pn("return (*C.%v)(v.P)", cTypeName)
	s.GoBody.Pn("}") // end func
}

func pStructGetFunc(s *SourceFile, fieldInfo *gi.FieldInfo, structName string, xField *xmlp.Field) {
	fieldName := fieldInfo.Name()
	if xField != nil && xField.Bits > 0 {
		s.GoBody.Pn("// TODO: ignore struct %v field %v, bits(=%v) > 0\n", structName, fieldName, xField.Bits)
		return
	}
	var varReg VarReg
	typeInfo := fieldInfo.Type()
	defer typeInfo.Unref()

	parseResult := parseFieldType(typeInfo, fieldName)
	getFnName := snake2Camel(fieldName)
	if fieldName == "p" {
		getFnName = "P0"
	}
	varResult := varReg.alloc("result")
	s.GoBody.Pn("func (v %v) %v() (%v %v) {", structName, getFnName, varResult, parseResult.goType)
	if !strings.Contains(parseResult.goType, "/*TODO*/") {
		s.GoBody.Pn("%v = %v", varResult, parseResult.expr)
	}
	s.GoBody.Pn("    return")
	s.GoBody.Pn("}") // end func
}

type parseFieldTypeResult struct {
	goType string
	field  string
	expr   string
}

func parseFieldType(ti *gi.TypeInfo, fieldName string) *parseFieldTypeResult {
	if strSliceContains(_goKeywords, fieldName) {
		fieldName = "_" + fieldName
	}

	isPtr := ti.IsPointer()
	tag := ti.Tag()
	_ = tag
	_ = isPtr
	goType := "int /*TODO*/"
	fieldExpr := "v.p()." + fieldName
	expr := fmt.Sprintf("int(%v) /* TODO */", fieldExpr)

	switch tag {
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
	// 字符串类型
	//goType = "string"

	case gi.TYPE_TAG_INT8, gi.TYPE_TAG_UINT8,
		gi.TYPE_TAG_INT16, gi.TYPE_TAG_UINT16,
		gi.TYPE_TAG_INT32, gi.TYPE_TAG_UINT32,
		gi.TYPE_TAG_INT64, gi.TYPE_TAG_UINT64,
		gi.TYPE_TAG_FLOAT, gi.TYPE_TAG_DOUBLE:
		// 简单类型
		if !isPtr {
			goType = getTypeWithTag(tag)
			expr = fmt.Sprintf("%v(%v)", goType, fieldExpr)
		}

	case gi.TYPE_TAG_BOOLEAN:
		if !isPtr {
			goType = "bool"
			expr = fmt.Sprintf("gi.Int2Bool(int(%v))", fieldExpr)
		}

	case gi.TYPE_TAG_UNICHAR:
		goType = "rune"
		expr = fmt.Sprintf("%v(%v)", goType, fieldExpr)

	case gi.TYPE_TAG_INTERFACE:
	// TODO

	case gi.TYPE_TAG_GTYPE:
		goType = "gi.GType"
		expr = fmt.Sprintf("%v(%v)", goType, fieldExpr)

	case gi.TYPE_TAG_VOID:
		if isPtr {
			goType = "unsafe.Pointer"
			expr = fmt.Sprintf("%v(%v)", goType, fieldExpr)
		}

	case gi.TYPE_TAG_ARRAY:
		// TODO
	}

	return &parseFieldTypeResult{
		goType: goType,
		expr:   expr,
	}
}

// 给 XXXGetType 用的 id
var _getTypeNextId int

func pGetTypeFunc(s *SourceFile, name, realName string) {
	if realName == "" {
		realName = name
	}
	if strSliceContains(_cfg.NoGetType, name) {
		s.GoBody.Pn("// noGetType %s\n", name)
		_getTypeNextId++
		return
	}

	s.GoBody.Pn("func %sGetType() gi.GType {", name)

	if _optNamespace == "GObject" || _optNamespace == "Gio" {
		s.GoBody.Pn("ret := _I.GetGType1(%v, %q, %q)", _getTypeNextId, _optNamespace, realName)
	} else {
		s.GoBody.Pn("ret := _I.GetGType(%v, %q)", _getTypeNextId, realName)
	}

	s.GoBody.Pn("return ret")
	s.GoBody.Pn("}")
	_getTypeNextId++
}

func pUnion(s *SourceFile, ui *gi.UnionInfo, idxLv1 int) {
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

	pGetTypeFunc(s, name, "")

	numMethod := ui.NumMethod()
	for idxLv2 := 0; idxLv2 < numMethod; idxLv2++ {
		fi := ui.Method(idxLv2)
		pFunction(s, fi, idxLv1, idxLv2)
	}
}

func pInterface(s *SourceFile, ii *gi.InterfaceInfo, idxLv1 int) {
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

	pGetTypeFunc(s, name, "")

	numMethod := ii.NumMethod()
	for idxLv2 := 0; idxLv2 < numMethod; idxLv2++ {
		fi := ii.Method(idxLv2)
		pFunction(s, fi, idxLv1, idxLv2)
	}

	numSig := ii.NumSignal()
	for i := 0; i < numSig; i++ {
		si := ii.Signal(i)
		pSignal(si)
	}
}

//  isParentImplIfc 返回是否父类型实现了 ifcInfo 接口
func isParentImplIfc(oi *gi.ObjectInfo, ifcInfo *gi.InterfaceInfo) bool {
	ifcGType := ifcInfo.GetGType()
	parent := oi.Parent()
	if parent == nil {
		return false
	}
	numIfcs := parent.NumInterface()
	for i := 0; i < numIfcs; i++ {
		ii := parent.Interface(i)
		gType := ii.GetGType()
		if gType == ifcGType {
			return true
		}
		ii.Unref()
	}
	result := isParentImplIfc(parent, ifcInfo)
	parent.Unref()
	return result
}

func pObject(s *SourceFile, oi *gi.ObjectInfo, idxLv1 int) {
	name := oi.Name()
	if oi.IsDeprecated() {
		markDeprecated(s)
	}
	s.GoBody.Pn("// Object %s", name)
	s.GoBody.Pn("type %s struct {", name)

	var embeddedIfcs []string

	numIfcs := oi.NumInterface()
	for i := 0; i < numIfcs; i++ {
		ii := oi.Interface(i)

		// 如果父类型没有实现此接口，才嵌入它
		if !isParentImplIfc(oi, ii) {
			typeName := getTypeNameWithBaseInfo(gi.ToBaseInfo(ii))
			s.GoBody.Pn("%sIfc", typeName)
			embeddedIfcs = append(embeddedIfcs, ii.Name())
		}

		ii.Unref()
	}

	// object 继承关系
	parent := oi.Parent()
	if parent != nil {
		parentTypeName := getTypeNameWithBaseInfo(gi.ToBaseInfo(parent))
		s.GoBody.Pn("%s", parentTypeName)
		parent.Unref()
	} else {
		s.GoBody.Pn("P unsafe.Pointer")
	}

	s.GoBody.Pn("}") // end struct

	s.GoBody.P("func Wrap%s(p unsafe.Pointer) (r %s) {", name, name)
	s.GoBody.P("r.P = p;")
	s.GoBody.Pn("return }")

	s.GoBody.P("type I%s interface {", name)
	s.GoBody.Pn("P_%s() unsafe.Pointer }", name)
	s.GoBody.Pn("func (v %s) P_%s() unsafe.Pointer { return v.P }", name, name)

	for _, ifc := range embeddedIfcs {
		s.GoBody.Pn("func (v %s) P_%s() unsafe.Pointer { return v.P }", name, ifc)
	}

	pGetTypeFunc(s, name, "")

	numMethod := oi.NumMethod()
	for idxLv2 := 0; idxLv2 < numMethod; idxLv2++ {
		fi := oi.Method(idxLv2)
		pFunction(s, fi, idxLv1, idxLv2)
	}

	numSig := oi.NumSignal()
	for i := 0; i < numSig; i++ {
		si := oi.Signal(i)
		pSignal(si)
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
