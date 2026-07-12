// API client for the Mira backend. Auth token is stored in localStorage.
import type {
  Agent,
  ChatEvent,
  Message,
  Provider,
  Session,
  SkillInfo,
  ToolInfo,
  MCPServer,
  MCPCapabilities,
  CronJob,
} from './types'

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

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
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
  let data: { error?: string } | null = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
    }
  }
  if (!res.ok) {
    throw new Error(data?.error || `HTTP ${res.status}`)
  }
  return data as T
}

export const api = {
  health: () => req<{ status: string }>('GET', '/health'),
  listAgents: () => req<{ agents: Agent[] }>('GET', '/agents'),
  createAgent: (a: Partial<Agent>) => req<Agent>('POST', '/agents', a),
  updateAgent: (key: string, a: Partial<Agent>) => req<{ status: string }>('PUT', `/agents/${key}`, a),
  listProviders: () => req<{ providers: Provider[] }>('GET', '/providers'),
  createProvider: (p: Partial<Provider> & { api_key?: string }) => req<Provider>('POST', '/providers', p),
  updateProvider: (id: string, p: Record<string, unknown>) => req<{ status: string }>('PUT', `/providers/${id}`, p),
  deleteProvider: (id: string) => req<{ status: string }>('DELETE', `/providers/${id}`),
  listSessions: () => req<{ sessions: Session[] }>('GET', '/sessions'),
  getSession: (key: string) => req<{ session: Session; messages: Message[] }>('GET', `/sessions/${key}`),
  abortSession: (key: string) => req<{ aborted: boolean }>('POST', `/sessions/${key}/abort`),
  listTools: () => req<{ tools: ToolInfo[]; exec_enabled: boolean; browser_enabled: boolean }>('GET', '/tools'),
  setTool: (name: string, enabled: boolean) => req<{ status: string }>('PUT', `/tools/${name}`, { enabled }),
  listSkills: () => req<{ skills: SkillInfo[] }>('GET', '/skills'),
  setSkill: (slug: string, enabled: boolean) => req<{ status: string }>('PUT', `/skills/${slug}`, { enabled }),
  reloadSkills: () => req<{ status: string }>('POST', '/skills/reload'),
  listMCPServers: () => req<{ servers: MCPServer[] }>('GET', '/mcp/servers'),
  getMCPCapabilities: (id: string) => req<MCPCapabilities>('GET', `/mcp/servers/${id}/capabilities`),
  createMCPServer: (s: Partial<MCPServer>) => req<MCPServer>('POST', '/mcp/servers', s),
  updateMCPServer: (id: string, s: Record<string, unknown>) => req<{ status: string }>('PUT', `/mcp/servers/${id}`, s),
  deleteMCPServer: (id: string) => req<{ status: string }>('DELETE', `/mcp/servers/${id}`),
  reloadMCP: () => req<{ status: string }>('POST', '/mcp/reload'),
  listCronJobs: () => req<{ jobs: CronJob[] }>('GET', '/cron/jobs'),
  createCronJob: (j: Partial<CronJob>) => req<CronJob>('POST', '/cron/jobs', j),
  updateCronJob: (id: string, j: Record<string, unknown>) => req<{ status: string }>('PUT', `/cron/jobs/${id}`, j),
  deleteCronJob: (id: string) => req<{ status: string }>('DELETE', `/cron/jobs/${id}`),
  reloadCron: () => req<{ status: string }>('POST', '/cron/reload'),
}

export async function chat(
  sessionKey: string,
  message: string,
  agentKey: string,
  onEvent: (ev: ChatEvent) => void,
): Promise<void> {
  await streamSSE(`/api/sessions/${encodeURIComponent(sessionKey)}/chat`, {
    method: 'POST',
    body: JSON.stringify({ message, agent_key: agentKey }),
  }, onEvent)
}

/** Subscribe to background runs (e.g. schedule_run) for a session. */
export async function watchSession(
  sessionKey: string,
  onEvent: (ev: ChatEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  await streamSSE(`/api/sessions/${encodeURIComponent(sessionKey)}/watch`, { method: 'GET', signal }, onEvent)
}

async function streamSSE(
  path: string,
  init: RequestInit,
  onEvent: (ev: ChatEvent) => void,
): Promise<void> {
  const token = getToken()
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  if (init.body) headers['Content-Type'] = 'application/json'
  const res = await fetch(path, {
    ...init,
    headers: { ...headers, ...(init.headers as Record<string, string> | undefined) },
  })
  if (!res.ok) {
    const text = await res.text()
    let msg = `HTTP ${res.status}`
    try {
      msg = JSON.parse(text).error || msg
    } catch {}
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
