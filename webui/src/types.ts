export interface Agent {
  id: string
  key: string
  display_name?: string
  provider: string
  model: string
  system_extra?: string
}

export interface Provider {
  id: string
  name: string
  display_name?: string
  api_base: string
  enabled: boolean
}

export interface Session {
  id: string
  key: string
  agent_key: string
  title?: string
}

export interface Message {
  id: string
  role: string
  content: string
  thinking?: string
  tool_calls_json?: string
  tool_call_id?: string
  tool_name?: string
  seq?: number
}

export interface ToolInfo {
  name: string
  description: string
  enabled: boolean
}

export interface SkillInfo {
  slug: string
  name: string
  description: string
  source: string
  enabled: boolean
}

export interface MCPCapabilities {
  connected: boolean
  tools: MCPToolCapability[]
  resources: MCPResourceInfo[]
  templates: MCPResourceTemplateInfo[]
}

export interface MCPToolCapability {
  name: string
  mcp_name: string
  description: string
  enabled: boolean
}

export interface MCPResourceInfo {
  uri: string
  name: string
  title?: string
  description?: string
  mime_type?: string
}

export interface MCPResourceTemplateInfo {
  uri_template: string
  name: string
  title?: string
  description?: string
  mime_type?: string
}

export interface MCPServer {
  id: string
  name: string
  display_name?: string
  transport: 'stdio' | 'sse' | 'streamable'
  command?: string
  args?: string[]
  url?: string
  env?: Record<string, string>
  enabled: boolean
}

export interface CronJob {
  id: string
  name: string
  agent_key: string
  message: string
  schedule: string
  enabled: boolean
  last_run_at?: string
}

export interface ChatEvent {
  type: string
  content?: string
  thinking?: string
  id?: string
  name?: string
  arguments?: Record<string, unknown>
  result?: string
  is_error?: boolean
  error?: string
  title?: string
  position?: number
}

export interface SkillDraft {
  id: string
  slug: string
  content: string
  note?: string
  created_at?: string
}

export interface WorkspaceEntry {
  name: string
  path: string
  is_dir: boolean
  size?: number
  mod_time?: string
}
