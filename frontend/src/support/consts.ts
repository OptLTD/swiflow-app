let base_host = 'localhost:11235'
let base_addr = `http://${base_host}/api`

const webSchema = ['http:', 'https:']
const appSchema = ['wails:', 'tauri:']
const { location, navigator } = globalThis || window || {}
const fromApp = appSchema.includes(location.protocol)
if (!fromApp && webSchema.includes(location.protocol)) {
  const { protocol, host } = location
  base_host = `${location.host}`
  base_addr = `${protocol}//${host}/api`
}

export const BASE_HOST = base_host
export const BASE_ADDR = base_addr
export const isFromApp = () => {
  return fromApp
}
export const getAppTags = () => {
  if (!fromApp) {
    return ''
  }
  const ua = navigator.userAgent
  if (ua.indexOf('Mac OS') > -1) {
    return 'in-macos'
  }
  if (ua.indexOf('Windows') > -1) {
    return 'in-windows'
  }
  return ''
}
export const getWebSocketUrl = () => {
  const isSsl = location.protocol == 'https:'
  const wsProto = isSsl && !fromApp ? 'wss' : 'ws'
  const source = fromApp ? 'browser' : 'client'
  const sessid = localStorage.getItem('sessid') || ''
  const query = `source=${source}&sessid=${sessid}`
  return `${wsProto}://${base_host}/socket?${query}`;
}