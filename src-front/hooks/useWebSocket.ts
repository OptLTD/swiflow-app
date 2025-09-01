
let address: string | null = null;
let connect: WebSocket | null = null;
let wstimer: NodeJS.Timeout | null = null;
let handles: Map<string, (payload: any) => void> = new Map();
let reconnectAttempts = 0;
const maxReconnectAttempts = 10; // 最大重连次数
const reconnectIntervals = [1, 3, 5, 8, 30]; // 重连间隔时间（秒）

export const useWebSocket = () => {
  const onOpen = function (this: WebSocket, _e: Event): void {
    console.log('WebSocket connected');
    reconnectAttempts = 0; // 重置重连次数
    // wait auth succ
    setTimeout(() => {
      this.send(
        JSON.stringify({
          method: 'system',
          action: 'hello',
          detail: 'hi,swiflow'
        })
      )
    }, 150)
  };

  const onError = function (this: WebSocket, e: Event): void {
    console.error('WebSocket error:', e);
    reConnect();
  };

  const onClose = function (this: WebSocket, e: CloseEvent): void {
    console.error('WebSocket closed:', e);
    reConnect();
  };

  const onMessage = function (this: WebSocket, e: MessageEvent): void {
    if (!e || !e.data) return;
    try {
      const payload = JSON.parse(e.data);
      if (!payload || !payload.method) {
        console.log('err msg:', payload);
        return
      }
      if (handles.has(payload.method)) {
        const handle = handles.get(payload.method);
        handle?.call(connect, payload)
      }
    } catch (error) {
      console.error('Failed to parse message:', error);
    }
  };

  const reConnect = (): void => {
    if (reconnectAttempts >= maxReconnectAttempts) {
      console.error('Max reconnection attempts reached. Stopping reconnection.');
      return;
    }

    if (wstimer) {
      clearTimeout(wstimer);
      wstimer = null;
    }

    const interval = reconnectIntervals[Math.min(reconnectAttempts, reconnectIntervals.length - 1)];
    reconnectAttempts++;

    wstimer = setTimeout(() => {
      console.log(`Reconnecting... Attempt ${reconnectAttempts}`);
      doConnect(address as string);
    }, interval * 1000);
  };

  const useHandle = (name: string, callable: (payload: any) => void): void => {
    handles.set(name, callable);
  };

  const doConnect = (addr: string): void => {
    if (connect && connect.readyState === WebSocket.OPEN) {
      console.log('WebSocket is already connected.');
      return;
    }

    if (wstimer) {
      clearTimeout(wstimer);
      wstimer = null;
    }

    address = addr;
    connect = new WebSocket(addr);

    connect.onopen = onOpen;
    connect.onerror = onError;
    connect.onclose = onClose;
    connect.onmessage = onMessage;
  };

  const getConnect = (): WebSocket | null => {
    return connect;
  };

  const disconnect = (): void => {
    if (connect) {
      connect.close();
      connect = null;
    }

    if (wstimer) {
      clearTimeout(wstimer);
      wstimer = null;
    }

    reconnectAttempts = 0; // 重置重连次数
  };

  return {
    useHandle,
    doConnect,
    getConnect,
    disconnect,
  };
};
