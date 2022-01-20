package g

/*
#cgo pkg-config: glib-2.0 gobject-2.0
#include <glib.h>
#include <glib-object.h>
#include <stdlib.h>

extern void	goMarshal(GClosure *, GValue *, guint, GValue *, gpointer, GValue *);

static GClosure * _g_closure_new() {
	GClosure	*closure;
	closure = g_closure_new_simple(sizeof(GClosure), NULL);
	g_closure_set_marshal(closure, (GClosureMarshal)(goMarshal));
	return closure;
}

extern void	removeClosure(gpointer, GClosure *);

static void _g_closure_add_finalize_notifier(GClosure *closure) {
	g_closure_add_finalize_notifier(closure, NULL, removeClosure);
}

static gboolean _g_is_value(GValue *val) {
	return G_IS_VALUE(val);
}

static GType _g_value_type(GValue *val) {
	return G_VALUE_TYPE(val);
}

static GObjectClass * _g_object_get_class (GObject *object) {
	return G_OBJECT_GET_CLASS(object);
}

*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
	"time"
	"unsafe"

	"github.com/electricface/go-gir/gi"
)

const (
	TYPE_INVALID   gi.GType = C.G_TYPE_INVALID
	TYPE_NONE      gi.GType = C.G_TYPE_NONE
	TYPE_INTERFACE gi.GType = C.G_TYPE_INTERFACE
	TYPE_CHAR      gi.GType = C.G_TYPE_CHAR
	TYPE_UCHAR     gi.GType = C.G_TYPE_UCHAR
	TYPE_BOOLEAN   gi.GType = C.G_TYPE_BOOLEAN
	TYPE_INT       gi.GType = C.G_TYPE_INT   // int32
	TYPE_UINT      gi.GType = C.G_TYPE_UINT  // uint32
	TYPE_LONG      gi.GType = C.G_TYPE_LONG  // int64
	TYPE_ULONG     gi.GType = C.G_TYPE_ULONG // uint64
	TYPE_INT64     gi.GType = C.G_TYPE_INT64
	TYPE_UINT64    gi.GType = C.G_TYPE_UINT64
	TYPE_ENUM      gi.GType = C.G_TYPE_ENUM
	TYPE_FLAGS     gi.GType = C.G_TYPE_FLAGS
	TYPE_FLOAT     gi.GType = C.G_TYPE_FLOAT
	TYPE_DOUBLE    gi.GType = C.G_TYPE_DOUBLE
	TYPE_STRING    gi.GType = C.G_TYPE_STRING
	TYPE_POINTER   gi.GType = C.G_TYPE_POINTER
	TYPE_BOXED     gi.GType = C.G_TYPE_BOXED
	TYPE_PARAM     gi.GType = C.G_TYPE_PARAM
	TYPE_OBJECT    gi.GType = C.G_TYPE_OBJECT
	TYPE_VARIANT   gi.GType = C.G_TYPE_VARIANT
)

type closureContext struct {
	fn       interface{}
	userData interface{}
}

var (
	_closures = struct {
		sync.RWMutex
		m map[unsafe.Pointer]closureContext
	}{
		m: make(map[unsafe.Pointer]closureContext),
	}

	_signals = make(map[SignalHandle]Closure)
)

// removeClosure removes a closure from the internal _closures map.  This is
// needed to prevent a leak where Go code can access the closure context
// (along with rf and userdata) even after an object has been destroyed and
// the GClosure is invalidated and will never run.
//
//export removeClosure
func removeClosure(_ C.gpointer, closure *C.GClosure) {
	_closures.Lock()
	delete(_closures.m, unsafe.Pointer(closure))
	_closures.Unlock()
}

//export goMarshal
func goMarshal(closure *C.GClosure, retValue *C.GValue,
	nParams C.guint, params *C.GValue,
	invocationHint C.gpointer, marshalData *C.GValue) {

	_ = invocationHint
	_ = marshalData

	// Get the context associated with this callback closure.
	_closures.RLock()
	cc := _closures.m[unsafe.Pointer(closure)]
	_closures.RUnlock()

	var paramsIfc []interface{}
	nGLibParams := int(nParams)
	if nGLibParams > 0 {
		paramsIfc = make([]interface{}, nGLibParams)
		gValues := gValueSlice(params, nGLibParams)
		for i := 0; i < nGLibParams; i++ {
			v := Value{P: unsafe.Pointer(&gValues[i])}
			val, _ := v.Get()
			paramsIfc[i] = val
		}
	}

	defer func() {
		err := recover()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "func panic with error:", err)
			debug.PrintStack()
		}
	}()

	switch fn := cc.fn.(type) {
	case func():
		// 无参数，无返回值
		fn()
	case func() interface{}:
		// 无参数，有返回值
		ret := fn()
		gRetValue := Value{unsafe.Pointer(retValue)}
		err := gRetValue.Set(ret)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cannot save callback return value: %v", err)
		}
	case func(p gi.ParamBox):
		// 有参数，无返回值
		fn(gi.ParamBox{Params: paramsIfc, UserData: cc.userData})
	case func(p gi.ParamBox) interface{}:
		// 有参数，有返回值
		ret := fn(gi.ParamBox{Params: paramsIfc, UserData: cc.userData})
		gRetValue := Value{unsafe.Pointer(retValue)}
		err := gRetValue.Set(ret)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cannot save callback return value: %v", err)
		}

	default:
		_, _ = fmt.Fprintf(os.Stderr, "invalid func type")
	}
}

// gValueSlice converts a C array of GValues to a Go slice.
func gValueSlice(values *C.GValue, nValues int) (slice []C.GValue) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	header.Cap = nValues
	header.Len = nValues
	header.Data = uintptr(unsafe.Pointer(values))
	return
}

func isValidClosureFn(fn interface{}) bool {
	switch fn.(type) {
	case func(),
		func() interface{},
		func(b gi.ParamBox),
		func(b gi.ParamBox) interface{}:
		return true
	default:
		return false
	}
}

// NewClosure creates a new GClosure and adds its callback function
// to the internally-maintained map. It's exported for visibility to other
// gotk3 packages and shouldn't be used in application code.
func NewClosure(fn interface{}, userData ...interface{}) Closure {
	if !isValidClosureFn(fn) {
		panic("fn is not a valid closure function")
	}

	// Create closure context which points to the reflected func.
	cc := closureContext{fn: fn}
	if len(userData) > 0 {
		if len(userData) > 1 {
			panic("len(userData) > 1, len(userData) must be equal to 0 or 1")
		}
		cc.userData = userData[0]
	}

	//fType := rf.Type()
	//for i := 0; i < fType.NumIn(); i++ {
	//argType := fType.In(i)
	//fmt.Printf("i: %d, argType: %v\n", i, argType)
	//if argType.Implements(cgtIfc) {
	//	argEmptyValue := reflect.New(argType)
	//	cctm := argEmptyValue.Interface().(CanGetTypeAndGValueGetter)
	//	registerGValueGetter(cctm.GetType(), cctm.GetGValueGetter())
	//}
	//}

	c := C._g_closure_new()
	C._g_closure_add_finalize_notifier(c)

	// Associate the GClosure with rf.  rf will be looked up in this
	// map by the closure when the closure runs.
	_closures.Lock()
	_closures.m[unsafe.Pointer(c)] = cc
	_closures.Unlock()

	return Closure{unsafe.Pointer(c)}
}

func (v Closure) native() *C.GClosure {
	return (*C.GClosure)(v.P)
}

// 用于信号处理的
type SignalHandle uint

type SourceHandle uint

// IdleAdd adds an idle source to the default main event loop
// context.  After running once, the source func will be removed
// from the main event loop, unless f returns a single bool true.
//
// This function will cause a panic when f eventually runs if the
// types of args do not match those of f.
func IdleAdd(fn SourceFunc) (SourceHandle, error) {

	// Create an idle source func to be added to the main loop context.
	idleSrc := C.g_idle_source_new()
	if idleSrc == nil {
		return 0, nilPtrErr
	}
	return sourceAttach(idleSrc, fn)
}

var nilPtrErr = errors.New("cgo returned unexpected nil pointer")

// TimeoutAdd adds an timeout source to the default main event loop
// context.  After running once, the source func will be removed
// from the main event loop, unless f returns a single bool true.
func TimeoutAdd(interval time.Duration, fn SourceFunc) (SourceHandle, error) {
	// Create a interval source func to be added to the main loop context.
	// interval is in milliseconds
	intervalMs := interval.Milliseconds()
	timeoutSrc := C.g_timeout_source_new(C.guint(intervalMs))
	if timeoutSrc == nil {
		return 0, nilPtrErr
	}

	return sourceAttach(timeoutSrc, fn)
}

func TimeoutAddSeconds(interval uint, fn SourceFunc) (SourceHandle, error) {
	timeoutSrc := C.g_timeout_source_new_seconds(C.guint(interval))
	if timeoutSrc == nil {
		return 0, nilPtrErr
	}
	return sourceAttach(timeoutSrc, fn)
}

// sourceAttach attaches a source to the default main loop context.
func sourceAttach(src *C.struct__GSource, fn SourceFunc) (SourceHandle, error) {
	if src == nil {
		return 0, nilPtrErr
	}

	// Create a new GClosure from f that invalidates itself when
	// f returns false.  The error is ignored here, as this will
	// always be a function.
	f := func() interface{} {
		return fn()
	}
	closure := NewClosure(f)

	// Set closure to run as a callback when the idle source runs.
	C.g_source_set_closure(src, closure.native())

	// Attach the idle source func to the default main event loop
	// context.
	cid := C.g_source_attach(src, nil)
	return SourceHandle(cid), nil
}

// TODO:
//func SourceSetClosure(src glib.Source, closure Closure) {
//	C.g_source_set_closure((*C.GSource)(src.Ptr), closure.native())
//}

/* ---- Value ---- */

func (v Value) native() *C.GValue {
	return (*C.GValue)(v.P)
}

// Type 获取 Value 的类型 id。
// 和 GetGType 方法不一样，GetGType 要求 Value 保存的值是 GType 类型，并且获取保存的值。
func (v Value) Type() gi.GType {
	ret := C._g_value_type(v.native())
	return gi.GType(ret)
}

// IsValid 检查 Value 是否合法和已经初始化了。
func (v Value) IsValid() bool {
	ret := C._g_is_value(v.native())
	return gi.Int2Bool(int(ret))
}

func (v Value) Init(gType gi.GType) Value {
	ret := C.g_value_init(v.native(), C.GType(gType))
	return Value{P: unsafe.Pointer(ret)}
}

func (v Value) Reset() Value {
	ret := C.g_value_reset(v.native())
	return Value{P: unsafe.Pointer(ret)}
}

func (v Value) Unset() {
	C.g_value_unset(v.native())
}

/* ---- Object ---- */

func (v Object) connectClosure(after bool, detailedSignal string, f interface{}, userData ...interface{}) SignalHandle {
	cstr := C.CString(detailedSignal)

	closure := NewClosure(f, userData...)

	c := C.g_signal_connect_closure(C.gpointer(v.P),
		(*C.gchar)(cstr), closure.native(), C.gboolean(gi.Bool2Int(after)))
	C.free(unsafe.Pointer(cstr))

	handle := SignalHandle(c)
	// Map the signal handle to the closure.
	_signals[handle] = closure
	return handle
}

func (v Object) native() *C.GObject {
	return (*C.GObject)(v.P)
}

func (v Object) GetClass() ObjectClass {
	ret := C._g_object_get_class(v.native())
	return ObjectClass{P: unsafe.Pointer(ret)}
}

// GetPropertyType returns the Type of a property of the underlying GObject.
// If the property is missing it will return TYPE_INVALID and an error.
func (v Object) GetPropertyType(name string) (gi.GType, error) {
	cName := C.CString(name)
	paramSpec := C.g_object_class_find_property(C._g_object_get_class(v.native()), (*C.gchar)(cName))
	C.free(unsafe.Pointer(cName))

	if paramSpec == nil {
		return TYPE_INVALID, errors.New("couldn't find Property")
	}
	return gi.GType(paramSpec.value_type), nil
}

// GetProperty is a wrapper around g_object_get_property().
// 需要释放返回的 Value
func (v Object) Get(name string) (Value, error) {
	t, err := v.GetPropertyType(name)
	if err != nil {
		return Value{}, err
	}

	value, err := NewValueT(t)
	if err != nil {
		return Value{}, err
	}
	v.get(name, value)
	return value, nil
}

func (v Object) get(name string, value Value) {
	cName := C.CString(name)
	C.g_object_get_property(v.native(), (*C.gchar)(cName), value.native())
	C.free(unsafe.Pointer(cName))
	return
}

func (v Object) GetProperties(names []string, dest ...interface{}) error {
	if len(names) != len(dest) {
		return errors.New("Object.GetProperties: length mismatch")
	}
	value, err := NewValue()
	if err != nil {
		return err
	}
	defer value.Free()

	for i, name := range names {
		t, err := v.GetPropertyType(name)
		if err != nil {
			return err
		}
		value.Unset()
		value.Init(t)
		v.get(name, value)
		err = value.Store(dest[i])
		if err != nil {
			return err
		}
	}
	return nil
}

/* ---- List ---- */

func NewList() List {
	ret := C.g_list_alloc()
	return wrapList(ret)
}

func (v List) ForEach(fn func(item unsafe.Pointer)) {
	for l := v.p(); l != nil; l = l.next {
		fn(unsafe.Pointer(l.data))
	}
}

// g_list_foreach
func (v List) ForEachC(fn Func) {
	callableInfo := gi.GetCallableInfo("GLib", "Func")
	cId, funcPtr := gi.RegisterFClosure(func(result unsafe.Pointer, args []unsafe.Pointer) {
		CallFunc(fn, result, args)
	}, gi.ScopeCall, callableInfo)

	C.g_list_foreach(v.p(), (*[0]byte)(funcPtr), nil)

	callableInfo.Unref()
	gi.UnregisterFClosure(cId)
}

func (v *List) FullFree(fn func(item unsafe.Pointer)) {
	v.ForEach(fn)
	v.Free()
	v.P = nil
}

func wrapList(p *C.GList) List {
	return List{P: unsafe.Pointer(p)}
}


func (v List) SetPrevious(prev List) {
	native := v.p()
	native.prev = prev.p()
}

// Free 释放所有被 List 使用的内存。如果列表的元素包含动态分配的内存，
// 应该使用 FreeFull 或则首先释放它们。
func (v *List) Free() {
	C.g_list_free(v.p())
	v.P = nil
}

func (v *List) FreeFull(freeFn func(item unsafe.Pointer)) {
	v.ForEach(freeFn)
	v.Free()
}

// Free1 释放一个元素，不会更新与列表中前和后的元素的链接关系，
// 因此不应该在这个元素还是列表的一部分时调用这个函数。
//
// 它通常用在 RemoveLink 之后。
func (v *List) Free1() {
	C.g_list_free_1(v.p())
	v.P = nil
}

func (v List) RemoveLink(lLink List) List {
	ret := C.g_list_remove_link(v.p(), lLink.p())
	return wrapList(ret)
}

func (v List) Length() int {
	return int(C.g_list_length(v.p()))
}

func (v List) NthData(n uint) unsafe.Pointer {
	data := C.g_list_nth_data(v.p(), C.guint(n))
	return unsafe.Pointer(data)
}

func (v List) Nth(n uint) List {
	list := C.g_list_nth(v.p(), C.guint(n))
	return wrapList(list)
}

func (v List) Append(data unsafe.Pointer) List {
	list := C.g_list_append(v.p(), C.gpointer(data))
	return wrapList(list)
}

func (v List) Prepend(data unsafe.Pointer) List {
	list := C.g_list_prepend(v.p(), C.gpointer(data))
	return wrapList(list)
}

func (v List) Insert(data unsafe.Pointer, position int) List {
	list := C.g_list_insert(v.p(), C.gpointer(data), C.gint(position))
	return wrapList(list)
}

func (v List) InsertBefore(sibling List, data unsafe.Pointer) List {
	list := C.g_list_insert_before(v.p(), sibling.p(), C.gpointer(data))
	return wrapList(list)
}

/* ---- SList ---- */

// g_slist_alloc
//
// [ result ] trans: everything
func NewSList() SList {
	ret := C.g_slist_alloc()
	return wrapSList(ret)
}

func wrapSList(p *C.GSList) SList {
	return SList{P: unsafe.Pointer(p)}
}

func (v SList) Append(data unsafe.Pointer) SList {
	list := C.g_slist_append(v.p(), C.gpointer(data))
	return wrapSList(list)
}

func (v SList) Prepend(data unsafe.Pointer) SList {
	list := C.g_slist_prepend(v.p(), C.gpointer(data))
	return wrapSList(list)
}

func (v SList) Insert(data unsafe.Pointer, position int) SList {
	list := C.g_slist_insert(v.p(), C.gpointer(data), C.gint(position))
	return wrapSList(list)
}

func (v SList) InsertBefore(sibling SList, data unsafe.Pointer) SList {
	list := C.g_slist_insert_before(v.p(), sibling.p(), C.gpointer(data))
	return wrapSList(list)
}

func (v SList) Remove(data unsafe.Pointer) SList {
	list := C.g_slist_remove(v.p(), C.gconstpointer(data))
	return wrapSList(list)
}

func (v SList) RemoveLink(link_ SList) SList {
	list := C.g_slist_remove_link(v.p(), link_.p())
	return wrapSList(list)
}

func (v SList) DeleteLink(link_ SList) SList {
	list := C.g_slist_delete_link(v.p(), link_.p())
	return wrapSList(list)
}

func (v SList) RemoveAll(data unsafe.Pointer) SList {
	list := C.g_slist_remove_all(v.p(), C.gconstpointer(data))
	return wrapSList(list)
}

// Free 释放所有被 SList 使用的内存。如果列表的元素包含动态分配的内存，
// 应该使用 FreeFull 或则首先释放它们。
func (v *SList) Free() {
	C.g_slist_free(v.p())
	v.P = nil
}

func (v *SList) FreeFull(freeFn func(item unsafe.Pointer)) {
	v.ForEach(freeFn)
	v.Free()
}

// Free1 释放一个元素，不会更新与列表中前和后的元素的链接关系，
// 因此你不应该在这个元素还是列表的一部分时调用这个函数。
//
// 它通常用在 RemoveLink 之后。
func (v *SList) Free1() {
	C.g_slist_free_1(v.p())
	v.P = nil
}

func (v SList) Length() uint {
	return uint(C.g_slist_length(v.p()))
}

func (v SList) Copy() SList {
	list := C.g_slist_copy(v.p())
	return wrapSList(list)
}

// TODO g_slist_deep_copy

func (v SList) Reverse() SList {
	list := C.g_slist_reverse(v.p())
	return wrapSList(list)
}

// TODO: g_slist_insert_sorted_with_data

// TODO: g_slist_sort

// TODO: g_slist_sort_with_data

func (v SList) Concat(list2 SList) SList {
	list := C.g_slist_concat(v.p(), list2.p())
	return wrapSList(list)
}

func (v SList) ForEach(fn func(item unsafe.Pointer)) {
	for l := v.p(); l != nil; l = l.next {
		fn(unsafe.Pointer(l.data))
	}
}

// g_slist_foreach
func (v SList) ForEachC(fn Func) {
	callableInfo := gi.GetCallableInfo("GLib", "Func")
	cId, funcPtr := gi.RegisterFClosure(func(result unsafe.Pointer, args []unsafe.Pointer) {
		CallFunc(fn, result, args)
	}, gi.ScopeCall, callableInfo)

	C.g_slist_foreach(v.p(), (*[0]byte)(funcPtr), nil)

	callableInfo.Unref()
	gi.UnregisterFClosure(cId)
}

func (v SList) Last() SList {
	list := C.g_slist_last(v.p())
	return wrapSList(list)
}

func (v SList) Nth(n uint) SList {
	list := C.g_slist_nth(v.p(), C.guint(n))
	return wrapSList(list)
}

func (v SList) NthData(n uint) unsafe.Pointer {
	ptr := C.g_slist_nth_data(v.p(), C.guint(n))
	return unsafe.Pointer(ptr)
}

func (v SList) Find(data unsafe.Pointer) SList {
	list := C.g_slist_find(v.p(), C.gconstpointer(data))
	return wrapSList(list)
}

// TODO: g_slist_find_custom

func (v SList) Position(llist SList) int {
	return int(C.g_slist_position(v.p(), llist.p()))
}

func (v SList) Index(data unsafe.Pointer) int {
	return int(C.g_slist_index(v.p(), C.gconstpointer(data)))
}

//type Type uint

//type CanGetTypeAndGValueGetter interface {
//	GetType() Type
//	GetGValueGetter() GValueGetter
//}

//var cgt CanGetTypeAndGValueGetter
//var cgtIfc = reflect.TypeOf(&cgt).Elem()
