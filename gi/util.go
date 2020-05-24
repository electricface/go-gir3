package gi

import (
	"fmt"
	"sync"
)

type InvokerCache struct {
	namespace string
	mu sync.RWMutex
	m map[uint]Invoker
}

func NewInvokerCache(ns string) *InvokerCache {
	return &InvokerCache{
		namespace:  ns,
		m: make(map[uint]Invoker),
	}
}

func (ic *InvokerCache) put(id uint, invoker Invoker) {
	ic.mu.Lock()
	ic.m[id] = invoker
	ic.mu.Unlock()
}

var defaultRepo = getDefaultRepository()

func DefaultRepository() *Repository {
	return defaultRepo
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

	switch bi.Type() {
	case INFO_TYPE_FUNCTION:
		funcInfo := ToFunctionInfo(bi)
		invoker, err := funcInfo.PrepInvoker()
		if err != nil {
			return Invoker{}, err
		}
		ic.put(id, invoker)
		return invoker, nil

	case INFO_TYPE_INTERFACE:
		ifcInfo := ToInterfaceInfo(bi)
		methodInfo := ifcInfo.FindMethod(fnName)

		if methodInfo.IsNil() {
			return Invoker{}, fmt.Errorf("not found %q in interface %v", fnName, name)
		}
		defer methodInfo.Unref()

		invoker, err := methodInfo.PrepInvoker()
		if err != nil {
			return Invoker{}, err
		}
		ic.put(id, invoker)
		return invoker, nil

	case INFO_TYPE_OBJECT:
		objInfo := ToObjectInfo(bi)
		methodInfo := objInfo.FindMethod(fnName)

		if methodInfo.IsNil() {
			return Invoker{}, fmt.Errorf("not found %q in interface %v", fnName, name)
		}
		defer methodInfo.Unref()

		invoker, err := methodInfo.PrepInvoker()
		if err != nil {
			return Invoker{}, err
		}
		ic.put(id, invoker)
		return invoker, nil

	default:
		// TODO: support more type
		return Invoker{}, fmt.Errorf("unsupported info type %s", bi.Type())
	}
}
