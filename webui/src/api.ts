// API client for the Swiflow backend.
import { desktopDownloadWorkspaceFile } from './lib/desktopWorkspace'
import { isDesktop } from './lib/desktop'
import type {
  Agent,
  ChatEvent,
  Message,
  Provider,
  RuntimeInfo,
  RunSnapshot,
  Session,
  SkillInfo,
  SkillDraft,
  ToolInfo,
  MCPServer,
  MCPCapabilities,
  CronJob,
  LightApp,
  WorkspaceEntry,
} from './types'

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch('/api' + path, {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(30_000),
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
  health: () => req<{ status: string; version?: string }>('GET', '/health'),
  getRuntime: () => req<RuntimeInfo>('GET', '/runtime'),
  listAgents: () => req<{ agents: Agent[] }>('GET', '/agents'),
  createAgent: (a: Partial<Agent>) => req<Agent>('POST', '/agents', a),
  updateAgent: (key: string, a: Partial<Agent>) => req<{ status: string }>('PUT', `/agents/${key}`, a),
  listProviders: () => req<{ providers: Provider[] }>('GET', '/providers'),
  createProvider: (p: Partial<Provider> & { api_key?: string }) => req<Provider>('POST', '/providers', p),
  updateProvider: (id: string, p: Record<string, unknown>) => req<{ status: string }>('PUT', `/providers/${id}`, p),
  deleteProvider: (id: string) => req<{ status: string }>('DELETE', `/providers/${id}`),
  listSessions: () => req<{ sessions: Session[] }>('GET', '/sessions'),
  getSession: (key: string) => req<{ session: Session; messages: Message[] }>('GET', `/sessions/${key}`),
  deleteSession: (key: string) => req<{ status: string }>('DELETE', `/sessions/${key}`),
  abortSession: (key: string) => req<{ aborted: boolean }>('POST', `/sessions/${key}/abort`),
  listRuns: () => req<{ runs: RunSnapshot[] }>('GET', '/runs'),
  getRun: (id: string) => req<{ run: RunSnapshot; children: RunSnapshot[] }>('GET', `/runs/${id}`),
  listSessionChildren: (id: string) =>
    req<{ children: { session?: Session; run?: RunSnapshot }[] }>('GET', `/sessions/${id}/children`),
  listTools: () => req<{ tools: ToolInfo[]; exec_enabled: boolean; browser_enabled: boolean }>('GET', '/tools'),
  setTool: (name: string, enabled: boolean) => req<{ status: string }>('PUT', `/tools/${name}`, { enabled }),
  getSearchSettings: () =>
    req<{ provider: string; base_url: string; api_key_set: boolean }>('GET', '/settings/search'),
  putSearchSettings: (body: { provider?: string; api_key?: string; base_url?: string }) =>
    req<{ status: string; provider: string; base_url: string; api_key_set: boolean }>('PUT', '/settings/search', body),
  listSkills: () => req<{ skills: SkillInfo[] }>('GET', '/skills'),
  setSkill: (slug: string, enabled: boolean) => req<{ status: string }>('PUT', `/skills/${slug}`, { enabled }),
  reloadSkills: () => req<{ status: string }>('POST', '/skills/reload'),
  listSkillDrafts: () => req<{ drafts: SkillDraft[] }>('GET', '/skills/drafts'),
  acceptSkillDraft: (id: string) => req<{ status: string }>('POST', `/skills/drafts/${id}/accept`),
  deleteSkillDraft: (id: string) => req<{ status: string }>('DELETE', `/skills/drafts/${id}`),
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
  listLightApps: () => req<{ apps: LightApp[] }>('GET', '/light-apps'),
  createLightApp: (a: Partial<LightApp>) => req<LightApp>('POST', '/light-apps', a),
  deleteLightApp: (id: string) => req<{ status: string }>('DELETE', `/light-apps/${id}`),
  launchLightApp: (id: string) => req<{ url: string; port: number }>('POST', `/light-apps/${id}/launch`),
  stopLightApp: (id: string) => req<{ status: string }>('POST', `/light-apps/${id}/stop`),
  listLightAppEnv: () => req<{ env: Record<string, string> }>('GET', '/light-apps/env'),
  setLightAppEnv: (key: string, value: string) => req<{ status: string }>('POST', '/light-apps/env', { key, value }),
  deleteLightAppEnv: (key: string) => req<{ status: string }>('DELETE', `/light-apps/env/${encodeURIComponent(key)}`),
  listWorkspace: (path = '.') =>
    req<{ path: string; entries: WorkspaceEntry[] }>('GET', `/workspace/list?path=${encodeURIComponent(path)}`),
  readWorkspaceFile: (path: string) =>
    req<{ path: string; content: string; truncated?: boolean }>('GET', `/workspace/read?path=${encodeURIComponent(path)}`),
  downloadWorkspaceFile: (path: string) => downloadWorkspaceFile(path),
  uploadWorkspace: (path: string, files: File[]) => uploadWorkspaceFiles(path, files),
  openURL: (url: string) => req<{ status: string }>('POST', '/open-url', { url }),
  replyWindow: (id: string, result?: string, error?: string) =>
    req<{ ok: boolean }>('POST', '/window/reply', {
      id,
      ...(result !== undefined ? { result } : {}),
      ...(error !== undefined ? { error } : {}),
    }),
}

function decodeBase64Payload(data: { encoding: string; content: string }): ArrayBuffer {
  if (data.encoding !== 'base64' || typeof data.content !== 'string') {
    throw new Error('unexpected download encoding')
  }
  const binary = atob(data.content)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}

async function downloadWorkspaceFile(path: string): Promise<ArrayBuffer> {
  if (isDesktop()) {
    const data = await desktopDownloadWorkspaceFile(path)
    if (data) return decodeBase64Payload(data)
  }
  const data = await req<{ path: string; encoding: string; content: string; size: number }>(
    'POST',
    '/workspace/download',
    { path },
  )
  return decodeBase64Payload(data)
}

async function uploadWorkspaceFiles(
  path: string,
  files: File[],
): Promise<{ path: string; uploaded: { name: string; path: string; size: number }[] }> {
  const fd = new FormData()
  fd.append('path', path)
  for (const file of files) {
    fd.append('files', file, file.name)
  }
  const res = await fetch('/api/workspace/upload', {
    method: 'POST',
    body: fd,
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
  return data as { path: string; uploaded: { name: string; path: string; size: number }[] }
}

export async function chat(
  sessionKey: string,
  message: string,
  agentKey: string,
  onEvent: (ev: ChatEvent) => void,
): Promise<{ queued?: boolean; position?: number }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const res = await fetch(`/api/sessions/${encodeURIComponent(sessionKey)}/chat`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ message, agent: agentKey }),
  })
  if (res.status === 202) {
    const data = (await res.json()) as { queued?: boolean; position?: number }
    onEvent({ type: 'queued', position: data.position })
    return data
  }
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
  return {}
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
  const headers: Record<string, string> = {}
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
