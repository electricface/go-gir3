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
	"fmt"
	"os"
	"sync"
	"unsafe"
)

var funcNextId uint
var funcMap = make(map[uint]func(interface{}))
var funcMapMu sync.RWMutex

func RegisterFunc(fn func(v interface{})) unsafe.Pointer {
	funcMapMu.Lock()
	defer funcMapMu.Unlock()

	id := funcNextId
	funcMap[id] = fn
	funcNextId++
	return unsafe.Pointer(uintptr(unsafe.Pointer(nil)) + uintptr(id))
}

func GetFunc(id uint) func(interface{}) {
	funcMapMu.RLock()
	fn := funcMap[id]
	funcMapMu.RUnlock()
	return fn
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

func DefaultRepository() *Repository {
	return defaultRepo
}

func (ic *InvokerCache) GetGType(id uint, typeName string) GType {
	ic.mu.RLock()
	gType, ok := ic.typeMap[id]
	ic.mu.RUnlock()
	if ok {
		return gType
	}

	bi := defaultRepo.FindByName(ic.namespace, typeName)
	if bi.IsNil() {
		_, _ = fmt.Fprintf(os.Stderr, "not found type %v in namespace %v", typeName, ic.namespace)
		return 0
	}
	defer bi.Unref()

	rti := ToRegisteredTypeInfo(bi)
	gType = rti.GetGType()
	ic.putGType(id, gType)
	return gType
}

func (ic *InvokerCache) Get(id uint, name, fnName string) (Invoker, error) {
	ic.mu.RLock()
	invoker, ok := ic.m[id]
	ic.mu.RUnlock()
	if ok {
		return invoker, nil
	}

	bi := defaultRepo.FindByName(ic.namespace, name)
	if bi.IsNil() {
		return Invoker{}, fmt.Errorf("not found %q in namespace %v", name, ic.namespace)
	}
	defer bi.Unref()

	type0 := bi.Type()
	var funcInfo *FunctionInfo
	switch type0 {
	case INFO_TYPE_FUNCTION:
		funcInfo = ToFunctionInfo(bi)
		// NOTE: 不要再 unref funcInfo 了

	case INFO_TYPE_INTERFACE, INFO_TYPE_OBJECT, INFO_TYPE_STRUCT, INFO_TYPE_UNION:
		var methodInfo *FunctionInfo
		switch type0 {
		case INFO_TYPE_INTERFACE:
			ifcInfo := ToInterfaceInfo(bi)
			methodInfo = ifcInfo.FindMethod(fnName)
		case INFO_TYPE_OBJECT:
			objInfo := ToObjectInfo(bi)
			methodInfo = objInfo.FindMethod(fnName)
		case INFO_TYPE_STRUCT:
			si := ToStructInfo(bi)
			methodInfo = si.FindMethod(fnName)
		case INFO_TYPE_UNION:
			ui := ToUnionInfo(bi)
			methodInfo = ui.FindMethod(fnName)
		}
		if methodInfo == nil {
			return Invoker{}, fmt.Errorf("not found %q in %s %v", fnName, type0, name)
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

func Int2Bool(v int) bool {
	return v != 0
}

func Bool2Int(v bool) int {
	if v {
		return 1
	}
	return 0
}
