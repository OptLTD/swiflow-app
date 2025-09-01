package support

import (
	"slices"
	"sync"
)

var manager *eventManager

type EventHandler = func(string, any)
type eventManager struct {
	lock sync.RWMutex // 保证并发安全

	listeners map[string][]EventHandler
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
			listeners: make(map[string][]EventHandler),
		}
	}
	manager.Listen(name, handle)
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

// RemoveListener 移除一个事件监听器
func (em *eventManager) Remove(name string, handle EventHandler) {
	em.lock.Lock()
	defer em.lock.Unlock()

	listeners := em.listeners[name]
	for i, listener := range listeners {
		if &listener == &handle { // 比较函数指针
			em.listeners[name] = slices.Delete(listeners, i, i+1)
			break
		}
	}
}
