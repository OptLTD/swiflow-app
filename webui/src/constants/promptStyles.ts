export interface PromptStylePreset {
  id: string
  label: string
  prompt: string
}

/** Quick sys_prompt styles for onboarding / agent settings. */
export const PROMPT_STYLE_PRESETS: PromptStylePreset[] = [
  {
    id: 'concise',
    label: '简洁',
    prompt: '回答简洁直接，结论先行，少客套。必要时用短列表，不写长篇铺垫。',
  },
  {
    id: 'engineer',
    label: '工程师',
    prompt:
      '你是务实的工程师助手。优先给可执行步骤和具体命令/代码；改动要小而明确；说明风险与回滚方式。不确定时先问关键约束。',
  },
  {
    id: 'researcher',
    label: '研究',
    prompt:
      '你偏分析与求证。先澄清问题与假设，再分点推理；区分事实与推测，并标明依据或不确定之处。',
  },
  {
    id: 'teacher',
    label: '导师',
    prompt:
      '用循序渐进的方式讲解。先给直觉与例子，再补细节；必要时用类比。检查对方是否理解后再深入。',
  }
]

export function guessPromptStyleId(prompt: string): string {
  const t = prompt.trim()
  if (!t) return 'none'
  const hit = PROMPT_STYLE_PRESETS.find((p) => p.id !== 'none' && p.prompt === t)
  return hit?.id || 'other'
}
