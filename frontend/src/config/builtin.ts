import { t } from '@/config/i18n'

// Optional runtime overrides for builtin tool titles and descriptions
const builtinTitleOverrides: Record<string, string> = {}
const builtinDescOverrides: Record<string, string> = {}

export const setTitle = (name: string, title: string) => {
  builtinTitleOverrides[name] = title
}

export const setDesc = (name: string, desc: string) => {
  builtinDescOverrides[name] = desc
}

export const getTitle = (name: string): string | undefined => {
  const tools = ['image-ocr', 'python3', 'command']
  if (tools.includes(name)) {
    const base = name.replace('-', '')
    return t(`builtin.${base}Name`)
  }
  return builtinTitleOverrides[name]
}

export const getDesc = (name: string): string | undefined => {
  // Prefer runtime override if present
  if (builtinDescOverrides[name]) {
    return builtinDescOverrides[name]
  }
  const variants = [
    name.replace(/-/g, ''),
    name.replace(/-/g, '_'),
    name,
  ]
  for (const base of variants) {
    const keyDesc = `builtin.${base}Desc`
    const translated = t(keyDesc)
    if (translated && translated !== keyDesc) {
      return translated
    }
  }
  return undefined
}

// Get both name and description for a tool, with fallbacks
export const getLabel = (name: string, fallbackName?: string, fallbackDesc?: string) => {
  const translatedName = getTitle(name)
  const translatedDesc = getDesc(name)
  
  return {
    name: translatedName || fallbackName || name,
    desc: translatedDesc || fallbackDesc || ''
  }
}