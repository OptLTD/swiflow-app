
export function useLocalStorage() {

  // 从 localStorage 读取初始值
  const read = (key: string) => {
    return localStorage.getItem(key)
  }

  // 写入 localStorage 并通知
  const write = (key: string, newValue: any) => {
    var oldValue = localStorage.getItem(key)
    localStorage.setItem(key, newValue)
    window.dispatchEvent(new StorageEvent('storage', {
      key, storageArea: localStorage,
      newValue: newValue, oldValue: oldValue,
    }))
  }
  const remove = (key: string) => {
    var oldValue = localStorage.getItem(key)
    window.dispatchEvent(new StorageEvent('storage', {
      key, storageArea: localStorage,
      newValue: null, oldValue: oldValue,
    }))
  }

  return {
    getItem: read,
    setItem: write,
    removeItem: remove,
  }
}