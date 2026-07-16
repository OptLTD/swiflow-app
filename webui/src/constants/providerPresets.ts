export interface ProviderPreset {
  id: string
  label: string
  api_base: string
  model: string
  /** Suggested model when configuring the vision tab. */
  vision_model?: string
}

export const PROVIDER_PRESETS: ProviderPreset[] = [
  {
    id: 'bigmodel',
    label: 'BigModel',
    api_base: 'https://open.bigmodel.cn/api/paas/v4',
    model: 'glm-4-flash',
    vision_model: 'glm-4v-flash',
  },
  {
    id: 'deepseek',
    label: 'Deepseek',
    api_base: 'https://api.deepseek.com',
    model: 'deepseek-chat',
    vision_model: 'deepseek-chat',
  },
  {
    id: 'openai',
    label: 'OpenAI',
    api_base: 'https://api.openai.com/v1',
    model: 'gpt-4o-mini',
    vision_model: 'gpt-4o',
  },
  {
    id: 'other',
    label: '其他',
    api_base: '',
    model: '',
    vision_model: '',
  },
]

export const DEFAULT_PROVIDER_PRESET_ID = 'bigmodel'

export function defaultProviderPreset(): ProviderPreset {
  return PROVIDER_PRESETS.find((p) => p.id === DEFAULT_PROVIDER_PRESET_ID) || PROVIDER_PRESETS[0]
}

export function guessPresetId(apiBase: string): string {
  const base = apiBase.toLowerCase()
  if (base.includes('bigmodel.cn') || base.includes('bigmodel')) return 'bigmodel'
  if (base.includes('deepseek')) return 'deepseek'
  if (base.includes('openai.com')) return 'openai'
  return 'other'
}
