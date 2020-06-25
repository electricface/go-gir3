package gi

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"unsafe"
)

var funcNextId uint
var funcMap = make(map[uint]func(interface{}))
var funcMapMu sync.RWMutex

func RegisterFunc(fn func(v interface{})) unsafe.Pointer {
	funcMapMu.Lock()

	id := funcNextId
	funcMap[id] = fn
	funcNextId++

	funcMapMu.Unlock()
	return unsafe.Pointer(uintptr(unsafe.Pointer(nil)) + uintptr(id))
}

func UnregisterFunc(fnId unsafe.Pointer) {
	funcMapMu.Lock()
	delete(funcMap, uint(uintptr(fnId)))
	funcMapMu.Unlock()
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

func (ic *InvokerCache) Get1(id uint, ns, name, fnName string) (Invoker, error) {
	ic.mu.RLock()
	invoker, ok := ic.m[id]
	ic.mu.RUnlock()
	if ok {
		return invoker, nil
	}

	bi := defaultRepo.FindByName(ns, name)
	if bi.P == nil {
		return Invoker{}, fmt.Errorf("not found %q in namespace %v", name, ic.namespace)
	}
	defer bi.Unref()

	type0 := bi.Type()
	var funcInfo FunctionInfo
	switch type0 {
	case INFO_TYPE_FUNCTION:
		funcInfo = WrapFunctionInfo(bi.P)
		// NOTE: 不要再 unref funcInfo 了

	case INFO_TYPE_INTERFACE, INFO_TYPE_OBJECT, INFO_TYPE_STRUCT, INFO_TYPE_UNION:
		var methodInfo FunctionInfo
		switch type0 {
		case INFO_TYPE_INTERFACE:
			ifcInfo := WrapInterfaceInfo(bi.P)
			methodInfo = ifcInfo.FindMethod(fnName)
		case INFO_TYPE_OBJECT:
			objInfo := WrapObjectInfo(bi.P)
			methodInfo = objInfo.FindMethod(fnName)
		case INFO_TYPE_STRUCT:
			si := WrapStructInfo(bi.P)
			methodInfo = si.FindMethod(fnName)
		case INFO_TYPE_UNION:
			ui := WrapUnionInfo(bi.P)
			methodInfo = ui.FindMethod(fnName)
		}
		if methodInfo.P == nil {
			return Invoker{}, fmt.Errorf("not found %q in %s %v in namespace %v",
				fnName, type0, name, ns)
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

func (ic *InvokerCache) Get(id uint, name, fnName string) (Invoker, error) {
	return ic.Get1(id, ic.namespace, name, fnName)
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

type Enum int

type Flags uint

type Long int64

type Ulong uint64

var TypeInt = reflect.TypeOf(0)
var TypeUint = reflect.TypeOf(uint(0))
