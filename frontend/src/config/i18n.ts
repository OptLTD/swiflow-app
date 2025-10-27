import { createI18n } from 'vue-i18n'
import zhCN from '@/locales/zh-CN'
import enUS from '@/locales/en-US'

const getLang = () => {
  const language = (navigator.language || 'en').toLocaleLowerCase()
  const locale = localStorage.getItem('lang') || language.split('-')[0]
  return locale || 'en'
}

export const i18n = createI18n({
  messages: {
    zh: zhCN,
    en: enUS,
  },
  legacy: false,
  locale: getLang(),
  fallbackLocale: 'zh',
  globalInjection: true,
})

// Convenience wrappers usable outside Vue components
export const t = (key: string, ...args: any[]) => {
  // @ts-ignore typing from vue-i18n
  return i18n.global.t(key, ...(args as any)) as string
}
export const te = (key: string) => {
  return i18n.global.te(key)
}