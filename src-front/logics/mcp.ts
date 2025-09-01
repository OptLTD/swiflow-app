import { request } from '@/support'

export function extractServers(json: any) {
  const servers = [] as McpServer[]
  const config = json.mcpServers || json.servers || json.mcp || json 
  if (!config) {
    throw new Error('未找到有效的 servers 配置字段')
  }
  for (var uuid in config) {
    servers.push({ ...config[uuid], uuid })
  }
  return servers[0]
}

export const checkNetEnv = async (): Promise<string> => {
  const url = `/toolenv?act=net-env`
  try {
    const resp = await request.post(url)
    return (resp as any)?.result || ''
  } catch (err) {
    return ''
  }
}

export const checkMcpEnv = async (callback?: (info: any) => void) => {
  request.get('/toolenv?act=mcp-env').then((resp: any) => {
    if (resp['running']) {
      setTimeout(() => checkMcpEnv(callback), 1500)
      return
    }
    callback && callback(resp)
  }).catch((err) => {
    if (callback) callback({error: err})
  })
}

export const doInstall = async (netEnv: string, name: string, callback?: (ok: boolean) => void) => {
  try {
    const url = `/toolenv?act=install&name=${name}&net-env=${netEnv}`
    const resp = await request.post(url, {})
    console.log('init py & uv', resp)
  } catch (err) {
    console.error('init py & uv', err)
  } finally {
    setTimeout(() => checkMcpEnv(callback), 1500)
  }
}

export const mcpTestNew = async (json: any) => {
  const url = `/mcp?act=test-mcp`
  return await request.post<any>(url, json)
}

export const mcpTestSet = async (data: any, uuid: string) => {
  const url = `/mcp?act=test-mcp&uuid=${uuid}`
  return await request.post<any>(url, data)
}

export const mcpSaveNew = async (json: any) => {
  const url = `/mcp?act=set-new`
  return await request.post<any>(url, json)
}

export const mcpSaveSet = async (data: any, uuid: string) => {
  const url = `/mcp?act=set-mcp&uuid=${uuid}`
  return await request.post<any>(url, data)
}

export const mcpDelete = async (data: any, uuid: string) => {
  const url = `/mcp?act=del-mcp&uuid=${uuid}`
  return await request.post<any>(url, data)
}


export const getMcpList = async () => {
  const url = `/mcp?act=get-mcp`
  return await request.post<any>(url)
}

export const doActiveMcp = async (uuid: string, callback: Function) => {
  try {
    const url = `/mcp?act=active&uuid=${uuid}`
    const resp = await request.post<any>(url)
    if (resp?.errmsg) {
      throw resp?.errmsg
    }
    callback && callback(resp as McpStatus)
  } catch (err) {
    throw err
  }
}