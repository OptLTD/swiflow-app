// API client for the Mira backend. Auth token is stored in localStorage.
const TOKEN_KEY = 'mira_token'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}
export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t)
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

async function req<T>(method: string, path: string, body?: any): Promise<T> {
  const token = getToken()
  const res = await fetch('/api' + path, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    throw new Error(data?.error || `HTTP ${res.status}`)
  }
  return data as T
}

export const api = {
  health: () => req<{ status: string }>('GET', '/health'),
  listAgents: () => req<{ agents: any[] }>('GET', '/agents'),
  createAgent: (a: any) => req<any>('POST', '/agents', a),
  updateAgent: (key: string, a: any) => req<any>('PUT', `/agents/${key}`, a),
  listProviders: () => req<{ providers: any[] }>('GET', '/providers'),
  createProvider: (p: any) => req<any>('POST', '/providers', p),
  updateProvider: (id: string, p: any) => req<any>('PUT', `/providers/${id}`, p),
  deleteProvider: (id: string) => req<any>('DELETE', `/providers/${id}`),
  listSessions: () => req<{ sessions: any[] }>('GET', '/sessions'),
  getSession: (key: string) => req<{ session: any; messages: any[] }>('GET', `/sessions/${key}`),
  abortSession: (key: string) => req<{ aborted: boolean }>('POST', `/sessions/${key}/abort`),
  listTools: () => req<{ tools: any[] }>('GET', '/tools'),
  setTool: (name: string, enabled: boolean) => req<any>('PUT', `/tools/${name}`, { enabled }),
  listSkills: () => req<{ skills: any[] }>('GET', '/skills'),
  setSkill: (slug: string, enabled: boolean) => req<any>('PUT', `/skills/${slug}`, { enabled }),
  reloadSkills: () => req<any>('POST', '/skills/reload'),
}

// Chat over SSE (POST with streaming response). Calls onEvent for each parsed
// event object. Resolves when the stream ends.
export async function chat(
  sessionKey: string,
  message: string,
  agentKey: string,
  onEvent: (ev: any) => void,
): Promise<void> {
  const token = getToken()
  const res = await fetch(`/api/sessions/${encodeURIComponent(sessionKey)}/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ message, agent_key: agentKey }),
  })
  if (!res.ok) {
    const text = await res.text()
    let msg = `HTTP ${res.status}`
    try { msg = JSON.parse(text).error || msg } catch {}
    throw new Error(msg)
  }
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    let idx
    while ((idx = buf.indexOf('\n\n')) >= 0) {
      const frame = buf.slice(0, idx)
      buf = buf.slice(idx + 2)
      for (const line of frame.split('\n')) {
        if (line.startsWith('data: ')) {
          try {
            onEvent(JSON.parse(line.slice(6)))
          } catch {}
        }
      }
    }
  }
}
