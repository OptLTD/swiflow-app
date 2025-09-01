
export const shownProviders = [
  'doubao', 'qwen', 'bigmodel',
  'openai', 'grok', 'claude', 'gemini',
  'openrouter', 'siliconflow', 'openai-like',
]

export const allProviders: Record<string, ModelMeta> = {
  swiflow: {
    provider: 'Swiflow',
    useModel: 'qwen-plus-latest',
    apiUrl: 'https://api.swiflow.cn',
    models: [
      'qwen-plus-latest',
      'doubao-seed-1-6-250615'
    ]
  },
  doubao: {
    provider: 'Doubao',
    useModel: 'doubao-seed-1-6-250615',
    apiUrl: 'https://ark.cn-beijing.volces.com/api/v3'
  },
  qwen: {
    provider: 'Qwen',
    useModel: 'qwen-plus-latest',
    apiUrl:'https://dashscope.aliyuncs.com/compatible-mode/v1',
  },
  bigmodel: {
    provider: 'BigModel',
    useModel: 'glm-4.5',
    apiUrl: 'https://open.bigmodel.cn/api/paas/v4',
  },
  deepseek: {
    provider: 'DeepSeek',
    models: ['v3'],
    useModel: 'v3',
    apiUrl: 'https://api.deepseek.com'
  },
  openai: {
    provider: 'ChatGPT',
    apiUrl: 'https://api.openai.com'
  },
  grok: {
    provider: 'Grok',
    apiUrl: 'https://api.x.ai/v1',
  },
  claude: {
    provider: 'Claude',
    apiUrl: 'https://api.anthropic.com'
  },
  gemini: {
    provider: 'Gemini',
    useModel: 'gemini-2.5-pro',
    apiUrl: 'https://generativelanguage.googleapis.com'
  },
  mistral: {
    provider: 'Mistral',
    apiUrl:  'https://api.mistral.ai'
  },
  ollama: {
    provider: 'Ollama',
    apiUrl: 'http://localhost:11434'
  },
  openrouter: {
    provider: 'OpenRouter',
    apiUrl: 'https://openrouter.ai/api/v1',
    useModel: 'anthropic/claude-3.7-sonnet',
  },
  siliconflow: {
    provider: 'Silicon Flow',
    apiUrl: 'https://api.siliconflow.cn',
    useModel: 'moonshotai/Kimi-K2-Instruct',
  },
  'openai-like': {
    provider: 'OpenAI Compatible',
  },
}

export const getProviders = (): OptMeta[] => {
  const items = shownProviders.map(key => ({
    label: allProviders[key].provider || '',
    value: key
  }))
  const defaultValue = {
    value: '', label: 'Default Provider',
  }
  return [defaultValue].concat(...items) as OptMeta[]
}