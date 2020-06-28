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

package gi

/*
   #include <stdlib.h>
   #include <girepository.h>
   #include <girffi.h>
   #include <ffi.h>

   static inline void free_string(char *p) { free(p); }
   static inline void free_gstring(gchar *p) { if (p) g_free(p); }
   static inline char *gpointer_to_charp(gpointer p) { return p; }
   static inline gchar **next_gcharptr(gchar **s) { return s+1; }

   static void wrap_ffi_call(ffi_cif *cif, void (*fn)(void), void *rvalue,
   	GIArgument *args, int n_args, void *out_args) {

   	void **avalue = NULL;
   	if (n_args > 0) {
   		avalue = (void**)alloca(sizeof(gpointer) * n_args);
   		int i;
   		for (i = 0; i < n_args; i++) {
   			avalue[i] = &args[i];
   		}
   	}
   	ffi_call(cif, fn, rvalue, avalue);
   }

   #cgo pkg-config: gobject-introspection-1.0 gobject-introspection-no-export-1.0 libffi
*/
import "C"
import (
	"errors"
	"unsafe"
)

type GType uint

func Malloc(n int) unsafe.Pointer {
	return unsafe.Pointer(C.g_malloc(C.gsize(n)))
}

func Malloc0(n int) unsafe.Pointer {
	return unsafe.Pointer(C.g_malloc0(C.gsize(n)))
}

func Free(p unsafe.Pointer) {
	if p == nil {
		return
	}
	// 一般情况下 g_free 和 free 是一样的。
	// C.CString("str") 返回的内存可以用这个函数释放。
	C.g_free(C.gpointer(p))
}

// 注意需要 free 这个字符串
func CString(str string) unsafe.Pointer {
	if str == NilStr {
		return nil
	}
	return unsafe.Pointer(C.CString(str))
}

func GoString(p unsafe.Pointer) string {
	str := C.GoString((*C.char)(p))
	return str
}

type StrPtr struct {
	P unsafe.Pointer
}

func (v StrPtr) Take() string {
	str := C.GoString((*C.char)(v.P))
	C.free(v.P)
	return str
}

func (v StrPtr) Copy() string {
	str := C.GoString((*C.char)(v.P))
	return str
}

type Invoker struct {
	c *C.GIFunctionInvoker
}

type Repository struct {
	P unsafe.Pointer
}

// g_irepository_get_default
func getDefaultRepository() Repository {
	ret := C.g_irepository_get_default()
	return Repository{P: unsafe.Pointer(ret)}
}

func (r Repository) p() *C.GIRepository {
	return (*C.GIRepository)(r.P)
}

// g_irepository_find_by_name
func (r Repository) FindByName(namespace, name string) BaseInfo {
	gNamespace := _GoStringToGString(namespace)
	gName := _GoStringToGString(name)
	ret := C.g_irepository_find_by_name(r.p(), gNamespace, gName)
	C.free_gstring(gName)
	C.free_gstring(gNamespace)
	return BaseInfo{P: unsafe.Pointer(ret)}
}

// g_irepository_get_info
func (r Repository) Info(namespace string, index int) BaseInfo {
	gNamespace := _GoStringToGString(namespace)
	ret := C.g_irepository_get_info(r.p(), gNamespace, C.gint(index))
	C.free_gstring(gNamespace)
	return BaseInfo{P: unsafe.Pointer(ret)}
}

type Typelib struct {
	//c *C.GITypelib
	P unsafe.Pointer
}

type RepositoryLoadFlags int

const (
	REPOSITORY_LOAD_FLAG_LAZY RepositoryLoadFlags = C.G_IREPOSITORY_LOAD_FLAG_LAZY
)

// g_irepository_require
func (r *Repository) Require(namespace, version string, flags RepositoryLoadFlags) (Typelib, error) {
	var err *C.GError
	gNamespace := _GoStringToGString(namespace)
	gVersion := _GoStringToGString(version)
	ret := C.g_irepository_require(r.p(), gNamespace, gVersion, C.GIRepositoryLoadFlags(flags), &err)
	C.free_gstring(gVersion)
	C.free_gstring(gNamespace)

	if err != nil {
		return Typelib{}, _GErrorToOSError(err)
	}

	var tlwrap Typelib
	if ret != nil {
		tlwrap = Typelib{P: unsafe.Pointer(ret)}
	}

	return tlwrap, nil
}

type BaseInfo struct {
	P unsafe.Pointer
}

func (bi BaseInfo) p() *C.GIBaseInfo {
	return (*C.GIBaseInfo)(bi.P)
}

func (bi BaseInfo) Unref() {
	C.g_base_info_unref(bi.p())
}

// g_base_info_get_type
func (bi BaseInfo) Type() InfoType {
	return InfoType(C.g_base_info_get_type(bi.p()))
}

// g_base_info_get_name
func (bi BaseInfo) Name() string {
	return _GStringToGoString(C.g_base_info_get_name(bi.p()))
}

type InfoType int

const (
	INFO_TYPE_INVALID    InfoType = C.GI_INFO_TYPE_INVALID
	INFO_TYPE_FUNCTION   InfoType = C.GI_INFO_TYPE_FUNCTION
	INFO_TYPE_CALLBACK   InfoType = C.GI_INFO_TYPE_CALLBACK
	INFO_TYPE_STRUCT     InfoType = C.GI_INFO_TYPE_STRUCT
	INFO_TYPE_BOXED      InfoType = C.GI_INFO_TYPE_BOXED
	INFO_TYPE_ENUM       InfoType = C.GI_INFO_TYPE_ENUM
	INFO_TYPE_FLAGS      InfoType = C.GI_INFO_TYPE_FLAGS
	INFO_TYPE_OBJECT     InfoType = C.GI_INFO_TYPE_OBJECT
	INFO_TYPE_INTERFACE  InfoType = C.GI_INFO_TYPE_INTERFACE
	INFO_TYPE_CONSTANT   InfoType = C.GI_INFO_TYPE_CONSTANT
	INFO_TYPE_INVALID_0  InfoType = C.GI_INFO_TYPE_INVALID_0
	INFO_TYPE_UNION      InfoType = C.GI_INFO_TYPE_UNION
	INFO_TYPE_VALUE      InfoType = C.GI_INFO_TYPE_VALUE
	INFO_TYPE_SIGNAL     InfoType = C.GI_INFO_TYPE_SIGNAL
	INFO_TYPE_VFUNC      InfoType = C.GI_INFO_TYPE_VFUNC
	INFO_TYPE_PROPERTY   InfoType = C.GI_INFO_TYPE_PROPERTY
	INFO_TYPE_FIELD      InfoType = C.GI_INFO_TYPE_FIELD
	INFO_TYPE_ARG        InfoType = C.GI_INFO_TYPE_ARG
	INFO_TYPE_TYPE       InfoType = C.GI_INFO_TYPE_TYPE
	INFO_TYPE_UNRESOLVED InfoType = C.GI_INFO_TYPE_UNRESOLVED
)

// g_info_type_to_string
func (it InfoType) String() string {
	return _GStringToGoString(C.g_info_type_to_string(C.GIInfoType(it)))
}

type CallableInfo struct {
	BaseInfo
}

type RegisteredTypeInfo struct {
	BaseInfo
}

type InterfaceInfo struct {
	RegisteredTypeInfo
}

func WrapInterfaceInfo(p unsafe.Pointer) (ret InterfaceInfo) {
	ret.P = p
	return
}

func (ii InterfaceInfo) p() *C.GIInterfaceInfo {
	return (*C.GIInterfaceInfo)(ii.P)
}

// g_interface_info_find_method
func (ii InterfaceInfo) FindMethod(name string) FunctionInfo {
	gName := _GoStringToGString(name)
	ret := C.g_interface_info_find_method(ii.p(), gName)
	C.free_gstring(gName)
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

// g_interface_info_get_n_methods
func (ii InterfaceInfo) NumMethods() int {
	return int(C.g_interface_info_get_n_methods(ii.p()))
}

// g_interface_info_get_method
func (ii InterfaceInfo) Method(index int) FunctionInfo {
	ret := C.g_interface_info_get_method(ii.p(), C.gint(index))
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

type ObjectInfo struct {
	RegisteredTypeInfo
}

func (oi ObjectInfo) p() *C.GIObjectInfo {
	return (*C.GIObjectInfo)(oi.P)
}

func WrapObjectInfo(p unsafe.Pointer) (ret ObjectInfo) {
	ret.P = p
	return
}

// g_object_info_find_method
func (oi ObjectInfo) FindMethod(name string) FunctionInfo {
	gName := _GoStringToGString(name)
	ret := C.g_object_info_find_method(oi.p(), gName)
	C.free_gstring(gName)
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

// g_object_info_get_n_methods
func (oi ObjectInfo) NumMethods() int {
	return int(C.g_object_info_get_n_methods(oi.p()))
}

// g_object_info_get_method
func (oi ObjectInfo) Method(index int) FunctionInfo {
	ret := C.g_object_info_get_method(oi.p(), C.gint(index))
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

type StructInfo struct {
	RegisteredTypeInfo
}

func (si StructInfo) p() *C.GIStructInfo {
	return (*C.GIStructInfo)(si.P)
}

func WrapStructInfo(p unsafe.Pointer) (ret StructInfo) {
	ret.P = p
	return
}

// g_struct_info_find_method
func (si StructInfo) FindMethod(name string) FunctionInfo {
	gName := _GoStringToGString(name)
	ret := C.g_struct_info_find_method(si.p(), gName)
	C.free_gstring(gName)
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

// g_struct_info_get_field
func (si StructInfo) Field(n int) FieldInfo {
	ret := C.g_struct_info_get_field(si.p(), C.gint(n))
	return WrapFieldInfo(unsafe.Pointer(ret))
}

// g_struct_info_get_n_fields
func (si StructInfo) NumFields() int {
	ret := C.g_struct_info_get_n_fields(si.p())
	return int(ret)
}

func (si StructInfo) FindField(name string) FieldInfo {
	num := si.NumFields()
	for i := 0; i < num; i++ {
		fi := si.Field(i)
		if fi.Name() == name {
			return fi
		}
		fi.Unref()
	}
	return FieldInfo{}
}

// g_struct_info_get_n_methods
func (si StructInfo) NumMethods() int {
	return int(C.g_struct_info_get_n_methods(si.p()))
}

// g_struct_info_get_method
func (si StructInfo) Method(index int) FunctionInfo {
	ret := C.g_struct_info_get_method(si.p(), C.gint(index))
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

type UnionInfo struct {
	RegisteredTypeInfo
}

func (ui UnionInfo) p() *C.GIUnionInfo {
	return (*C.GIUnionInfo)(ui.P)
}

func WrapUnionInfo(p unsafe.Pointer) (ret UnionInfo) {
	ret.P = p
	return
}

// g_union_info_find_method
func (ui UnionInfo) FindMethod(name string) FunctionInfo {
	gName := _GoStringToGString(name)
	ret := C.g_union_info_find_method(ui.p(), gName)
	C.free_gstring(gName)
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

// g_union_info_get_n_methods
func (ui UnionInfo) NumMethods() int {
	return int(C.g_union_info_get_n_methods(ui.p()))
}

// g_union_info_get_method
func (ui UnionInfo) Method(index int) FunctionInfo {
	ret := C.g_union_info_get_method(ui.p(), C.gint(index))
	return WrapFunctionInfo(unsafe.Pointer(ret))
}

// g_union_info_get_field
func (ui UnionInfo) Field(n int) FieldInfo {
	ret := C.g_union_info_get_field(ui.p(), C.gint(n))
	return WrapFieldInfo(unsafe.Pointer(ret))
}

// g_union_info_get_n_fields
func (ui UnionInfo) NumFields() int {
	ret := C.g_union_info_get_n_fields(ui.p())
	return int(ret)
}

func (ui UnionInfo) FindField(name string) FieldInfo {
	num := ui.NumFields()
	for i := 0; i < num; i++ {
		fi := ui.Field(i)
		if fi.Name() == name {
			return fi
		}
		fi.Unref()
	}
	return FieldInfo{}
}

func (rti RegisteredTypeInfo) p() *C.GIRegisteredTypeInfo {
	return (*C.GIRegisteredTypeInfo)(rti.P)
}

func (rti RegisteredTypeInfo) GetGType() GType {
	ret := C.g_registered_type_info_get_g_type(rti.p())
	return GType(ret)
}

func WrapRegisteredTypeInfo(p unsafe.Pointer) (ret RegisteredTypeInfo) {
	ret.P = p
	return
}

type FunctionInfo struct {
	CallableInfo
}

func (fi FunctionInfo) p() *C.GIFunctionInfo {
	return (*C.GIFunctionInfo)(fi.P)
}

func WrapFunctionInfo(p unsafe.Pointer) (ret FunctionInfo) {
	ret.P = p
	return
}

func (fi FunctionInfo) PrepInvoker() (Invoker, error) {
	var err *C.GError
	var cInvoker C.GIFunctionInvoker
	ret := C.g_function_info_prep_invoker(fi.p(), &cInvoker, &err)
	if ret == 0 {
		goErr := _GErrorToOSError(err)
		return Invoker{}, goErr
	}
	return Invoker{c: &cInvoker}, nil
}

type FieldInfo struct {
	BaseInfo
}

func WrapFieldInfo(p unsafe.Pointer) (ret FieldInfo) {
	ret.P = p
	return
}

func (fi FieldInfo) p() *C.GIFieldInfo {
	return (*C.GIFieldInfo)(fi.P)
}

func (fi FieldInfo) GetField(mem unsafe.Pointer) (Argument, bool) {
	var value C.GIArgument
	ret := C.g_field_info_get_field(fi.p(), C.gpointer(mem), &value)

	ok := Int2Bool(int(ret))
	if ok {
		arg := *(*Argument)(unsafe.Pointer(&value))
		return arg, true
	}
	return Argument{}, false
}

func (invoker Invoker) Call(args []Argument, retVal *Argument, pOutArgs *Argument) {
	// pOutArgs 是用来传入 C 的 wrap_ffi_call 函数的，防止它指向的 outArgs 数组的地址改变。
	var cArgs *C.GIArgument
	if len(args) > 0 {
		cArgs = (*C.GIArgument)(unsafe.Pointer(&args[0]))
	}
	C.wrap_ffi_call(&invoker.c.cif, (*[0]byte)(unsafe.Pointer(invoker.c.native_address)),
		unsafe.Pointer(retVal), cArgs, C.int(len(args)), unsafe.Pointer(pOutArgs))
}

// GError to os.Error, frees "err"
func _GErrorToOSError(err *C.GError) (goerr error) {
	goerr = errors.New(_GStringToGoString(err.message))
	C.g_error_free(err)
	return
}

func ToError(ptr unsafe.Pointer) (err error) {
	if ptr == nil {
		return nil
	}
	cErr := (*C.GError)(ptr)
	return _GErrorToOSError(cErr)
}

// Go string to glib C string, "" == NULL
func _GoStringToGString(s string) *C.gchar {
	if s == "" {
		return nil
	}
	return (*C.gchar)(unsafe.Pointer(C.CString(s)))
}

// glib C string to Go string, NULL == ""
func _GStringToGoString(s *C.gchar) string {
	if s == nil {
		return ""
	}
	return C.GoString((*C.char)(unsafe.Pointer(s)))
}

// C string to Go string, NULL == ""
func _CStringToGoString(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}
