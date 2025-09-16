declare module 'vue3-emoji-picker'

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare type UserInput = {
  content: string;
  uploads?: string[];
}

declare type BotReply = {
  content: string;
}

declare type Annotate = {
  subject: string;
  context: string;
}

declare type Thinking = {
  content: string;
}

declare type Complete = {
  content: string;
}

declare type MakeAsk = {
  checked: number;
  question: string;
  options: string[];
}

declare type DefaultResult = {
  result: string
  errmsg: string
}

declare type DefaultAction = {
  content: string;
}

declare type PathListFiles = {
  path: string;
}

declare type FileGetContent = {
  path: string;
}

declare type FilePutContent = {
  path: string;
  content: string;
}

declare type FileReplaceText = {
  path: string;
  diff: string;
}

declare type ExecuteCommand = {
  command: string;
}

declare type StartAsyncCmd = {
  session: string;
  command: string;
}

declare type UseMcpTool = {
  title?: string;
  args?: Record;
  tool: string;
  server: string;
  more: boolean // 详细展示
}

declare type UseSelfTool = {
  title?: string;
  args?: Record;
  tool: string;
  more: boolean // 详细展示
}

declare type SetSelfTool = {
  uuid?: string;
  title?: string;
  toolName: string;
  code: string
}

declare type DefaultProps = {
  type: string,
  hash: string,
  hide: boolean
  msgid: string,
  checked: number;
}

declare type MsgAct = (
   Annotate | Thinking | UserInput | BotReply
  | ExecuteCommand | DefaultAction | MakeAsk | Complete
  | UseMcpTool | UseSelfTool | SetSelfTool | StartAsyncCmd
  | FileGetContent | FilePutContent | FileReplaceText | PathListFiles
) & DefaultResult & DefaultProps

declare type ActionMsg = {
  context: object
  loading: boolean
  errors: string[]
  actions: MsgAct[]

  theMsgId: string
  thinking: string
  datetime: string
}

declare type SocketMsg = {
  action: string
  method: string
  chatid: string
  detail: any
}

// 新增文件变动消息类型
declare type ChangeMsg = {
  path: string
  type: "file" | "directory"
  operation: string
  timestamp: number
  fullPath: string
}

declare type InputMsg = {
  content: string
  uploads: string[]
  newTask?: string
  homePath?: string
  placeholder: string
}


declare type TaskEntity = {
  uuid: string
  name: string
  state: string
  bots: string[]
  command: string
  process: number
  // robot: string
}

declare type TodoEntity = {
  uuid: string
  time: string
  todo: string
  done: number
  task: string
}

declare type BotEntity = {
  uuid: string
  name: string
  emoji: string
  tools: string[]
  provider: string
  endpoint: string
  apiSecret: string
  modelName: string
  usePrompt: string
  sysPrompt: string
}

declare type MemEntity = {
  id: string
  bot: string
  type: string
  subject: string
  content: string
}

declare type McpServer = {
  uuid: string
  name: string
  type: string
  url: string
  env: Record
  args: string[]
  command: string
  status: McpStatus
  loading: boolean
  tags: string[]
}

declare type McpStatus = {
  errmsg: string
  active: boolean
  enable: boolean
  tools?: Record[]
  checked?: string[]
}


declare type ActionState = {
  state: string
  title: string
  preview: string
}


declare type GlobalResp = {
  useModel: string;
  active: BotEntity
  bots: BotEntity[]
  setup: SetupMeta
  login: LoginMeta
  mcpEnv: McpEnvMeta
  authGate: string
  epigraph?: string
}

declare type LoginMeta = {
  email: string
  avatar: string
  username: string
  userPlan: string
  expireAt: string
}

declare type SetupMeta = {
  theme: string
  version?: string
  sandbox: boolean
  dataPath: string
  language: "en" | "zh"
  notification: boolean
  ctxMsgSize: string
  maxCallTurns: string
  useCopy: string
  useDebug: boolean
  useMulti: boolean
  proxyUrl: string
  authGate: string
  sendNotifyOn: string[]
}

declare type McpEnvMeta = {
  nodejs: string
  python: string
  uvx: string
  npx: string
}

declare type MenuMeta = {
  label: string
  value: string
  other?: object
}

declare type OptMeta = {
  label: string
  value: string
  group: string
  other?: object
  disabled: boolean
}

declare type ModelInfo = {
  haskey: boolean
  status: string
  apiKey: string
  apiUrl: string
  models: string[]
  default: string
  provider: string
}

declare type ModelMeta = {
  apiKey?: string
  apiUrl?: string
  models?: string[]
  provider: string
  useModel?: string
}

declare type ToolsResp = Record<string, string>
declare type ModelResp = Record<string, ModelInfo>


declare type CfgBrowserMeta = {
  engine: string
  ignores: string
}

declare type CfgMySQLMeta = {
  host: string
  user: string
  pass: string
  name: string
  port: string
  deny: string
}
