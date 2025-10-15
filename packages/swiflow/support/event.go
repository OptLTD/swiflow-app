package support

import (
	"reflect"
	"runtime"
	"slices"
	"sync"
)

var manager *eventManager

type EventHandler = func(string, any)
type eventManager struct {
	lock sync.RWMutex // 保证并发安全

	listeners map[string][]EventHandler
	// 记录每个事件已注册的函数名，避免重复注册
	registered map[string]map[string]struct{}
}

func Emit(name string, uuid string, data any) {
	if manager == nil {
		return
	}
	manager.Emit(name, uuid, data)
}
func Listen(name string, handle EventHandler) {
	if manager == nil {
		manager = &eventManager{
			listeners:  make(map[string][]EventHandler),
			registered: make(map[string]map[string]struct{}),
		}
	}
	manager.Listen(name, handle)
}

// Once 仅注册一次监听器，如果已存在等价回调则不再追加
func Once(name string, handle EventHandler) {
	if manager == nil {
		manager = &eventManager{
			listeners:  make(map[string][]EventHandler),
			registered: make(map[string]map[string]struct{}),
		}
	}
	manager.Once(name, handle)
}
func Remove(name string, handle EventHandler) {
	if manager == nil {
		manager = &eventManager{
			listeners: make(map[string][]EventHandler),
		}
		return
	}
	manager.Remove(name, handle)
}

// Emit 触发一个事件
func (em *eventManager) Emit(name string, uuid string, data any) {
	em.lock.RLock()
	defer em.lock.RUnlock()

	listeners := em.listeners[name]
	for _, callback := range listeners {
		go callback(uuid, data) // 异步调用回调函数
	}
}

// AddListener 注册一个事件监听器
func (em *eventManager) Listen(name string, handle EventHandler) {
	em.lock.Lock()
	defer em.lock.Unlock()

	em.listeners[name] = append(em.listeners[name], handle)
}

// Once 注册一个事件监听器，如果已注册相同函数则忽略
func (em *eventManager) Once(name string, handle EventHandler) {
	em.lock.Lock()
	defer em.lock.Unlock()

	// 通过函数名去重，适用于方法绑定与闭包
	pc := reflect.ValueOf(handle).Pointer()
	fname := runtime.FuncForPC(pc).Name()
	if em.registered[name] == nil {
		em.registered[name] = make(map[string]struct{})
	}
	if _, ok := em.registered[name][fname]; ok {
		return
	}
	em.listeners[name] = append(em.listeners[name], handle)
	em.registered[name][fname] = struct{}{}
}

// RemoveListener 移除一个事件监听器
func (em *eventManager) Remove(name string, handle EventHandler) {
	em.lock.Lock()
	defer em.lock.Unlock()

	listeners := em.listeners[name]
	pc := reflect.ValueOf(handle).Pointer()
	fname := runtime.FuncForPC(pc).Name()
	for i, listener := range listeners {
		if runtime.FuncForPC(reflect.ValueOf(listener).Pointer()).Name() == fname {
			em.listeners[name] = slices.Delete(listeners, i, i+1)
			delete(em.registered[name], fname)
			break
		}
	}
}
