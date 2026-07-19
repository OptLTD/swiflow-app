import { createI18n } from 'vue-i18n'
import en from './locales/en'
import zhCN from './locales/zh-CN'

export const LOCALE_KEY = 'swiflow_locale'
export type AppLocale = 'zh-CN' | 'en'

export const SUPPORTED_LOCALES: { value: AppLocale; label: string }[] = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en', label: 'English' },
]

function detectLocale(): AppLocale {
  try {
    const stored = localStorage.getItem(LOCALE_KEY)
    if (stored === 'zh-CN' || stored === 'en') return stored
  } catch {
    /* ignore */
  }
  try {
    const nav = navigator.language || ''
    if (nav.toLowerCase().startsWith('zh')) return 'zh-CN'
    if (nav) return 'en'
  } catch {
    /* ignore */
  }
  return 'zh-CN'
}

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: {
    'zh-CN': zhCN,
    en,
  },
})

export function getLocale(): AppLocale {
  const loc = i18n.global.locale.value
  return loc === 'en' ? 'en' : 'zh-CN'
}

export function setLocale(locale: AppLocale) {
  i18n.global.locale.value = locale
  try {
    localStorage.setItem(LOCALE_KEY, locale)
  } catch {
    /* ignore */
  }
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale === 'zh-CN' ? 'zh-CN' : 'en'
  }
}

/** Apply detected locale to <html lang> on boot. */
export function applyDocumentLang() {
  setLocale(getLocale())
}

export function t(key: string, params?: Record<string, unknown>): string {
  return String(i18n.global.t(key, params || {}))
}
