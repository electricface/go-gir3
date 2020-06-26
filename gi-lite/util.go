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

type Enum int

type Flags uint

type Long int64

type Ulong uint64

var TypeInt = reflect.TypeOf(0)
var TypeUint = reflect.TypeOf(uint(0))
