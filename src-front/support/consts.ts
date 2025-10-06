let base_host = 'localhost:11235'
let base_addr = `http://${base_host}/api`

// @ts-ignore isTauri not exists in browser window
const { isTauri, location } = globalThis || window || {}
if (!isTauri && location.protocol !== 'tauri:') {
  const { protocol, host } = location
  base_host = `${location.host}`
  base_addr = `${protocol}//${host}/api`
}

export const BASE_HOST = base_host
export const BASE_ADDR = base_addr
export const getWebSocketUrl = () => {
  const isSsl = location.protocol == 'https:'
  const wsProto = isSsl && !isTauri ? 'wss' : 'ws'
  const source = isTauri ? 'browser' : 'client'
  const sessid = localStorage.getItem('sessid') || ''
  const query = `source=${source}&sessid=${sessid}`
  return `${wsProto}://${base_host}/socket?${query}`;
}