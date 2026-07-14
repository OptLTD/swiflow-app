export interface ProviderPreset {
  id: string
  label: string
  api_base: string
  model: string
}

export const PROVIDER_PRESETS: ProviderPreset[] = [
  {
    id: 'deepseek',
    label: 'Deepseek',
    api_base: 'https://api.deepseek.com',
    model: 'deepseek-chat',
  },
  {
    id: 'bigmodel',
    label: 'BigModel',
    api_base: 'https://open.bigmodel.cn/api/paas/v4',
    model: 'glm-4-flash',
  },
  {
    id: 'openai',
    label: 'OpenAI',
    api_base: 'https://api.openai.com/v1',
    model: 'gpt-4o-mini',
  },
  {
    id: 'other',
    label: '其他',
    api_base: '',
    model: '',
  },
]

export function guessPresetId(apiBase: string): string {
  const base = apiBase.toLowerCase()
  if (base.includes('deepseek')) return 'deepseek'
  if (base.includes('bigmodel')) return 'bigmodel'
  if (base.includes('openai.com')) return 'openai'
  return 'other'
}
