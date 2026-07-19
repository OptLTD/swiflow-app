import { i18n, t } from '../i18n'

const CODE_RE = /^[a-z][a-z0-9_]*$/

/** Translate a backend error code; leave free-text messages unchanged. */
export function mapApiError(raw: string | undefined | null, httpFallback?: string): string {
  const fallback = httpFallback || t('errors.unknown')
  if (!raw) return fallback
  if (!CODE_RE.test(raw)) return raw
  const key = `errors.${raw}`
  if (i18n.global.te(key)) return String(i18n.global.t(key))
  return t('errors.unknown')
}
