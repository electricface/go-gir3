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

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"unsafe"
)

type Closure struct {
	Fn    func(interface{})
	Scope Scope
}

type Scope uint

const (
	ScopeInvalid Scope = iota
	ScopeCall
	ScopeAsync
	ScopeNotified
)

var _funcNextId uint
var _funcMap = make(map[uint]Closure)
var _funcMapMu sync.RWMutex

func RegisterFunc(fn func(v interface{}), scope Scope) uint {
	_funcMapMu.Lock()

	id := _funcNextId
	_funcMap[id] = Closure{
		Fn:    fn,
		Scope: scope,
	}
	_funcNextId++

	_funcMapMu.Unlock()
	return id
}

func UnregisterFunc(id uint) {
	_funcMapMu.Lock()
	delete(_funcMap, id)
	_funcMapMu.Unlock()
}

func GetFunc(id uint) Closure {
	_funcMapMu.RLock()
	c := _funcMap[id]
	_funcMapMu.RUnlock()
	return c
}

type InvokerCache struct {
	namespace string
	mu        sync.RWMutex
	m         map[uint]Invoker
	typeMap   map[uint]GType
}

func NewInvokerCache(ns string) *InvokerCache {
	return &InvokerCache{
		namespace: ns,
		m:         make(map[uint]Invoker),
		typeMap:   make(map[uint]GType),
	}
}

func (ic *InvokerCache) put(id uint, invoker Invoker) {
	ic.mu.Lock()
	ic.m[id] = invoker
	ic.mu.Unlock()
}

func (ic *InvokerCache) putGType(id uint, gType GType) {
	ic.mu.Lock()
	ic.typeMap[id] = gType
	ic.mu.Unlock()
}

var defaultRepo = getDefaultRepository()

func DefaultRepository() Repository {
	return defaultRepo
}

func (ic *InvokerCache) GetGType1(id uint, ns, typeName string) GType {
	ic.mu.RLock()
	gType, ok := ic.typeMap[id]
	ic.mu.RUnlock()
	if ok {
		return gType
	}

	bi := defaultRepo.FindByName(ns, typeName)
	if bi.P == nil {
		_, _ = fmt.Fprintf(os.Stderr, "not found type %v in namespace %v", typeName, ic.namespace)
		return 0
	}
	defer bi.Unref()

	rti := WrapRegisteredTypeInfo(bi.P)
	gType = rti.GetGType()
	ic.putGType(id, gType)
	return gType
}

func (ic *InvokerCache) GetGType(id uint, typeName string) GType {
	return ic.GetGType1(id, ic.namespace, typeName)
}

// 需要 unref 返回值
func findInfoLv1(ns, name string, index int, infoType InfoType) BaseInfo {
	if index >= 0 {
		info := defaultRepo.Info(ns, index)
		if info.P != nil {
			if name == info.Name() && infoType == info.Type() {
				return info
			}
		}
	}
	return defaultRepo.FindByName(ns, name)
}

// 需要 unref 返回值
func findMethodInfo(info infoWithMethod, name string, index int, flags FindMethodFlags) (fi FunctionInfo) {
	if index >= 0 {
		fi = info.Method(index)
		if fi.P != nil {
			if name == fi.Name() {
				return
			}
		}
	}
	if flags&FindMethodNoCallFind == 0 {
		fi = info.FindMethod(name)
		if fi.P != nil {
			return
		}
	}
	numMethods := info.NumMethods()
	for i := 0; i < numMethods; i++ {
		method := info.Method(i)
		if method.Name() == name {
			return method
		}
		method.Unref()
	}

	return
}

type FindMethodFlags uint

const (
	FindMethodNoCallFind FindMethodFlags = 1 << iota // 不要调用 FindMethod 方法
)

type infoWithMethod interface {
	FindMethod(name string) FunctionInfo
	Method(index int) FunctionInfo
	NumMethods() int
}

func (ic *InvokerCache) Get1(id uint, ns, nameLv1, nameLv2 string, idxLv1, idxLv2 int, infoType InfoType, flags FindMethodFlags) (Invoker, error) {
	ic.mu.RLock()
	invoker, ok := ic.m[id]
	ic.mu.RUnlock()
	if ok {
		return invoker, nil
	}

	bi := findInfoLv1(ns, nameLv1, idxLv1, infoType)
	if bi.P == nil {
		return Invoker{}, fmt.Errorf("not found %q in namespace %v", nameLv1, ic.namespace)
	}
	defer bi.Unref()

	type0 := bi.Type()
	var funcInfo FunctionInfo
	switch type0 {
	case INFO_TYPE_FUNCTION:
		funcInfo = WrapFunctionInfo(bi.P)
		// NOTE: 不要再 unref funcInfo 了, 因为所有权在 bi。

	case INFO_TYPE_INTERFACE, INFO_TYPE_OBJECT, INFO_TYPE_STRUCT, INFO_TYPE_UNION:
		var infoM infoWithMethod
		switch type0 {
		case INFO_TYPE_INTERFACE:
			infoM = WrapInterfaceInfo(bi.P)
		case INFO_TYPE_OBJECT:
			infoM = WrapObjectInfo(bi.P)
		case INFO_TYPE_STRUCT:
			infoM = WrapStructInfo(bi.P)
		case INFO_TYPE_UNION:
			infoM = WrapUnionInfo(bi.P)
		}
		methodInfo := findMethodInfo(infoM, nameLv2, idxLv2, flags)
		if methodInfo.P == nil {
			return Invoker{}, fmt.Errorf("not found %q in %s %v in namespace %v",
				nameLv2, type0, nameLv1, ns)
		}
		defer methodInfo.Unref()
		funcInfo = methodInfo

	default:
		// TODO: support more type
		return Invoker{}, fmt.Errorf("unsupported info type %s", bi.Type())
	}

	invoker, err := funcInfo.PrepInvoker()
	if err != nil {
		return Invoker{}, err
	}
	ic.put(id, invoker)
	return invoker, nil
}

func (ic *InvokerCache) Get(id uint, nameLv1, nameLv2 string, idxLv1, idxLv2 int, infoType InfoType, flags FindMethodFlags) (Invoker, error) {
	return ic.Get1(id, ic.namespace, nameLv1, nameLv2, idxLv1, idxLv2, infoType, flags)
}

func Int2Bool(v int) bool {
	return v != 0
}

func Bool2Int(v bool) int {
	if v {
		return 1
	}
	return 0
}

func Uint2Ptr(n uint) unsafe.Pointer {
	return unsafe.Pointer(uintptr(unsafe.Pointer(nil)) + uintptr(n))
}

type Enum int

type Flags uint

type Long int64

type Ulong uint64

var TypeInt = reflect.TypeOf(0)
var TypeUint = reflect.TypeOf(uint(0))

func Store(src []interface{}, dest ...interface{}) error {
	if len(src) != len(dest) {
		return errors.New("gi.Store: length mismatch")
	}

	for i := range src {
		if err := storeInterfaces(src[i], dest[i]); err != nil {
			return err
		}
	}
	return nil
}

func StoreStruct(src []interface{}, dest interface{}) error {
	destRv := reflect.ValueOf(dest)
	if destRv.Kind() == reflect.Ptr {
		elem := destRv.Elem()
		if elem.Kind() == reflect.Struct {
			num := elem.NumField()
			if len(src) != num {
				return errors.New("gi.StoreStruct: length mismatch")
			}
			for i := range src {
				if err := store(reflect.ValueOf(src[i]), elem.Field(i)); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return errors.New("gi.StoreStruct: dest is not ptr")
}

func storeInterfaces(src, dest interface{}) error {
	return store(reflect.ValueOf(src), reflect.ValueOf(dest))
}

func store(src, dest reflect.Value) error {
	if dest.Kind() == reflect.Ptr {
		return store(src, dest.Elem())
	}
	return storeBase(src, dest)
}

func storeBase(src, dest reflect.Value) error {
	destType := dest.Type()

	if src.Type().ConvertibleTo(destType) {
		dest.Set(src.Convert(destType))
		return nil
	}

	if src.Kind() == reflect.UnsafePointer {
		ok := storeStructFieldP(dest, unsafe.Pointer(src.Pointer()))
		if ok {
			return nil
		}
	} else if src.Kind() == reflect.Struct {
		p := src.FieldByName("P")
		if p.Kind() == reflect.UnsafePointer {
			// src 是有 P unsafe.Pointer 字段的结构体，比如 g.Object
			ok := storeStructFieldP(dest, unsafe.Pointer(p.Pointer()))
			if ok {
				return nil
			}
		}
	}

	return fmt.Errorf("gi.Store: type mismatch: cannot covert %s to %s", src.Type(), dest.Type())
}

func storeStructFieldP(dest reflect.Value, ptr unsafe.Pointer) bool {
	if dest.Kind() == reflect.Struct {
		p := dest.FieldByName("P")
		if p.Kind() == reflect.UnsafePointer {
			p.SetPointer(ptr)
			return true
		}
	}
	return false
}
