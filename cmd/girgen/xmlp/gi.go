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

package xmlp

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// key 如 GLib-2.0
var loadedRepos = make(map[string]*Repository)

func GetLoadedRepo(ns string) *Repository {
	if strings.Contains(ns, "-") {
		return loadedRepos[ns]
	}

	for nsVer, repo := range loadedRepos {
		if strings.HasPrefix(nsVer, ns) {
			return repo
		}
	}
	return nil
}

type Repository struct {
	XMLName   xml.Name
	Version   string     `xml:"version,attr"`
	Includes  []*Include `xml:"include"`
	Packages  []*Package `xml:"package"`
	Namespace *Namespace `xml:"namespace"`

	includeRepos map[string]*Repository
	typeMap      map[string]TypeDefine
}

type TypeDefine interface {
	Name() string
	CType() *CType
}

type ParseCTypeError struct {
	err        error
	typeDefine TypeDefine
}

func (err *ParseCTypeError) Error() string {
	return fmt.Sprintf("failed to parse ctype for %#v %s",
		err.typeDefine, err.err.Error())
}

func (r *Repository) postDecode() {
	r.loadIncludeRepos()

	var err error
	r.typeMap = make(map[string]TypeDefine)

	ns := r.Namespace
	for _, enum := range ns.Enums {
		enum.cType, err = ParseCType(enum.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: enum,
			})
		}
		r.typeMap[enum.NameAttr] = enum
	}

	for _, enum := range ns.Bitfields {
		enum.cType, err = ParseCType(enum.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: enum,
			})
		}
		if _, ok := r.typeMap[enum.NameAttr]; ok {
			panic("duplicate type " + enum.NameAttr)
		}

		r.typeMap[enum.NameAttr] = enum
	}

	for _, struct0 := range ns.Structs {
		struct0.cType, err = ParseCType(struct0.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: struct0,
			})
		}

		for _, fn := range struct0.Functions {
			fn.container = struct0
		}
		for _, fn := range struct0.Constructors {
			fn.container = struct0
		}
		for _, fn := range struct0.Methods {
			fn.container = struct0
		}

		if _, ok := r.typeMap[struct0.NameAttr]; ok {
			panic("duplicate type " + struct0.NameAttr)
		}
		r.typeMap[struct0.NameAttr] = struct0
	}

	for _, union := range ns.Unions {
		if strings.HasPrefix(union.NameAttr, "_") {
			// ignore _XXXX
			continue
		}

		union.cType, err = ParseCType(union.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: union,
			})
		}

		if _, ok := r.typeMap[union.NameAttr]; ok {
			panic("duplicate type " + union.NameAttr)
		}
		r.typeMap[union.NameAttr] = union
	}

	for _, class := range ns.Objects {
		if class.CTypeAttr == "" && class.GlibTypeName != "" {
			class.CTypeAttr = class.GlibTypeName
		}
		class.cType, err = ParseCType(class.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: class,
			})
		}

		for _, fn := range class.Functions {
			fn.container = class
		}
		for _, fn := range class.Constructors {
			fn.container = class
		}
		for _, fn := range class.Methods {
			fn.container = class
		}
		for _, fn := range class.VirtualMethods {
			fn.container = class
		}
		for _, fn := range class.Signals {
			fn.container = class
		}

		if _, ok := r.typeMap[class.NameAttr]; ok {
			panic("duplicate type " + class.NameAttr)
		}

		r.typeMap[class.NameAttr] = class
	}

	for _, ifc := range ns.Interfaces {
		ifc.cType, err = ParseCType(ifc.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: ifc,
			})
		}

		for _, fn := range ifc.Functions {
			fn.container = ifc
		}
		for _, fn := range ifc.VirtualMethods {
			fn.container = ifc
		}
		for _, fn := range ifc.Methods {
			fn.container = ifc
		}

		if _, ok := r.typeMap[ifc.NameAttr]; ok {
			panic("duplicate type " + ifc.NameAttr)
		}

		r.typeMap[ifc.NameAttr] = ifc
	}

	for _, alias := range ns.Aliases {
		alias.cType, err = ParseCType(alias.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: alias,
			})
		}

		if _, ok := r.typeMap[alias.NameAttr]; ok {
			panic("duplicate type " + alias.NameAttr)
		}

		r.typeMap[alias.NameAttr] = alias
	}

	for _, callback := range ns.Callbacks {
		callback.cType, err = ParseCType(callback.CTypeAttr)
		if err != nil {
			panic(&ParseCTypeError{
				err:        err,
				typeDefine: callback,
			})
		}

		if _, ok := r.typeMap[callback.NameAttr]; ok {
			panic("duplicate type " + callback.NameAttr)
		}
		r.typeMap[callback.NameAttr] = callback
	}
	fmt.Println("// finish type register", r.Namespace.Name, r.Namespace.Version)
}

func (r *Repository) GetType(name string) (TypeDefine, string) {
	if strings.Contains(name, ".") {
		// get type from include repos
		nameParts := strings.Split(name, ".")
		ns := nameParts[0]
		name0 := nameParts[1] // name remove prefix

		if repo, ok := r.includeRepos[ns]; ok {
			typ := repo.typeMap[name0]
			if typ != nil {
				return typ, repo.Namespace.Name
			}
		}

		for _, repo := range r.includeRepos {
			typ, ns := repo.GetType(name)
			if typ != nil {
				return typ, ns
			}
		}
		return nil, ""
	}
	return r.typeMap[name], r.Namespace.Name
}

func (r *Repository) GetTypes() map[string]TypeDefine {
	return r.typeMap
}

func (r *Repository) CIncludes() []*Include {
	var ret []*Include
	for _, r := range r.Includes {
		if getSpace(r.XMLName.Space) == SpaceC {
			ret = append(ret, r)
		}
	}
	return ret
}

func (r *Repository) loadIncludeRepos() {
	r.includeRepos = make(map[string]*Repository)
	for _, inc := range r.CoreIncludes() {
		repo, err := Load(inc.Name, inc.Version)
		if err != nil {
			panic(err)
		}

		r.includeRepos[inc.Name] = repo
	}
}

func (r *Repository) CoreIncludes() []*Include {
	var ret []*Include
	for _, r := range r.Includes {
		if getSpace(r.XMLName.Space) == SpaceCore {
			ret = append(ret, r)
		}
	}
	return ret
}

type Include struct {
	XMLName xml.Name
	Name    string `xml:"name,attr"`
	Version string `xml:"version,attr"`
}

type Package struct {
	Name string `xml:"name,attr"`
}

type Namespace struct {
	Name                string `xml:"name,attr"`
	Version             string `xml:"version,attr"`
	SharedLibrary       string `xml:"shared-library,attr"`
	CIdentifierPrefixes string `xml:"identifier-prefixes,attr"`
	CSymbolPrefixes     string `xml:"symbol-prefixes,attr"`

	Aliases    []*AliasInfo     `xml:"alias"`
	Interfaces []*InterfaceInfo `xml:"interface"`
	Objects    []*ObjectInfo    `xml:"class"`
	Structs    []*StructInfo    `xml:"record"`
	Enums      []*EnumInfo      `xml:"enumeration"`
	Bitfields  []*EnumInfo      `xml:"bitfield"`
	Constants  []*ConstantInfo  `xml:"constant"`
	Unions     []*UnionInfo     `xml:"union"`
	Functions  []*FunctionInfo  `xml:"function"`
	Callbacks  []*CallbackInfo  `xml:"callback"`
}

type BaseInfo struct {
	NameAttr          string `xml:"name,attr"`
	CTypeAttr         string `xml:"type,attr"` // c:type attr
	Deprecated        bool   `xml:"deprecated,attr"`
	DeprecatedVersion string `xml:"deprecated-version,attr"`
	cType             *CType
}

func (b *BaseInfo) Name() string {
	return b.NameAttr
}

func (b *BaseInfo) CType() *CType {
	return b.cType
}

// c typedef?
type AliasInfo struct {
	BaseInfo
	SourceType *Type `xml:"type"`
}

type Type struct {
	Name     string `xml:"name,attr"`
	CType    string `xml:"type,attr"`
	ElemType *Type  `xml:"type"`
}

type FunctionInfo struct {
	BaseInfo
	CIdentifier    string      `xml:"identifier,attr"`
	MovedTo        string      `xml:"moved-to,attr"`
	ReturnValue    *Parameter  `xml:"return-value"`
	Parameters     *Parameters `xml:"parameters"`
	Throws         bool        `xml:"throws,attr"`
	Introspectable bool        `xml:"introspectable,attr"`
	Shadows        string      `xml:"shadows,attr"`
	ShadowedBy     string      `xml:"shadowed-by,attr"`

	container TypeDefine
}

type CallbackInfo struct {
	BaseInfo
	ReturnValue *Parameter  `xml:"return-value"`
	Parameters  *Parameters `xml:"parameters"`
}

// set Introspectable default value to true
func (f *FunctionInfo) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type functionInfo0 FunctionInfo // new type to prevent recursion
	f0 := functionInfo0{
		Introspectable: true,
	}
	if err := d.DecodeElement(&f0, &start); err != nil {
		return err
	}
	*f = (FunctionInfo)(f0)
	return nil
}

func (f *FunctionInfo) Name() string {
	var noInstanceParam bool
	if f.Parameters != nil {
		noInstanceParam = f.Parameters.InstanceParameter == nil
	} else {
		noInstanceParam = true
	}

	var prefix string
	if f.container != nil && noInstanceParam {
		prefix = f.container.Name()
	}

	if f.MovedTo != "" {
		if strings.ContainsRune(f.MovedTo, '.') {
			parts := strings.SplitN(f.MovedTo, ".", 2)
			return parts[0] + snake2Camel(parts[1])
		}
		return snake2Camel(f.MovedTo)
	}
	if f.Shadows != "" {
		return prefix + snake2Camel(f.Shadows)
	}
	return prefix + snake2Camel(f.NameAttr)
}

type Parameters struct {
	Parameters        []*Parameter `xml:"parameter"`
	InstanceParameter *Parameter   `xml:"instance-parameter"`
}

type Parameter struct {
	Name                    string     `xml:"name,attr"`
	TransferOwnership       string     `xml:"transfer-ownership,attr"`
	Direction               string     `xml:"direction,attr"`
	CallerAllocates         bool       `xml:"caller-allocates,attr"`
	Optional                bool       `xml:"optional,attr"`
	Nullable                bool       `xml:"nullable,attr"`
	AllowNone               bool       `xml:"allow-none,attr"`
	Type                    *Type      `xml:"type"`
	Array                   *ArrayType `xml:"array"`
	LengthForParameter      *Parameter
	ClosureForCallbackParam *Parameter
	ClosureParam            *Parameter

	Scope        string `xml:"scope,attr"`
	ClosureIndex int    `xml:"closure,attr"`
	DestroyIndex int    `xml:"destroy,attr"`
}

// set ClosureIndex and DestroyIndex default value to -1
func (p *Parameter) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type parameter0 Parameter
	a := parameter0{
		ClosureIndex: -1,
		DestroyIndex: -1,
	}
	if err := d.DecodeElement(&a, &start); err != nil {
		return err
	}
	*p = Parameter(a)
	return nil
}

func (p *Parameter) IsArray() bool {
	return p.Array != nil
}

type ArrayType struct {
	Name           string         `xml:"name,attr"`
	LengthIndex    int            `xml:"length,attr"`
	ZeroTerminated bool           `xml:"zero-terminated,attr"`
	FixedSize      int            `xml:"fixed-size,attr"`
	CType          string         `xml:"type,attr"`
	ElemType       *ArrayElemType `xml:"type"`

	LengthParameter *Parameter
}

type ArrayElemType struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"type,attr"`
}

// set LengthIndex default value to -1
func (arr *ArrayType) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type arrayType0 ArrayType // new type to prevent recursion
	a := arrayType0{
		LengthIndex:    -1,
		ZeroTerminated: true,
	}
	if err := d.DecodeElement(&a, &start); err != nil {
		return err
	}
	*arr = ArrayType(a)
	return nil
}

type Property struct {
	Name              string     `xml:"name,attr"`
	Writable          bool       `xml:"writable,attr"`
	ConstructOnly     bool       `xml:"construct-only,attr"`
	TransferOwnership string     `xml:"transfer-ownership,attr"`
	Array             *ArrayType `xml:"array"`
}

type Field struct {
	Name     string        `xml:"name,attr"`
	Readable bool          `xml:"readable,attr"`
	Writable bool          `xml:"writable,attr"`
	Private  bool          `xml:"private,attr"`
	Bits     int           `xml:"bits,attr"`
	Type     *Type         `xml:"type"`
	Callback *CallbackInfo `xml:"callback"`
}

type SignalInfo struct {
	FunctionInfo
	When string `xml:"when,attr"`
}

type VFuncInfo struct {
	FunctionInfo
	Invoker string `xml:"invoker,attr"`
}

type RegisteredTypeInfo struct {
	BaseInfo
	GlibGetType  string `xml:"get-type,attr"`
	GlibTypeName string `xml:"type-name,attr"`
}

type StructInfo struct {
	RegisteredTypeInfo
	GlibIsGtypeStructFor string   `xml:"is-gtype-struct-for,attr"`
	Fields               []*Field `xml:"field"`
	Disguised            bool     `xml:"disguised,attr"`

	Functions    []*FunctionInfo `xml:"function"`
	Constructors []*FunctionInfo `xml:"constructor"`
	Methods      []*FunctionInfo `xml:"method"`
}

func (si *StructInfo) GetFieldByName(name string) *Field {
	for _, field := range si.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}

type UnionInfo struct {
	RegisteredTypeInfo
	Fields []*Field `xml:"field"`
}

type ConstantInfo struct {
	BaseInfo
	Value string `xml:"value,attr"`
	Type  *Type  `xml:"type"`
}

type EnumInfo struct {
	RegisteredTypeInfo
	Members []*EnumMember `xml:"member"`
}

type EnumMember struct {
	Name        string `xml:"name,attr"`
	Value       string `xml:"value,attr"`
	CIdentifier string `xml:"identifier,attr"`
	GlibNick    string `xml:"nick,attr"`
}

type ObjectInfo struct {
	RegisteredTypeInfo
	CSymbolPrefixes string `xml:"symbol-prefix,attr"`
	Parent          string `xml:"parent,attr"`
	GlibTypeStruct  string `xml:"type-struct,attr"`

	Functions      []*FunctionInfo `xml:"function"`
	Constructors   []*FunctionInfo `xml:"constructor"`
	VirtualMethods []*VFuncInfo    `xml:"virtual-method"`
	Methods        []*FunctionInfo `xml:"method"`

	Properties []*Property   `xml:"property"`
	Fields     []*Field      `xml:"field"`
	Signals    []*SignalInfo `xml:"signal"`

	Implements []*ImplementedInterface `xml:"implements"`
}

type ImplementedInterface struct {
	Name string `xml:"name,attr"`
}

func (oi ObjectInfo) ImplementedInterfaces() []string {
	ret := make([]string, len(oi.Implements))
	for idx, ifc := range oi.Implements {
		ret[idx] = ifc.Name
	}
	return ret
}

type InterfaceInfo struct {
	RegisteredTypeInfo
	CSymbolPrefixes string `xml:"symbol-prefix,attr"`
	GlibTypeStruct  string `xml:"type-struct,attr"`

	Functions      []*FunctionInfo `xml:"function"`
	VirtualMethods []*VFuncInfo    `xml:"virtual-method"`
	Methods        []*FunctionInfo `xml:"method"`

	Properties []*Property `xml:"property"`
}

func Load(namespace, version string) (*Repository, error) {
	nsVer := namespace + "-" + version
	if repo, ok := loadedRepos[nsVer]; ok {
		fmt.Printf("// repo %s loaded\n", nsVer)
		return repo, nil
	}

	var err error
	girFile := fmt.Sprintf("/usr/share/gir-1.0/%s-%s.gir", namespace, version)
	fmt.Println("// load file:", girFile)
	girFh, err := os.Open(girFile)
	if err != nil {
		return nil, err
	}

	var repo Repository
	dec := xml.NewDecoder(bufio.NewReader(girFh))
	err = dec.Decode(&repo)
	if err != nil {
		return nil, err
	}
	repo.postDecode()
	fmt.Println("// end load", namespace, version)
	loadedRepos[nsVer] = &repo
	return &repo, nil
}
